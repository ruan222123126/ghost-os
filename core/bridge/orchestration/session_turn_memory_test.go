package orchestration

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/memoryaug"
	"ghost-os/bridge/memorystore"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

type fixedRecallService struct {
	items []memoryaug.RecallItem
}

func (s fixedRecallService) Recall(context.Context, memoryaug.RecallInput) ([]memoryaug.RecallItem, error) {
	return s.items, nil
}

func (s fixedRecallService) LearnFromTurn(context.Context, memoryaug.LearnFromTurnInput) error {
	return nil
}

type capturingRecallService struct {
	fixedRecallService
	lastInput memoryaug.RecallInput
}

func (s *capturingRecallService) Recall(_ context.Context, input memoryaug.RecallInput) ([]memoryaug.RecallItem, error) {
	s.lastInput = input
	return s.items, nil
}

func TestSessionRunnerInjectsRecallOnlyIntoPrompt(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "noted"},
			FinishReason: llm.FinishStop,
		}},
	}
	runner := NewSessionAgentRunner(proTestRuntimeFactory{
		deps: buildMemoryTestDeps(completer, fixedRecallService{
			items: []memoryaug.RecallItem{{
				Entry: memorystore.MemoryEntry{
					ID:         "explicit:user://lang",
					ScopeType:  memorystore.ScopeTypeUser,
					ScopeID:    memorystore.DefaultUserScopeID,
					SourceKind: memorystore.SourceKindExplicit,
					MemoryType: memorystore.MemoryTypePreference,
					Summary:    "reply in Chinese",
					Confidence: 1,
					Status:     memorystore.MemoryStatusActive,
				},
			}},
		}),
	}, nil, sessionStore, nil)

	_, sessionID, err := runner.RunTurn(context.Background(), "What language should you use?", "", "trace-memory")
	if err != nil {
		t.Fatalf("run turn: %v", err)
	}
	if len(completer.requests) != 1 {
		t.Fatalf("expected one completion request, got %d", len(completer.requests))
	}
	if !strings.Contains(completer.requests[0].Messages[0].Text, "Memory context:") {
		t.Fatalf("expected memory block in system prompt, got %q", completer.requests[0].Messages[0].Text)
	}
	loaded, err := sessionStore.Load(sessionID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	for _, message := range loaded.Messages {
		if strings.Contains(message.Text, "Memory context:") {
			t.Fatalf("memory context should not persist into session history: %+v", loaded.Messages)
		}
	}
}

func TestSessionRunnerRecallDoesNotBreakAskHumanContinuation(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	sess := session.NewSession("base system prompt")
	createdAt := time.Date(2026, 3, 14, 10, 0, 0, 0, time.UTC)
	sess.AddPendingQuestion("q-1", session.PendingHumanQuestion{
		Prompt:     "Ship now?",
		ToolCallID: "call-ask",
		TraceID:    "trace-ask",
		CreatedAt:  createdAt,
	})
	if !sess.SetHumanAnswer("q-1", "yes") {
		t.Fatal("expected human answer to be recorded")
	}
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "continuing"},
			FinishReason: llm.FinishStop,
		}},
	}
	runner := NewSessionAgentRunner(proTestRuntimeFactory{
		deps: buildMemoryTestDeps(completer, fixedRecallService{
			items: []memoryaug.RecallItem{{
				Entry: memorystore.MemoryEntry{
					ID:         "mem-1",
					ScopeType:  memorystore.ScopeTypeSession,
					ScopeID:    sess.ID,
					SourceKind: memorystore.SourceKindLearned,
					MemoryType: memorystore.MemoryTypeWorkflow,
					Summary:    "waiting for ship approval",
					Confidence: 0.9,
					Status:     memorystore.MemoryStatusActive,
				},
			}},
		}),
	}, nil, sessionStore, nil)

	_, _, err := runner.RunTurn(context.Background(), "continue", sess.ID, "trace-resume")
	if err != nil {
		t.Fatalf("run resumed turn: %v", err)
	}
	request := completer.requests[0]
	if !containsToolMessage(request.Messages, "call-ask") {
		t.Fatalf("expected ask_human tool message to remain in resumed history, got %+v", request.Messages)
	}
	if !strings.Contains(request.Messages[0].Text, "Memory context:") {
		t.Fatalf("expected memory context in system prompt, got %q", request.Messages[0].Text)
	}
}

func TestSessionRunnerBuildsRecallQueryFromRecentContext(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	sess := session.NewSession("base system prompt")
	sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: "Please reply in Chinese."})
	sess.AddMessage(llm.Message{Role: llm.RoleAssistant, Text: "I will reply in Chinese."})
	sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: "We are updating the Android runtime settings screen."})
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "continuing"},
			FinishReason: llm.FinishStop,
		}},
	}
	recall := &capturingRecallService{}
	runner := NewSessionAgentRunner(proTestRuntimeFactory{
		deps: buildMemoryTestDeps(completer, recall),
	}, nil, sessionStore, nil)

	if _, _, err := runner.RunTurn(context.Background(), "continue with the runtime change", sess.ID, "trace-query"); err != nil {
		t.Fatalf("run turn: %v", err)
	}
	if !strings.Contains(recall.lastInput.Query, "Please reply in Chinese.") {
		t.Fatalf("expected recent user context in recall query, got %q", recall.lastInput.Query)
	}
	if !strings.Contains(recall.lastInput.Query, "We are updating the Android runtime settings screen.") {
		t.Fatalf("expected latest session context in recall query, got %q", recall.lastInput.Query)
	}
	if !strings.Contains(recall.lastInput.Query, "continue with the runtime change") {
		t.Fatalf("expected current user message in recall query, got %q", recall.lastInput.Query)
	}
}

type fixedMemoryService interface {
	memoryaug.RecallService
	memoryaug.LearningService
}

func buildMemoryTestDeps(completer *proTestCompleter, recall fixedMemoryService) agentRuntimeDependencies {
	return agentRuntimeDependencies{
		cfg: Config{
			MaxTurns:    3,
			PromptsPath: "",
			Provider:    ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"},
		},
		client:       completer,
		registry:     tools.NewRegistry(),
		systemPrompt: "base system prompt",
		memoryRecall: recall,
		memoryLearn:  recall,
	}
}

func newTempSessionStore(t *testing.T) *session.Store {
	t.Helper()
	store, err := session.NewStore(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	return store
}

func containsToolMessage(messages []llm.Message, toolCallID string) bool {
	for _, message := range messages {
		if message.Role == llm.RoleTool && message.ToolCallID == toolCallID {
			return true
		}
	}
	return false
}
