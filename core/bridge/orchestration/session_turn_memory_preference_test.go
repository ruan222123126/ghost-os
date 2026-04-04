package orchestration

import (
	"context"
	"testing"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/memoryaug"
)

type panicMemoryPlanner struct{}

func (panicMemoryPlanner) Plan(context.Context, memoryaug.PlannerInput) (memoryaug.PlannerDecision, error) {
	panic("memory planner should not run for a fresh global preference turn")
}

type panicEventRecallService struct{}

func (panicEventRecallService) Recall(context.Context, memoryaug.RecallInput) (memoryaug.RecallOutput, error) {
	panic("memory recall should not run for a fresh global preference turn")
}

func TestSessionRunnerSkipsEventPlanningForFreshGlobalPreferenceTurn(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "已记录。"},
			FinishReason: llm.FinishStop,
		}},
	}
	learn := &capturingLearningService{}
	runner := NewSessionAgentRunner(proTestRuntimeFactory{
		deps: buildMemoryTestDeps(completer, panicMemoryPlanner{}, panicEventRecallService{}, learn),
	}, nil, sessionStore, nil)

	if _, _, err := runner.RunTurn(context.Background(), "请记住：我偏好中文回答。", "", "trace-global-pref"); err != nil {
		t.Fatalf("run turn: %v", err)
	}
	if learn.lastInput.PrimaryEventID != "" {
		t.Fatalf("expected no primary event for global preference turn, got %+v", learn.lastInput)
	}
	if len(learn.lastInput.ActiveEventIDs) != 0 {
		t.Fatalf("expected no active event ids for global preference turn, got %+v", learn.lastInput)
	}
	if !learn.lastInput.AllowWrite {
		t.Fatalf("expected global preference turn to keep memory learning enabled, got %+v", learn.lastInput)
	}
}

func TestSessionRunnerSkipsEventPlanningForFreshGreetingTurn(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "你好。"},
			FinishReason: llm.FinishStop,
		}},
	}
	learn := &capturingLearningService{}
	runner := NewSessionAgentRunner(proTestRuntimeFactory{
		deps: buildMemoryTestDeps(completer, panicMemoryPlanner{}, panicEventRecallService{}, learn),
	}, nil, sessionStore, nil)

	if _, _, err := runner.RunTurn(context.Background(), "你好", "", "trace-greeting"); err != nil {
		t.Fatalf("run turn: %v", err)
	}
	if learn.lastInput.PrimaryEventID != "" {
		t.Fatalf("expected no primary event for fresh greeting turn, got %+v", learn.lastInput)
	}
	if len(learn.lastInput.ActiveEventIDs) != 0 {
		t.Fatalf("expected no active event ids for fresh greeting turn, got %+v", learn.lastInput)
	}
	if !learn.lastInput.AllowWrite {
		t.Fatalf("expected fresh greeting turn to keep memory learning enabled, got %+v", learn.lastInput)
	}
}
