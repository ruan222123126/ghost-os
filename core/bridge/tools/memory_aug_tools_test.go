package tools

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"ghost-os/bridge/memoryaug"
	"ghost-os/bridge/memorystore"
	"ghost-os/bridge/session"
)

type plannerDebugStub struct {
	decision memoryaug.PlannerDecision
}

func (s *plannerDebugStub) Plan(context.Context, memoryaug.PlannerInput) (memoryaug.PlannerDecision, error) {
	return s.decision, nil
}

type recallDebugStub struct {
	output memoryaug.RecallOutput
}

func (s *recallDebugStub) Recall(context.Context, memoryaug.RecallInput) (memoryaug.RecallOutput, error) {
	return s.output, nil
}

func TestMemoryLearnedListToolListsEventMemories(t *testing.T) {
	store, err := memorystore.NewStore(filepath.Join(t.TempDir(), "memory.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	event, err := store.CreateEventNode(context.Background(), memorystore.EventNodeInput{
		SessionID: "session-1",
		Title:     "Android runtime settings",
		Summary:   "Android runtime settings summary",
		Status:    memorystore.EventStatusActive,
	})
	if err != nil {
		t.Fatalf("create event node: %v", err)
	}
	if _, err := store.CreateEventMemory(context.Background(), memorystore.EventMemoryInput{
		EventID:    event.ID,
		MemoryType: memorystore.MemoryTypeWorkflow,
		Summary:    "run Android tests after runtime flag changes",
		Content:    "run Android tests after runtime flag changes",
		Confidence: 0.9,
	}, nil); err != nil {
		t.Fatalf("create event memory: %v", err)
	}

	tool := NewMemoryLearnedListTool(store)
	output, err := tool.Execute(context.Background(), json.RawMessage(`{"session_id":"session-1","status":"active"}`), "trace-learned-list")
	if err != nil {
		t.Fatalf("execute memory_learned_list: %v", err)
	}
	var result memoryLearnedListResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].EventID != event.ID {
		t.Fatalf("unexpected learned list result: %+v", result)
	}
}

func TestMemoryRecallDebugToolReturnsPlannerDecisionAndPromptBlock(t *testing.T) {
	planner := &plannerDebugStub{
		decision: memoryaug.PlannerDecision{
			PrimaryEvent: memoryaug.PlannerEventRef{
				EventID: "event-1",
				Reason:  "same task",
			},
			RecallPlan: memoryaug.RecallPlan{
				EventIDs:           []string{"event-1"},
				IncludeNodeSummary: true,
				IncludeWorkflow:    true,
				IncludeFact:        true,
				AllowLearning:      true,
			},
		},
	}
	recall := &recallDebugStub{
		output: memoryaug.RecallOutput{
			PrimaryEvent: &memoryaug.RecallEventHit{
				Event: memorystore.EventNode{ID: "event-1", Title: "Android runtime settings"},
				Role:  "primary",
			},
			PromptBlock: "Active event:\n- primary: Android runtime settings",
		},
	}
	tool := NewMemoryRecallDebugTool(planner, recall, memorystore.DefaultUserScopeID)
	sess := session.NewSession("system")
	sess.ID = "session-1"

	output, err := tool.Execute(
		WithSession(context.Background(), sess),
		json.RawMessage(`{"query":"current phase"}`),
		"trace-recall-debug",
	)
	if err != nil {
		t.Fatalf("execute memory_recall_debug: %v", err)
	}
	var result memoryRecallDebugResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.SessionID != "session-1" || result.PlannerDecision.PrimaryEvent.EventID != "event-1" {
		t.Fatalf("unexpected planner decision in result: %+v", result)
	}
	if len(result.ActiveNodes) != 1 || result.PromptBlock == "" {
		t.Fatalf("unexpected active nodes result: %+v", result)
	}
}
