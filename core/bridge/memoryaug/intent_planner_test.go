package memoryaug

import (
	"context"
	"testing"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/memorystore"
)

type plannerTestCompleter struct {
	response string
}

func (c plannerTestCompleter) Complete(context.Context, llm.CompletionRequest) (*llm.CompletionResponse, error) {
	return &llm.CompletionResponse{
		Message: llm.Message{Role: llm.RoleAssistant, Text: c.response},
	}, nil
}

func TestIntentPlannerReusesExistingEvent(t *testing.T) {
	store := newTestStore(t)
	event := mustCreateEventNode(t, store, "session-1", "Android runtime settings")
	planner := NewIntentPlanner(newTestSettings(), store, plannerTestCompleter{
		response: `{"primary_event":{"event_id":"` + event.ID + `","reason":"same task"},"adjacent_events":[],"reuse_existing":true,"create_new_event":false,"recall_plan":{"event_ids":["` + event.ID + `"],"include_node_summary":true,"include_workflow":true,"include_preference":true,"include_fact":true,"include_profile":false,"allow_learning":true}}`,
	})

	decision, err := planner.Plan(context.Background(), PlannerInput{
		SessionID:   "session-1",
		UserMessage: "continue the runtime settings change",
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if decision.PrimaryEvent.EventID != event.ID || decision.CreateNewEvent {
		t.Fatalf("expected planner to reuse existing event, got %+v", decision)
	}
}

func TestIntentPlannerCreatesNewEvent(t *testing.T) {
	store := newTestStore(t)
	planner := NewIntentPlanner(newTestSettings(), store, plannerTestCompleter{
		response: `{"primary_event":{"reason":"new task"},"adjacent_events":[],"reuse_existing":false,"create_new_event":true,"new_event_title":"Web config parser","new_event_summary":"Fix parser regression","recall_plan":{"event_ids":[],"include_node_summary":true,"include_workflow":true,"include_preference":false,"include_fact":true,"include_profile":false,"allow_learning":true}}`,
	})

	decision, err := planner.Plan(context.Background(), PlannerInput{
		SessionID:   "session-1",
		UserMessage: "switch to the web config parser failure",
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if decision.PrimaryEvent.EventID == "" || !decision.CreateNewEvent {
		t.Fatalf("expected new event to be created, got %+v", decision)
	}
	node, err := store.GetEventNode(context.Background(), decision.PrimaryEvent.EventID)
	if err != nil {
		t.Fatalf("load created event: %v", err)
	}
	if node.Title != "Web config parser" {
		t.Fatalf("unexpected created event: %+v", node)
	}
	state, err := store.LoadSessionEventState(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("load session event state: %v", err)
	}
	if state.PrimaryEventID != node.ID {
		t.Fatalf("expected session state to point at new primary event, got %+v", state)
	}
}

func TestIntentPlannerNormalizesRecallPlanEventIDsToResolvedActiveIDs(t *testing.T) {
	store := newTestStore(t)
	planner := NewIntentPlanner(newTestSettings(), store, plannerTestCompleter{
		response: `{"primary_event":{"reason":"new task"},"adjacent_events":[],"reuse_existing":false,"create_new_event":true,"new_event_title":"搜索当前AI发展情况","new_event_summary":"用户希望获取现在AI领域的最新情况与整体动态。","recall_plan":{"event_ids":["evt_ai_current_landscape_search"],"include_node_summary":true,"include_workflow":true,"include_preference":false,"include_fact":true,"include_profile":false,"allow_learning":true}}`,
	})

	decision, err := planner.Plan(context.Background(), PlannerInput{
		SessionID:   "session-1",
		UserMessage: "帮我搜索现在ai的情况",
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(decision.RecallPlan.EventIDs) != 1 || decision.RecallPlan.EventIDs[0] != decision.PrimaryEvent.EventID {
		t.Fatalf("expected recall_plan.event_ids to be normalized to the resolved primary event id, got %+v", decision)
	}
	state, err := store.LoadSessionEventState(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("load session event state: %v", err)
	}
	if len(state.ActiveEventIDs) != 1 || state.ActiveEventIDs[0] != decision.PrimaryEvent.EventID {
		t.Fatalf("expected persisted active_event_ids to contain the resolved primary event id, got %+v", state)
	}
}

func TestPlannerBuildsParentEdgeForSubtask(t *testing.T) {
	store := newTestStore(t)
	parent := mustCreateEventNode(t, store, "session-1", "Android runtime settings")
	planner := NewIntentPlanner(newTestSettings(), store, plannerTestCompleter{
		response: `{"primary_event":{"reason":"new child task"},"adjacent_events":[{"event_id":"` + parent.ID + `","reason":"parent task"}],"reuse_existing":false,"create_new_event":true,"new_event_title":"Android runtime GraphQL toggle","new_event_summary":"Fix nested toggle wiring","edge_updates":[{"from_event_id":"` + parent.ID + `","to_event_id":"primary","edge_type":"parent_of","confidence":0.9}],"recall_plan":{"event_ids":["` + parent.ID + `"],"include_node_summary":true,"include_workflow":true,"include_preference":false,"include_fact":true,"include_profile":false,"allow_learning":true}}`,
	})

	decision, err := planner.Plan(context.Background(), PlannerInput{
		SessionID:   "session-1",
		UserMessage: "work on the GraphQL toggle subtask",
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	edges, err := store.ListEventEdges(context.Background(), "session-1", []string{parent.ID, decision.PrimaryEvent.EventID})
	if err != nil {
		t.Fatalf("list edges: %v", err)
	}
	if len(edges) != 1 || edges[0].EdgeType != memorystore.EventEdgeParentOf {
		t.Fatalf("expected parent_of edge, got %+v", edges)
	}
}
