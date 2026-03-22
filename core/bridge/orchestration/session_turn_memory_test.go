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

type fixedMemoryPlanner struct {
	decision  memoryaug.PlannerDecision
	lastInput memoryaug.PlannerInput
}

func (p *fixedMemoryPlanner) Plan(_ context.Context, input memoryaug.PlannerInput) (memoryaug.PlannerDecision, error) {
	p.lastInput = input
	return p.decision, nil
}

type fixedEventRecallService struct {
	output    memoryaug.RecallOutput
	lastInput memoryaug.RecallInput
}

func (s *fixedEventRecallService) Recall(_ context.Context, input memoryaug.RecallInput) (memoryaug.RecallOutput, error) {
	s.lastInput = input
	return s.output, nil
}

type capturingLearningService struct {
	lastInput memoryaug.LearnFromTurnInput
}

func (s *capturingLearningService) LearnFromTurn(_ context.Context, input memoryaug.LearnFromTurnInput) error {
	s.lastInput = input
	return nil
}

func TestSessionRunnerInjectsEventRecallOnlyIntoPrompt(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "noted"},
			FinishReason: llm.FinishStop,
		}},
	}
	planner := &fixedMemoryPlanner{
		decision: plannedMemoryDecision("event-1"),
	}
	recall := &fixedEventRecallService{
		output: memoryaug.RecallOutput{
			PrimaryEvent: &memoryaug.RecallEventHit{
				Event: memorystore.EventNode{ID: "event-1", Title: "Android runtime settings"},
				Role:  "primary",
			},
			PromptBlock: "Active event:\n- primary: Android runtime settings\n\nGlobal preferences:\n- reply_language=zh-CN",
		},
	}
	learn := &capturingLearningService{}
	runner := NewSessionAgentRunner(proTestRuntimeFactory{
		deps: buildMemoryTestDeps(completer, planner, recall, learn),
	}, nil, sessionStore, nil)

	_, sessionID, err := runner.RunTurn(context.Background(), "What language should you use?", "", "trace-memory")
	if err != nil {
		t.Fatalf("run turn: %v", err)
	}
	if !strings.Contains(completer.requests[0].Messages[0].Text, "Active event:") {
		t.Fatalf("expected event memory block in system prompt, got %q", completer.requests[0].Messages[0].Text)
	}
	loaded, err := sessionStore.Load(sessionID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	for _, message := range loaded.Messages {
		if strings.Contains(message.Text, "Active event:") || strings.Contains(message.Text, "Global preferences:") {
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
	planner := &fixedMemoryPlanner{decision: plannedMemoryDecision("event-1")}
	recall := &fixedEventRecallService{
		output: memoryaug.RecallOutput{
			PrimaryEvent: &memoryaug.RecallEventHit{
				Event: memorystore.EventNode{ID: "event-1", Title: "Android runtime settings"},
				Role:  "primary",
			},
			PromptBlock: "Active event:\n- primary: Android runtime settings",
		},
	}
	runner := NewSessionAgentRunner(proTestRuntimeFactory{
		deps: buildMemoryTestDeps(completer, planner, recall, &capturingLearningService{}),
	}, nil, sessionStore, nil)

	_, _, err := runner.RunTurn(context.Background(), "continue", sess.ID, "trace-resume")
	if err != nil {
		t.Fatalf("run resumed turn: %v", err)
	}
	request := completer.requests[0]
	if !containsToolMessage(request.Messages, "call-ask") {
		t.Fatalf("expected ask_human tool message to remain in resumed history, got %+v", request.Messages)
	}
	if !strings.Contains(request.Messages[0].Text, "Active event:") {
		t.Fatalf("expected event memory context in system prompt, got %q", request.Messages[0].Text)
	}
}

func TestSessionRunnerPlannerUsesRecentContext(t *testing.T) {
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
	planner := &fixedMemoryPlanner{decision: plannedMemoryDecision("event-1")}
	recall := &fixedEventRecallService{}
	runner := NewSessionAgentRunner(proTestRuntimeFactory{
		deps: buildMemoryTestDeps(completer, planner, recall, &capturingLearningService{}),
	}, nil, sessionStore, nil)

	if _, _, err := runner.RunTurn(context.Background(), "continue with the runtime change", sess.ID, "trace-query"); err != nil {
		t.Fatalf("run turn: %v", err)
	}
	if planner.lastInput.UserMessage != "continue with the runtime change" {
		t.Fatalf("expected planner to receive current user message, got %+v", planner.lastInput)
	}
	if len(planner.lastInput.RecentMessages) < 2 {
		t.Fatalf("expected planner to receive recent conversation context, got %+v", planner.lastInput)
	}
}

func TestSessionRunnerPassesPrimaryEventIntoLearning(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "done"},
			FinishReason: llm.FinishStop,
		}},
	}
	planner := &fixedMemoryPlanner{decision: plannedMemoryDecision("event-1")}
	recall := &fixedEventRecallService{}
	learn := &capturingLearningService{}
	runner := NewSessionAgentRunner(proTestRuntimeFactory{
		deps: buildMemoryTestDeps(completer, planner, recall, learn),
	}, nil, sessionStore, nil)

	if _, _, err := runner.RunTurn(context.Background(), "continue", "", "trace-learn"); err != nil {
		t.Fatalf("run turn: %v", err)
	}
	if learn.lastInput.PrimaryEventID != "event-1" {
		t.Fatalf("expected learning input to carry primary event, got %+v", learn.lastInput)
	}
	if len(learn.lastInput.ActiveEventIDs) != 1 || learn.lastInput.ActiveEventIDs[0] != "event-1" {
		t.Fatalf("expected active_event_ids to match planner decision, got %+v", learn.lastInput)
	}
}

func TestSessionRunnerInjectsDynamicToolStateIntoPrompt(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	sess := session.NewSession("base system prompt")
	sess.AdvanceToolTurn(3)
	sess.EnsureDynamicToolLoaded("graphql_query", "tfind")
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "continuing"},
			FinishReason: llm.FinishStop,
		}},
	}
	registry := tools.NewRegistry()
	for _, name := range []string{"ask_human", "tfind", "graphql_query"} {
		registry.Register(&runnerMockTool{name: name})
	}

	runner := NewSessionAgentRunner(proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg: Config{
				MaxTurns:    3,
				PromptsPath: "",
				Provider:    ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"},
				ToolSearch: ToolSearchConfig{
					Enabled:   true,
					IdleTurns: 3,
				},
			},
			client:       completer,
			registry:     registry,
			systemPrompt: "base system prompt",
		},
	}, nil, sessionStore, nil)

	if _, _, err := runner.RunTurn(context.Background(), "continue", sess.ID, "trace-dynamic-tools"); err != nil {
		t.Fatalf("run turn: %v", err)
	}
	request := completer.requests[0]
	prompt := request.Messages[0].Text
	for _, snippet := range []string{
		"## Dynamic Tool State",
		"`graphql_query` is active in this session; remaining_idle_turns=3.",
		"`tfind(action=\"search\")`",
	} {
		if !strings.Contains(prompt, snippet) {
			t.Fatalf("expected prompt to contain %q, got %q", snippet, prompt)
		}
	}
	if !containsToolDef(request.Tools, "graphql_query") {
		t.Fatalf("expected dynamically loaded tool to be available this turn, got %+v", request.Tools)
	}
}

func plannedMemoryDecision(primaryEventID string) memoryaug.PlannerDecision {
	return memoryaug.PlannerDecision{
		PrimaryEvent: memoryaug.PlannerEventRef{
			EventID: primaryEventID,
			Reason:  "same task",
		},
		RecallPlan: memoryaug.RecallPlan{
			EventIDs:           []string{primaryEventID},
			IncludeNodeSummary: true,
			IncludeWorkflow:    true,
			IncludePreference:  true,
			IncludeFact:        true,
			AllowLearning:      true,
		},
	}
}

func buildMemoryTestDeps(
	completer *proTestCompleter,
	planner memoryaug.IntentPlanner,
	recall memoryaug.RecallService,
	learn memoryaug.LearningService,
) agentRuntimeDependencies {
	return agentRuntimeDependencies{
		cfg: Config{
			MaxTurns:    3,
			PromptsPath: "",
			Provider:    ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"},
			MemoryAugmentation: MemoryAugmentationConfig{
				UserScopeID: memorystore.DefaultUserScopeID,
			},
		},
		client:       completer,
		registry:     tools.NewRegistry(),
		systemPrompt: "base system prompt",
		memoryPlan:   planner,
		memoryRecall: recall,
		memoryLearn:  learn,
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

func containsToolDef(defs []llm.ToolDef, name string) bool {
	for _, def := range defs {
		if def.Name == name {
			return true
		}
	}
	return false
}
