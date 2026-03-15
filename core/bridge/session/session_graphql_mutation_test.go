package session

import (
	"reflect"
	"testing"
	"time"
)

func TestPendingGraphQLMutationIntentRoundTripAndClone(t *testing.T) {
	sess := NewSession("system")
	intent := PendingGraphQLMutationIntent{
		IntentID:     "intent-1",
		Source:       "crm",
		Domain:       "people",
		PolicyName:   "update_viewer",
		RootMutation: "updateViewer",
		DeliveryKey:  "delivery-1",
		RequestHash:  "request-hash-1",
		Query:        "mutation { updateViewer { ok } }",
		Variables: map[string]any{
			"input": map[string]any{"id": "user-1"},
		},
		QuestionID: "q-1",
		ToolCallID: "call-1",
		TraceID:    "trace-1",
		PreparedAt: time.Now().UTC(),
		Status:     GraphQLMutationIntentPendingApproval,
		Summary:    "summary",
	}
	sess.StorePendingGraphQLMutationIntent(intent)

	cloned, ok := sess.PendingGraphQLMutationIntent(intent.IntentID)
	if !ok {
		t.Fatalf("expected stored intent")
	}
	if !reflect.DeepEqual(cloned, normalizePendingGraphQLMutationIntent(intent)) {
		t.Fatalf("unexpected cloned intent: got=%+v want=%+v", cloned, intent)
	}
	input := cloned.Variables["input"].(map[string]any)
	input["id"] = "mutated"

	stored, ok := sess.PendingGraphQLMutationIntent(intent.IntentID)
	if !ok {
		t.Fatalf("expected stored intent")
	}
	if storedInput := stored.Variables["input"].(map[string]any); storedInput["id"] != "user-1" {
		t.Fatalf("stored variables should remain immutable, got %+v", storedInput)
	}
}

func TestSetHumanAnswerUpdatesGraphQLMutationIntentStatus(t *testing.T) {
	sess := NewSession("system")
	sess.StorePendingGraphQLMutationIntent(PendingGraphQLMutationIntent{
		IntentID:     "intent-1",
		Source:       "crm",
		Domain:       "people",
		PolicyName:   "update_viewer",
		RootMutation: "updateViewer",
		Query:        "mutation { updateViewer { ok } }",
		QuestionID:   "q-1",
		Status:       GraphQLMutationIntentPendingApproval,
	})
	sess.AddPendingQuestion("q-1", PendingHumanQuestion{
		Prompt:     "Approve?",
		ToolName:   "graphql_mutation",
		ToolCallID: "call-1",
		TraceID:    "trace-1",
		CreatedAt:  time.Now().UTC(),
	})

	if !sess.SetHumanAnswer("q-1", "Approve") {
		t.Fatalf("expected approve answer to be accepted")
	}
	approved, ok := sess.PendingGraphQLMutationIntent("intent-1")
	if !ok || approved.Status != GraphQLMutationIntentApproved || approved.CommitState != GraphQLMutationCommitStateApproved || approved.ApprovedAt.IsZero() {
		t.Fatalf("expected approved intent, got %+v ok=%v", approved, ok)
	}

	if !sess.MarkPendingGraphQLMutationIntentExecuted("intent-1", time.Now().UTC()) {
		t.Fatalf("expected executed intent update")
	}
	executed, ok := sess.PendingGraphQLMutationIntent("intent-1")
	if !ok || executed.Status != GraphQLMutationIntentExecuted || executed.ExecutedAt.IsZero() {
		t.Fatalf("expected executed intent, got %+v ok=%v", executed, ok)
	}
}

func TestRemovePendingQuestionDiscardsGraphQLMutationIntent(t *testing.T) {
	sess := NewSession("system")
	sess.StorePendingGraphQLMutationIntent(PendingGraphQLMutationIntent{
		IntentID:     "intent-1",
		Source:       "crm",
		Domain:       "people",
		PolicyName:   "update_viewer",
		RootMutation: "updateViewer",
		Query:        "mutation { updateViewer { ok } }",
		QuestionID:   "q-1",
		Status:       GraphQLMutationIntentPendingApproval,
	})
	sess.AddPendingQuestion("q-1", PendingHumanQuestion{
		Prompt:     "Approve?",
		ToolName:   "graphql_mutation",
		ToolCallID: "call-1",
	})

	if _, ok := sess.RemovePendingQuestion("q-1"); !ok {
		t.Fatalf("expected question removal")
	}
	intent, ok := sess.PendingGraphQLMutationIntent("intent-1")
	if !ok || intent.Status != GraphQLMutationIntentDiscarded {
		t.Fatalf("expected discarded intent after question removal, got %+v ok=%v", intent, ok)
	}
}

func TestReplacePendingGraphQLMutationIntentStoresReceipts(t *testing.T) {
	sess := NewSession("system")
	sess.StorePendingGraphQLMutationIntent(PendingGraphQLMutationIntent{
		IntentID:     "intent-1",
		Source:       "crm",
		Domain:       "people",
		PolicyName:   "update_viewer",
		RootMutation: "updateViewer",
		Status:       GraphQLMutationIntentApproved,
		CommitState:  GraphQLMutationCommitStateApproved,
		Query:        "mutation { updateViewer { ok } }",
		QuestionID:   "q-1",
	})

	ok := sess.ReplacePendingGraphQLMutationIntent(PendingGraphQLMutationIntent{
		IntentID:      "intent-1",
		Source:        "crm",
		Domain:        "people",
		PolicyName:    "update_viewer",
		RootMutation:  "updateViewer",
		Status:        GraphQLMutationIntentDeliveryUnknown,
		CommitState:   GraphQLMutationCommitStateDeliveryUnknown,
		Query:         "mutation { updateViewer { ok } }",
		QuestionID:    "q-1",
		AttemptCount:  1,
		ResponseHash:  "response-hash-1",
		ResponseBytes: 42,
		Receipts: []GraphQLMutationReceipt{{
			Attempt:       1,
			State:         GraphQLMutationIntentDeliveryUnknown,
			DeliveryKey:   "delivery-1",
			RequestHash:   "request-hash-1",
			ResponseHash:  "response-hash-1",
			ResponseBytes: 42,
			HTTPStatus:    0,
			Error:         "timeout",
		}},
	})
	if !ok {
		t.Fatal("expected replace to succeed")
	}

	intent, ok := sess.PendingGraphQLMutationIntent("intent-1")
	if !ok || len(intent.Receipts) != 1 || intent.Receipts[0].State != GraphQLMutationIntentDeliveryUnknown {
		t.Fatalf("expected persisted receipt, got %+v ok=%v", intent, ok)
	}
}
