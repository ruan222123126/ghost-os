package orchestration

import (
	"encoding/json"
	"testing"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
)

func TestSessionHistoryBuilder_BuildHistoryWithResolvedQuestionsReturnsAnsweredQuestions(t *testing.T) {
	sess := session.NewSession("system")
	createdAt := time.Date(2026, 3, 7, 11, 0, 0, 0, time.UTC)
	sess.AddPendingQuestion("q-1", session.PendingHumanQuestion{
		Prompt:     "Ship now?",
		ToolCallID: "call-ask",
		TraceID:    "trace-q1",
		CreatedAt:  createdAt,
	})
	if !sess.SetHumanAnswer("q-1", "yes") {
		t.Fatalf("expected human answer to be accepted")
	}

	builder := newSessionHistoryBuilder(ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"}, "system", nil)
	history, resolved := builder.BuildHistoryWithResolvedQuestions(sess)
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved question, got %d", len(resolved))
	}
	if resolved[0].QuestionID != "q-1" || resolved[0].Prompt != "Ship now?" || resolved[0].Answer != "yes" {
		t.Fatalf("unexpected resolved question: %+v", resolved[0])
	}
	if resolved[0].AskedAt != createdAt {
		t.Fatalf("unexpected asked_at: got %s want %s", resolved[0].AskedAt, createdAt)
	}
	if resolved[0].AnsweredAt.IsZero() {
		t.Fatalf("expected answered_at to be populated")
	}
	if len(sess.PendingQuestions) != 0 || len(sess.HumanAnswers) != 0 {
		t.Fatalf("expected resolved question to be consumed, pending=%v answers=%v", sess.PendingQuestions, sess.HumanAnswers)
	}

	messages := history.Messages()
	if len(messages) == 0 {
		t.Fatalf("expected history messages")
	}
	last := messages[len(messages)-1]
	if last.Role != llm.RoleTool || last.ToolCallID != "call-ask" {
		t.Fatalf("expected injected tool message, got %+v", last)
	}
	envelope, ok := agent.ParseToolResultEnvelope(last.Text)
	if !ok {
		t.Fatalf("expected tool result envelope, got %q", last.Text)
	}
	if envelope.Tool != "ask_human" {
		t.Fatalf("unexpected tool name: %q", envelope.Tool)
	}
	if envelope.TraceID != "trace-q1" {
		t.Fatalf("unexpected trace id: %q", envelope.TraceID)
	}
}

func TestSessionHistoryBuilderInjectsGraphQLMutationResolvedToolResult(t *testing.T) {
	sess := session.NewSession("system")
	createdAt := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
	sess.StorePendingGraphQLMutationIntent(session.PendingGraphQLMutationIntent{
		IntentID:     "intent-1",
		Source:       "crm",
		Domain:       "people",
		PolicyName:   "update_viewer",
		RootMutation: "updateViewer",
		Query:        "mutation { updateViewer { ok } }",
		QuestionID:   "q-graphql",
		ToolCallID:   "call-mutation",
		TraceID:      "trace-mutation",
		PreparedAt:   createdAt,
		Status:       session.GraphQLMutationIntentPendingApproval,
		Summary:      "summary",
	})
	sess.AddPendingQuestion("q-graphql", session.PendingHumanQuestion{
		Prompt:     "Approve mutation?",
		ToolName:   "graphql_mutation",
		ToolCallID: "call-mutation",
		TraceID:    "trace-mutation",
		CreatedAt:  createdAt,
	})
	if !sess.SetHumanAnswer("q-graphql", "Approve") {
		t.Fatalf("expected human answer to be accepted")
	}

	builder := newSessionHistoryBuilder(ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"}, "system", nil)
	history, resolved := builder.BuildHistoryWithResolvedQuestions(sess)
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved question, got %d", len(resolved))
	}
	if resolved[0].ToolName != "graphql_mutation" || resolved[0].Summary != "summary" {
		t.Fatalf("unexpected resolved question metadata: %+v", resolved[0])
	}

	last := history.Messages()[len(history.Messages())-1]
	envelope, ok := agent.ParseToolResultEnvelope(last.Text)
	if !ok {
		t.Fatalf("expected tool result envelope, got %q", last.Text)
	}
	if envelope.Tool != "graphql_mutation" {
		t.Fatalf("unexpected tool name: %q", envelope.Tool)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(envelope.Output), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload["intent_id"] != "intent-1" || payload["approval_status"] != session.GraphQLMutationIntentApproved {
		t.Fatalf("unexpected graphql_mutation payload: %+v", payload)
	}
}
