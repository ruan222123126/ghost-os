package orchestration

import (
	"context"
	"errors"
	"os"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
)

type draftStreamingCompleter struct {
	deltas   []llm.LLMDelta
	response *llm.CompletionResponse
	runErr   error
}

func (c *draftStreamingCompleter) Complete(_ context.Context, _ llm.CompletionRequest) (*llm.CompletionResponse, error) {
	if c.runErr != nil {
		return nil, c.runErr
	}
	if c.response == nil {
		return nil, errors.New("response is nil")
	}
	return c.response, nil
}

func (c *draftStreamingCompleter) CompleteStream(
	ctx context.Context,
	_ llm.CompletionRequest,
	sink llm.LLMStreamSink,
) (*llm.CompletionResponse, error) {
	for _, delta := range c.deltas {
		if err := sink.OnDelta(ctx, delta); err != nil {
			return nil, err
		}
	}
	if c.runErr != nil {
		return nil, c.runErr
	}
	if c.response == nil {
		return nil, errors.New("response is nil")
	}
	return c.response, nil
}

func TestRunTurnStreamInputPersistsAssistantDraftOnError(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	sess := newPersistedSessionForDraftTests(t, sessionStore, "session-stream-draft")

	completer := &draftStreamingCompleter{
		deltas: []llm.LLMDelta{
			{Kind: llm.DeltaKindText, Text: "partial "},
			{Kind: llm.DeltaKindText, Text: "answer"},
		},
		runErr: errors.New("stream interrupted"),
	}
	runner := newDraftTestRunner(sessionStore, completer)

	_, returnedSessionID, err := runner.RunTurnStreamInput(
		context.Background(),
		llm.Message{Role: llm.RoleUser, Text: "continue"},
		sess.ID,
		"trace-stream-draft",
		streaming.NopSink{},
	)
	if err == nil {
		t.Fatal("expected stream run to fail")
	}
	if returnedSessionID != sess.ID {
		t.Fatalf("expected persisted session id on failed turn, got %q want %q", returnedSessionID, sess.ID)
	}

	loaded, loadErr := sessionStore.Load(sess.ID)
	if loadErr != nil {
		t.Fatalf("load session: %v", loadErr)
	}
	if len(loaded.Messages) != 2 {
		t.Fatalf("expected system + user messages, got %+v", loaded.Messages)
	}
	if loaded.Messages[1].Role != llm.RoleUser || loaded.Messages[1].Text != "continue" {
		t.Fatalf("unexpected persisted user message: %+v", loaded.Messages[1])
	}
	if loaded.AssistantDraft == nil {
		t.Fatal("expected assistant draft to persist on stream error")
	}
	if loaded.AssistantDraft.Text != "partial answer" {
		t.Fatalf("unexpected assistant draft: %q", loaded.AssistantDraft.Text)
	}
}

func TestRunTurnStreamInputPersistsAssistantDraftOnCancel(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	sess := newPersistedSessionForDraftTests(t, sessionStore, "session-stream-cancel")

	completer := &draftStreamingCompleter{
		deltas: []llm.LLMDelta{
			{Kind: llm.DeltaKindText, Text: "partial "},
			{Kind: llm.DeltaKindText, Text: "answer"},
		},
		runErr: context.Canceled,
	}
	runner := newDraftTestRunner(sessionStore, completer)

	_, returnedSessionID, err := runner.RunTurnStreamInput(
		context.Background(),
		llm.Message{Role: llm.RoleUser, Text: "continue"},
		sess.ID,
		"trace-stream-cancel",
		streaming.NopSink{},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if returnedSessionID != sess.ID {
		t.Fatalf("expected persisted session id on cancellation, got %q want %q", returnedSessionID, sess.ID)
	}

	loaded, loadErr := sessionStore.Load(sess.ID)
	if loadErr != nil {
		t.Fatalf("load session: %v", loadErr)
	}
	if len(loaded.Messages) != 2 {
		t.Fatalf("expected system + user messages, got %+v", loaded.Messages)
	}
	if loaded.Messages[1].Role != llm.RoleUser || loaded.Messages[1].Text != "continue" {
		t.Fatalf("unexpected persisted user message: %+v", loaded.Messages[1])
	}
	if loaded.AssistantDraft == nil {
		t.Fatal("expected assistant draft to persist on cancellation")
	}
	if loaded.AssistantDraft.Text != "partial answer" {
		t.Fatalf("unexpected assistant draft: %q", loaded.AssistantDraft.Text)
	}
}

func TestRunTurnInputClearsAssistantDraftOnSuccess(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	sess := newPersistedSessionForDraftTests(t, sessionStore, "session-success-clear-draft")
	sess.AssistantDraft = &session.AssistantDraft{
		Text:    "stale draft",
		TraceID: "trace-old",
		Turn:    1,
	}
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save stale draft session: %v", err)
	}

	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "done"},
			FinishReason: llm.FinishStop,
		}},
	}
	runner := newDraftTestRunner(sessionStore, completer)

	if _, _, err := runner.RunTurnInput(
		context.Background(),
		llm.Message{Role: llm.RoleUser, Text: "hello"},
		sess.ID,
		"trace-clear-draft",
	); err != nil {
		t.Fatalf("run turn input: %v", err)
	}

	loaded, loadErr := sessionStore.Load(sess.ID)
	if loadErr != nil {
		t.Fatalf("load session: %v", loadErr)
	}
	if loaded.AssistantDraft != nil {
		t.Fatalf("expected assistant draft to be cleared, got %+v", loaded.AssistantDraft)
	}
}

func newDraftTestRunner(sessionStore *session.Store, completer llm.Completer) *SessionAgentRunner {
	return NewSessionAgentRunner(proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg: bridgeconfig.Config{
				MaxTurns:    3,
				PromptsPath: "",
				PromptsDir:  os.Getenv("GHOST_PROMPTS_DIR"),
				Provider:    bridgeconfig.ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"},
			},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "base system prompt",
		},
	}, nil, sessionStore, nil)
}

func newPersistedSessionForDraftTests(t *testing.T, store *session.Store, sessionID string) *session.Session {
	t.Helper()
	sess := session.NewSession("base system prompt")
	sess.ID = sessionID
	if err := store.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}
	return sess
}
