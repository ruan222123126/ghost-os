package app

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestDispatchRejectsInvalidCLIInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantDetail string
		wantUsage  string
	}{
		{name: "missing subcommand", args: nil, wantDetail: "missing subcommand", wantUsage: cliUsage},
		{name: "unknown subcommand", args: []string{"foo"}, wantDetail: `unknown subcommand "foo"`, wantUsage: cliUsage},
		{name: "ping extra args", args: []string{"ping", "now"}, wantDetail: "ping does not accept arguments", wantUsage: cliUsage},
		{name: "serve extra args", args: []string{"serve", "8080", "extra"}, wantDetail: "serve accepts at most one port argument", wantUsage: cliUsage},
		{name: "agent missing message", args: []string{"agent"}, wantDetail: "agent requires a non-empty message", wantUsage: cliUsage},
		{name: "agent blank message", args: []string{"agent", "   "}, wantDetail: "agent requires a non-empty message", wantUsage: cliUsage},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := newFailOnCallDispatcher(t).dispatch(context.Background(), tt.args)
			assertUsageError(t, err, tt.wantDetail, tt.wantUsage)
		})
	}
}

func TestDispatchRejectsInvalidServePort(t *testing.T) {
	t.Parallel()

	_, err := newFailOnCallDispatcher(t).dispatch(context.Background(), []string{"serve", "abc"})
	if err == nil {
		t.Fatal("expected invalid port error")
	}
	if strings.Contains(err.Error(), cliUsage) {
		t.Fatalf("invalid port should not be rewritten as usage error: %v", err)
	}
	if !strings.Contains(err.Error(), `invalid port "abc", expected 1-65535`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDispatchRoutesServeDefaultPort(t *testing.T) {
	t.Parallel()

	dispatcher := commandDispatcher{
		runPing: failPing(t),
		runServe: func(_ context.Context, port int) (string, error) {
			if port != defaultServePort {
				t.Fatalf("expected default port %d, got %d", defaultServePort, port)
			}
			return "serving", nil
		},
		runAgent: failAgent(t),
	}

	output, err := dispatcher.dispatch(context.Background(), []string{"serve"})
	if err != nil {
		t.Fatalf("dispatch returned error: %v", err)
	}
	if output != "serving" {
		t.Fatalf("unexpected output %q", output)
	}
}

func TestDispatchRoutesPingWithCallerContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dispatcher := commandDispatcher{
		runPing: func(callCtx context.Context) (string, error) {
			if callCtx != ctx {
				t.Fatal("expected ping dispatcher to pass caller context")
			}
			if err := callCtx.Err(); err != context.Canceled {
				t.Fatalf("expected canceled context, got %v", err)
			}
			return "pong", nil
		},
		runServe: failServe(t),
		runAgent: failAgent(t),
	}

	output, err := dispatcher.dispatch(ctx, []string{"ping"})
	if err != nil {
		t.Fatalf("dispatch returned error: %v", err)
	}
	if output != "pong" {
		t.Fatalf("unexpected output %q", output)
	}
}

func TestDispatchRoutesAgentMessage(t *testing.T) {
	t.Parallel()

	dispatcher := commandDispatcher{
		runPing:  failPing(t),
		runServe: failServe(t),
		runAgent: func(_ context.Context, message string) (string, error) {
			if message != "hello ghost" {
				t.Fatalf("unexpected agent message %q", message)
			}
			return "ok", nil
		},
	}

	output, err := dispatcher.dispatch(context.Background(), []string{"agent", "hello", "ghost"})
	if err != nil {
		t.Fatalf("dispatch returned error: %v", err)
	}
	if output != "ok" {
		t.Fatalf("unexpected output %q", output)
	}
}

func newFailOnCallDispatcher(t *testing.T) commandDispatcher {
	t.Helper()

	return commandDispatcher{
		runPing:  failPing(t),
		runServe: failServe(t),
		runAgent: failAgent(t),
	}
}

func failPing(t *testing.T) func(context.Context) (string, error) {
	t.Helper()
	return func(context.Context) (string, error) {
		t.Fatal("runPing should not be called")
		return "", nil
	}
}

func failServe(t *testing.T) func(context.Context, int) (string, error) {
	t.Helper()
	return func(context.Context, int) (string, error) {
		t.Fatal("runServe should not be called")
		return "", nil
	}
}

func failAgent(t *testing.T) func(context.Context, string) (string, error) {
	t.Helper()
	return func(context.Context, string) (string, error) {
		t.Fatal("runAgent should not be called")
		return "", nil
	}
}

func assertUsageError(t *testing.T, err error, wantDetail string, wantUsage string) {
	t.Helper()

	if err == nil {
		t.Fatal("expected usage error")
	}

	var usageErr usageError
	if !errors.As(err, &usageErr) {
		t.Fatalf("expected usageError, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), wantDetail) {
		t.Fatalf("expected detail %q in error %q", wantDetail, err.Error())
	}
	if !strings.Contains(err.Error(), wantUsage) {
		t.Fatalf("expected usage %q in error %q", wantUsage, err.Error())
	}
}
