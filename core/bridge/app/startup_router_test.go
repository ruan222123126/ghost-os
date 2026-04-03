package app

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"
)

func TestRunServeStartupRouterReady(t *testing.T) {
	logs := captureStartupLogs(t)
	withRunDispatcher(t, commandDispatcher{
		runPing: noCallPing(t),
		runServe: func(_ context.Context, port int) (string, error) {
			if port != defaultServePort {
				t.Fatalf("expected default serve port %d, got %d", defaultServePort, port)
			}
			return "ready", nil
		},
		runAgent: noCallAgent(t),
	})

	output, err := Run(context.Background(), []string{"serve"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if output != "ready" {
		t.Fatalf("unexpected output %q", output)
	}

	assertLogContains(t, logs.String(), "startup checkpoint stage=dispatch status=begin")
	assertLogContains(t, logs.String(), "startup checkpoint stage=dispatch status=ready")
	if strings.Contains(logs.String(), "status=error") {
		t.Fatalf("serve ready path should not log error status, got %q", logs.String())
	}
}

func TestRunNonServeBypassesStartupCheckpoint(t *testing.T) {
	logs := captureStartupLogs(t)
	withRunDispatcher(t, commandDispatcher{
		runPing: func(_ context.Context) (string, error) {
			return "pong", nil
		},
		runServe: noCallServe(t),
		runAgent: noCallAgent(t),
	})

	output, err := Run(context.Background(), []string{"ping"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if output != "pong" {
		t.Fatalf("unexpected output %q", output)
	}

	if strings.Contains(logs.String(), "startup checkpoint stage=dispatch") {
		t.Fatalf("non-serve path should not log startup checkpoints, got %q", logs.String())
	}
}

func withRunDispatcher(t *testing.T, dispatcher commandDispatcher) {
	t.Helper()
	original := newRunCommandDispatcher
	newRunCommandDispatcher = func() commandDispatcher {
		return dispatcher
	}
	t.Cleanup(func() {
		newRunCommandDispatcher = original
	})
}

func captureStartupLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	previousWriter := log.Writer()
	previousFlags := log.Flags()
	previousPrefix := log.Prefix()
	log.SetOutput(buf)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() {
		log.SetOutput(previousWriter)
		log.SetFlags(previousFlags)
		log.SetPrefix(previousPrefix)
	})
	return buf
}

func noCallPing(t *testing.T) func(context.Context) (string, error) {
	t.Helper()
	return func(context.Context) (string, error) {
		t.Fatal("runPing should not be called")
		return "", nil
	}
}

func noCallServe(t *testing.T) func(context.Context, int) (string, error) {
	t.Helper()
	return func(context.Context, int) (string, error) {
		t.Fatal("runServe should not be called")
		return "", nil
	}
}

func noCallAgent(t *testing.T) func(context.Context, string) (string, error) {
	t.Helper()
	return func(context.Context, string) (string, error) {
		t.Fatal("runAgent should not be called")
		return "", nil
	}
}

func assertLogContains(t *testing.T, logs string, want string) {
	t.Helper()
	if !strings.Contains(logs, want) {
		t.Fatalf("missing log %q in %q", want, logs)
	}
}
