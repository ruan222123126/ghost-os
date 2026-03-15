package orchestration

import (
	"encoding/json"
	"strings"
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

	builder := newSessionHistoryBuilder(
		ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"},
		"system",
		nil,
		3,
	)
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

	builder := newSessionHistoryBuilder(
		ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"},
		"system",
		nil,
		3,
	)
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

func TestSessionHistoryBuilder_ProjectsToolSearchLoadSpanForModel(t *testing.T) {
	sess := session.NewSession("system")
	sess.AddMessage(llm.Message{
		Role: llm.RoleAssistant,
		ToolCalls: []llm.ToolCall{{
			ID:        "call-tfind-load",
			Name:      "tfind",
			Arguments: []byte(`{"action":"load","tool_names":["web_search"]}`),
		}},
	})
	sess.AddMessage(llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: "call-tfind-load",
		Text: agent.FormatToolResult(
			"tfind",
			"trace-tfind-load",
			`{"action":"load","items":[{"name":"web_search","status":"loaded","available_next_turn":true}]}`,
			nil,
		),
	})

	builder := newSessionHistoryBuilder(
		ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"},
		"system",
		nil,
		3,
	)
	history := builder.BuildHistory(sess)
	messages := history.Messages()
	if len(messages) != 2 {
		t.Fatalf("unexpected message count: got %d want 2", len(messages))
	}
	if messages[1].Role != llm.RoleAssistant {
		t.Fatalf("expected projected assistant summary, got %+v", messages[1])
	}
	if !strings.Contains(messages[1].Text, "Loaded dynamic session tools via tfind: `web_search`.") {
		t.Fatalf("unexpected projected summary: %q", messages[1].Text)
	}
	if !strings.Contains(messages[1].Text, "become available next turn") {
		t.Fatalf("expected next-turn hint in projected summary, got %q", messages[1].Text)
	}
}

func TestSessionHistoryBuilder_KeepsToolSearchSearchSpanUnchanged(t *testing.T) {
	sess := session.NewSession("system")
	sess.AddMessage(llm.Message{
		Role: llm.RoleAssistant,
		ToolCalls: []llm.ToolCall{{
			ID:        "call-tfind-search",
			Name:      "tfind",
			Arguments: []byte(`{"action":"search","query":"web"}`),
		}},
	})
	sess.AddMessage(llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: "call-tfind-search",
		Text: agent.FormatToolResult(
			"tfind",
			"trace-tfind-search",
			`{"action":"search","items":[{"name":"web_search","summary":"Search the web."}]}`,
			nil,
		),
	})

	builder := newSessionHistoryBuilder(
		ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"},
		"system",
		nil,
		3,
	)
	history := builder.BuildHistory(sess)
	messages := history.Messages()
	if len(messages) != 3 {
		t.Fatalf("unexpected message count: got %d want 3", len(messages))
	}
	if len(messages[1].ToolCalls) != 1 || messages[1].ToolCalls[0].Name != "tfind" {
		t.Fatalf("expected tfind tool call to remain in history, got %+v", messages[1])
	}
	if messages[2].Role != llm.RoleTool || messages[2].ToolCallID != "call-tfind-search" {
		t.Fatalf("expected tool result to remain in history, got %+v", messages[2])
	}
}
