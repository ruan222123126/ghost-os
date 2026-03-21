package app

import (
	"context"
	"strings"
	"testing"

	"ghost-os/bridge/execution"
)

type stubPingClient struct {
	call func(context.Context, string, map[string]any, string) (map[string]any, error)
}

func (c stubPingClient) Call(
	ctx context.Context,
	action string,
	params map[string]any,
	traceID string,
) (map[string]any, error) {
	return c.call(ctx, action, params, traceID)
}

func TestPingRunnerPassesCallerContextAndTraceID(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var gotCtx context.Context
	var gotTraceID string
	closed := false
	runner := pingRunner{
		newClient: func() execution.Client {
			return stubPingClient{
				call: func(callCtx context.Context, action string, params map[string]any, traceID string) (map[string]any, error) {
					gotCtx = callCtx
					gotTraceID = traceID
					if action != "PING" {
						t.Fatalf("unexpected action: %q", action)
					}
					if len(params) != 0 {
						t.Fatalf("expected empty params, got %#v", params)
					}
					if callCtx != ctx {
						t.Fatal("expected runner to pass caller context through to execution client")
					}
					if err := callCtx.Err(); err != context.Canceled {
						t.Fatalf("expected canceled context, got %v", err)
					}
					return map[string]any{"message": "pong"}, nil
				},
			}
		},
		closeClient: func(execution.Client) error {
			closed = true
			return nil
		},
		nextTraceID: func() string {
			return "trace-ping-test"
		},
	}

	message, err := runner.run(ctx)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if message != "pong" {
		t.Fatalf("unexpected message: %q", message)
	}
	if gotCtx != ctx {
		t.Fatal("expected captured context to match caller context")
	}
	if gotTraceID != "trace-ping-test" {
		t.Fatalf("unexpected trace id: %q", gotTraceID)
	}
	if !closed {
		t.Fatal("expected execution client to be closed")
	}
}

func TestNextPingTraceIDGeneratesDistinctValues(t *testing.T) {
	first := nextPingTraceID()
	second := nextPingTraceID()

	if first == second {
		t.Fatal("expected distinct trace ids for each ping invocation")
	}
	if !strings.HasPrefix(first, "bridge-ping-") {
		t.Fatalf("unexpected trace id prefix: %q", first)
	}
	if !strings.HasPrefix(second, "bridge-ping-") {
		t.Fatalf("unexpected trace id prefix: %q", second)
	}
}
