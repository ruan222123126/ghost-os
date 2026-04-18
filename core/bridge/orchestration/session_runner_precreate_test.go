package orchestration

import (
	"context"
	"errors"
	"strings"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
)

const preCreateForcedCompletionError = "forced completion failure"

type preCreateAssertCompleter struct {
	t            *testing.T
	sessionStore *session.Store
	systemPrompt string
	calls        int
}

func (c *preCreateAssertCompleter) Complete(
	_ context.Context,
	_ llm.CompletionRequest,
) (*llm.CompletionResponse, error) {
	c.calls++
	c.assertSessionExistsBeforeLoop()
	return nil, errors.New(preCreateForcedCompletionError)
}

func (c *preCreateAssertCompleter) assertSessionExistsBeforeLoop() {
	c.t.Helper()

	metadata, err := c.sessionStore.ListMetadata()
	if err != nil {
		c.t.Fatalf("list metadata before completion: %v", err)
	}
	if len(metadata) != 1 {
		c.t.Fatalf("expected pre-created session before completion, got %d", len(metadata))
	}

	sess, err := c.sessionStore.Load(metadata[0].ID)
	if err != nil {
		c.t.Fatalf("load pre-created session before completion: %v", err)
	}
	if sess.MessageCount != 1 || len(sess.Messages) != 1 {
		c.t.Fatalf("expected pre-created session shell with only system prompt, got %+v", sess.Messages)
	}
	if sess.Messages[0].Role != llm.RoleSystem || sess.Messages[0].Text != c.systemPrompt {
		c.t.Fatalf("unexpected pre-created system prompt: %+v", sess.Messages[0])
	}
}

func TestSessionAgentRunnerPreCreatesSessionBeforeLoopRunTurn(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	const systemPrompt = "base system prompt"
	completer := &preCreateAssertCompleter{
		t:            t,
		sessionStore: sessionStore,
		systemPrompt: systemPrompt,
	}
	runner := newPreCreateSessionRunner(completer, sessionStore, systemPrompt)

	_, sessionID, err := runner.RunTurn(context.Background(), "hello precreate", "", "trace-precreate")
	if err == nil || !strings.Contains(err.Error(), preCreateForcedCompletionError) {
		t.Fatalf("expected forced completion error, got: %v", err)
	}
	if completer.calls != 1 {
		t.Fatalf("expected one completion call, got %d", completer.calls)
	}
	assertPreCreatedSessionShellAfterFailure(t, sessionStore, sessionID, systemPrompt)
}

func TestSessionAgentRunnerPreCreatesSessionBeforeLoopRunTurnStream(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	const systemPrompt = "base system prompt"
	completer := &preCreateAssertCompleter{
		t:            t,
		sessionStore: sessionStore,
		systemPrompt: systemPrompt,
	}
	runner := newPreCreateSessionRunner(completer, sessionStore, systemPrompt)

	_, sessionID, err := runner.RunTurnStream(
		context.Background(),
		"hello precreate stream",
		"",
		"trace-precreate-stream",
		streaming.NopSink{},
	)
	if err == nil || !strings.Contains(err.Error(), preCreateForcedCompletionError) {
		t.Fatalf("expected forced completion error, got: %v", err)
	}
	if completer.calls != 1 {
		t.Fatalf("expected one completion call, got %d", completer.calls)
	}
	assertPreCreatedSessionShellAfterFailure(t, sessionStore, sessionID, systemPrompt)
}

func newPreCreateSessionRunner(
	completer llm.Completer,
	sessionStore *session.Store,
	systemPrompt string,
) *SessionAgentRunner {
	return NewSessionAgentRunner(proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg: bridgeconfig.Config{
				MaxTurns:    3,
				PromptsPath: "",
				Provider:    bridgeconfig.ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"},
			},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: systemPrompt,
		},
	}, nil, sessionStore, nil)
}

func assertPreCreatedSessionShellAfterFailure(
	t *testing.T,
	sessionStore *session.Store,
	sessionID string,
	systemPrompt string,
) {
	t.Helper()

	resolvedSessionID := strings.TrimSpace(sessionID)
	if resolvedSessionID == "" {
		metadata, err := sessionStore.ListMetadata()
		if err != nil {
			t.Fatalf("list metadata after failure: %v", err)
		}
		if len(metadata) != 1 {
			t.Fatalf("expected one persisted session after failure, got %d", len(metadata))
		}
		resolvedSessionID = strings.TrimSpace(metadata[0].ID)
	}
	loaded, err := sessionStore.Load(resolvedSessionID)
	if err != nil {
		t.Fatalf("load persisted session: %v", err)
	}
	if loaded.MessageCount != 1 || len(loaded.Messages) != 1 {
		t.Fatalf("expected only pre-created session shell after failure, got %+v", loaded.Messages)
	}
	if loaded.Messages[0].Role != llm.RoleSystem || loaded.Messages[0].Text != systemPrompt {
		t.Fatalf("unexpected persisted system prompt: %+v", loaded.Messages[0])
	}
}
