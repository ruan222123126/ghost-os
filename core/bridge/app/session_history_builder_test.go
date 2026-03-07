package app

import (
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

	builder := newSessionHistoryBuilder(Config{Provider: llm.ProviderOpenAI, Model: "gpt-4o"}, "system", nil)
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
