package orchestration

import (
	"context"
	"errors"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
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

func TestSessionAgentRunnerRejectsResumeLikeTurnWithoutPendingHumanAnswer(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	sess := session.NewSession("base system prompt")
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	runner := newPreCreateSessionRunner(&preCreateAssertCompleter{
		t:            t,
		sessionStore: sessionStore,
		systemPrompt: "base system prompt",
	}, sessionStore, "base system prompt")

	_, _, err := runner.RunTurn(context.Background(), "   ", sess.ID, "trace-empty-resume")
	if err == nil || !strings.Contains(err.Error(), "resume requires pending human answers") {
		t.Fatalf("expected resume guard error, got %v", err)
	}
}

func TestSessionAgentRunnerRunTurnInputPersistsUserImages(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "done"},
			FinishReason: llm.FinishStop,
		}},
	}
	runner := NewSessionAgentRunner(proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg:          bridgeconfig.Config{MaxTurns: 3, PromptsPath: "", Provider: bridgeconfig.ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"}},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "base system prompt",
		},
	}, nil, sessionStore, nil)

	input := llm.Message{Role: llm.RoleUser, Text: "describe this", Content: []llm.ContentPart{{Type: llm.ContentTypeImage, Image: &llm.ImageContent{URL: "data:image/png;base64,ZmFrZS1pbWFnZQ==", MimeType: "image/png"}}}}
	if _, sessionID, err := runner.RunTurnInput(context.Background(), input, "", "trace-image-input"); err != nil {
		t.Fatalf("run turn input: %v", err)
	} else if len(completer.requests) != 1 || len(completer.requests[0].Messages) < 2 {
		t.Fatalf("unexpected completer requests: %+v", completer.requests)
	} else if got := completer.requests[0].Messages[1]; got.Role != llm.RoleUser || len(got.Content) != 1 || got.Content[0].Image == nil {
		t.Fatalf("expected user image content in model request, got %+v", got)
	} else if loaded, err := sessionStore.Load(sessionID); err != nil {
		t.Fatalf("load session: %v", err)
	} else if len(loaded.Messages) < 2 || len(loaded.Messages[1].Content) != 1 || loaded.Messages[1].Content[0].Image == nil {
		t.Fatalf("expected persisted user image content, got %+v", loaded.Messages)
	} else if loaded.Messages[1].Content[0].Image.URL != "data:image/png;base64,ZmFrZS1pbWFnZQ==" {
		t.Fatalf("unexpected persisted image url: %q", loaded.Messages[1].Content[0].Image.URL)
	}
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
	if loaded.TurnDraft != nil {
		t.Fatalf("expected turn_draft to be cleared after error, got %+v", loaded.TurnDraft)
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
	if loaded.TurnDraft != nil {
		t.Fatalf("expected turn_draft to be cleared after cancellation, got %+v", loaded.TurnDraft)
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
	sess.TurnDraft = &session.TurnDraft{
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
	if loaded.TurnDraft != nil {
		t.Fatalf("expected turn_draft to be cleared, got %+v", loaded.TurnDraft)
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

const (
	titleRequestWaitTimeout = 2 * time.Second
	titleTestMaxTurns       = 3
)

func TestTitleFromFirstMessageUsesFirstSentenceAndTruncates(t *testing.T) {
	got := titleFromFirstMessage("   Build a session title. Then continue with details.\nnext paragraph")
	if got != "Build a session title." {
		t.Fatalf("unexpected title: %q", got)
	}

	longTitle := titleFromFirstMessage("abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz")
	if len([]rune(longTitle)) != sessionTitleRuneLimit || !strings.HasSuffix(longTitle, "...") {
		t.Fatalf("unexpected truncated title: %q", longTitle)
	}
}

func TestSessionAgentRunnerFirstMessageTitleOnlyForNewSession(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	runner := newTitleSessionRunner(newTitleTestCompleter(nil), sessionStore, bridgeconfig.SessionTitleModeFirstMessage)

	_, sessionID, err := runner.RunTurn(context.Background(), "Plan the release. Include checks.", "", "trace-title")
	if err != nil {
		t.Fatalf("run turn: %v", err)
	}
	loaded, err := sessionStore.Load(sessionID)
	if err != nil {
		t.Fatalf("load new session: %v", err)
	}
	if loaded.Title != "Plan the release." {
		t.Fatalf("unexpected new session title: %q", loaded.Title)
	}

	if _, _, err := runner.RunTurn(context.Background(), "Rename attempt", sessionID, "trace-title-2"); err != nil {
		t.Fatalf("run existing turn: %v", err)
	}
	loaded, err = sessionStore.Load(sessionID)
	if err != nil {
		t.Fatalf("reload session: %v", err)
	}
	if loaded.Title != "Plan the release." {
		t.Fatalf("existing session title changed: %q", loaded.Title)
	}
}

func TestSessionAgentRunnerAITitleFailureKeepsEmptyTitle(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	completer := newTitleTestCompleter(errors.New("title failed"))
	runner := newTitleSessionRunner(completer, sessionStore, bridgeconfig.SessionTitleModeAIGenerated)

	_, sessionID, err := runner.RunTurn(context.Background(), "Generate background title", "", "trace-ai-title")
	if err != nil {
		t.Fatalf("run turn: %v", err)
	}
	completer.waitForTitleRequest(t)
	loaded, err := sessionStore.Load(sessionID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if loaded.Title != "" {
		t.Fatalf("expected empty title after failure, got %q", loaded.Title)
	}
}

type titleTestCompleter struct {
	titleErr error
	done     chan struct{}
	once     sync.Once
}

func newTitleTestCompleter(titleErr error) *titleTestCompleter {
	return &titleTestCompleter{titleErr: titleErr, done: make(chan struct{})}
}

func (c *titleTestCompleter) Complete(
	_ context.Context,
	request llm.CompletionRequest,
) (*llm.CompletionResponse, error) {
	if isTitleCompletionRequest(request) {
		c.once.Do(func() { close(c.done) })
		if c.titleErr != nil {
			return nil, c.titleErr
		}
		return &llm.CompletionResponse{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "Generated Title"},
			FinishReason: llm.FinishStop,
		}, nil
	}
	return &llm.CompletionResponse{
		Message:      llm.Message{Role: llm.RoleAssistant, Text: "done"},
		FinishReason: llm.FinishStop,
	}, nil
}

func (c *titleTestCompleter) waitForTitleRequest(t *testing.T) {
	t.Helper()
	select {
	case <-c.done:
	case <-time.After(titleRequestWaitTimeout):
		t.Fatal("timed out waiting for title request")
	}
}

func isTitleCompletionRequest(request llm.CompletionRequest) bool {
	return len(request.Messages) > 0 && strings.Contains(request.Messages[0].Text, "会话标题生成器")
}

func newTitleSessionRunner(
	completer llm.Completer,
	sessionStore *session.Store,
	mode string,
) *SessionAgentRunner {
	return NewSessionAgentRunner(proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg: bridgeconfig.Config{
				MaxTurns:         titleTestMaxTurns,
				SessionTitleMode: mode,
				Provider: bridgeconfig.ProviderConfig{
					Type:  llm.ProviderOpenAI,
					Model: "gpt-4o",
				},
			},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "base system prompt",
		},
	}, nil, sessionStore, nil)
}
