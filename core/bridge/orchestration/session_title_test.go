package orchestration

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

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
