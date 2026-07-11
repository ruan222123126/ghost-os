package app

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRunServeReturnsDispatchError(t *testing.T) {
	logs := captureStartupLogs(t)
	withRunDispatcher(t, commandDispatcher{
		runPing: noCallPing(t),
		runServe: func(context.Context, int) (string, error) {
			return "", errors.New("listen failed")
		},
		runAgent: noCallAgent(t),
	})

	_, err := Run(context.Background(), []string{"serve"})
	if err == nil {
		t.Fatal("expected serve startup error")
	}
	if strings.Contains(err.Error(), "serve dispatch failed") {
		t.Fatalf("dispatch error should not add extra run wrapper, got %v", err)
	}
	if !strings.Contains(err.Error(), "serve command failed") {
		t.Fatalf("expected dispatch-stage serve wrapper, got %v", err)
	}
	if !strings.Contains(err.Error(), "listen failed") {
		t.Fatalf("expected wrapped cause, got %v", err)
	}
	assertLogContains(t, logs.String(), "startup checkpoint stage=dispatch status=error")
}

func TestRunServeUsageErrorPassesThrough(t *testing.T) {
	logs := captureStartupLogs(t)
	withRunDispatcher(t, commandDispatcher{
		runPing:  noCallPing(t),
		runServe: noCallServe(t),
		runAgent: noCallAgent(t),
	})

	_, err := Run(context.Background(), []string{"serve", "8080", "extra"})
	if err == nil {
		t.Fatal("expected usage error")
	}

	var usageErr usageError
	if !errors.As(err, &usageErr) {
		t.Fatalf("expected usageError, got %T: %v", err, err)
	}
	if strings.Contains(err.Error(), "serve dispatch failed") {
		t.Fatalf("usage error should not be wrapped, got %v", err)
	}
	assertLogContains(t, logs.String(), "startup checkpoint stage=dispatch status=begin")
	if strings.Contains(logs.String(), "status=error") {
		t.Fatalf("usage error path should not log startup error, got %q", logs.String())
	}
}
