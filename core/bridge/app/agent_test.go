package app

import (
	"context"
	"errors"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
)

type stubConfigStore struct{}

func (*stubConfigStore) Config() (bridgeconfig.Config, error) { return bridgeconfig.Config{}, nil }
func (*stubConfigStore) Snapshot() bridgeconfig.Snapshot      { return bridgeconfig.Snapshot{} }
func (*stubConfigStore) ListProviders() ([]bridgeconfig.ProviderRecord, error) {
	return nil, nil
}
func (*stubConfigStore) AddProvider(bridgeconfig.ProviderRecord) error { return nil }
func (*stubConfigStore) UpdateProvider(string, bridgeconfig.ProviderRecord) error {
	return nil
}
func (*stubConfigStore) DeleteProvider(string) error    { return nil }
func (*stubConfigStore) SetActiveProvider(string) error { return nil }
func (*stubConfigStore) Update(bridgeconfig.UpdateRequest) error {
	return nil
}
func (*stubConfigStore) SetProjectRoot(string) error { return nil }

type recordingTurnRunner struct {
	ctx       context.Context
	message   string
	sessionID string
	traceID   string
	response  string
	err       error
}

func (r *recordingTurnRunner) RunTurn(
	ctx context.Context,
	message string,
	sessionID string,
	traceID string,
) (string, string, error) {
	r.ctx = ctx
	r.message = message
	r.sessionID = sessionID
	r.traceID = traceID
	return r.response, "ignored-session", r.err
}

func TestRunAgentWithConfigStoreDelegatesToSessionTurnRunner(t *testing.T) {
	original := newAgentTurnRunner
	t.Cleanup(func() {
		newAgentTurnRunner = original
	})

	store := bridgeconfig.Store(&stubConfigStore{})
	runner := &recordingTurnRunner{response: "ok"}
	newAgentTurnRunner = func(got bridgeconfig.Store) sessionTurnRunner {
		if got != store {
			t.Fatalf("expected config store %v, got %v", store, got)
		}
		return runner
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	output, err := runAgentWithConfigStore(ctx, "hello ghost", store, "trace-app")
	if err != nil {
		t.Fatalf("runAgentWithConfigStore returned error: %v", err)
	}
	if output != "ok" {
		t.Fatalf("unexpected output %q", output)
	}
	if runner.ctx != ctx {
		t.Fatal("expected runner to receive caller context")
	}
	if runner.message != "hello ghost" {
		t.Fatalf("unexpected message %q", runner.message)
	}
	if runner.sessionID != "" {
		t.Fatalf("expected one-shot agent run to pass empty session_id, got %q", runner.sessionID)
	}
	if runner.traceID != "trace-app" {
		t.Fatalf("unexpected trace_id %q", runner.traceID)
	}
}

func TestRunAgentWithConfigStoreReturnsRunnerError(t *testing.T) {
	original := newAgentTurnRunner
	t.Cleanup(func() {
		newAgentTurnRunner = original
	})

	wantErr := errors.New("runner failed")
	newAgentTurnRunner = func(bridgeconfig.Store) sessionTurnRunner {
		return &recordingTurnRunner{err: wantErr}
	}

	if _, err := runAgentWithConfigStore(context.Background(), "hello", nil, "trace-app"); !errors.Is(err, wantErr) {
		t.Fatalf("expected runner error %v, got %v", wantErr, err)
	}
}
