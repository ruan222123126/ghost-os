package orchestration

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
)

func TestSessionHistoryBuilder_BuildHistoryWithResolvedQuestionsReturnsAnsweredQuestions(t *testing.T) {
	sess := session.NewSession("system")
	createdAt := time.Date(2026, 3, 7, 11, 0, 0, 0, time.UTC)
	sess.AddMessage(llm.Message{
		Role: llm.RoleAssistant,
		ToolCalls: []llm.ToolCall{{
			ID:        "call-ask",
			Name:      "ask_human",
			Arguments: json.RawMessage(`{"prompt":"Ship now?"}`),
		}},
	})
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
		bridgeconfig.ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"},
		"system",
		nil,
		3,
		false,
		"",
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

func TestSessionHistoryBuilder_ProjectsToolSearchLoadSpanForModel(t *testing.T) {
	sess := session.NewSession("system")
	sess.AddMessage(llm.Message{
		Role: llm.RoleAssistant,
		ToolCalls: []llm.ToolCall{{
			ID:        "call-sfind-load",
			Name:      "sfind",
			Arguments: []byte(`{"action":"load","skill_names":["release_flow"]}`),
		}},
	})
	sess.AddMessage(llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: "call-sfind-load",
		Text: agent.FormatToolResult(
			"sfind",
			"trace-sfind-load",
			`{"action":"load","kind":"skill","items":[{"name":"release_flow","status":"loaded","available_now":true}]}`,
			nil,
		),
	})
	sess.AddMessage(llm.Message{
		Role: llm.RoleAssistant,
		ToolCalls: []llm.ToolCall{{
			ID:        "call-keep-1",
			Name:      "read_file",
			Arguments: []byte(`{"path":"one.txt"}`),
		}},
	})
	sess.AddMessage(llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: "call-keep-1",
		Text:       agent.FormatToolResult("read_file", "trace-keep-1", "File: one.txt", nil),
	})
	sess.AddMessage(llm.Message{
		Role: llm.RoleAssistant,
		ToolCalls: []llm.ToolCall{{
			ID:        "call-keep-2",
			Name:      "read_file",
			Arguments: []byte(`{"path":"two.txt"}`),
		}},
	})
	sess.AddMessage(llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: "call-keep-2",
		Text:       agent.FormatToolResult("read_file", "trace-keep-2", "File: two.txt", nil),
	})

	builder := newSessionHistoryBuilder(
		bridgeconfig.ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"},
		"system",
		nil,
		3,
		true,
		"",
	)
	history := builder.BuildHistory(sess)
	messages := history.Messages()
	if len(messages) != 6 {
		t.Fatalf("unexpected message count: got %d want 6", len(messages))
	}
	if messages[1].Role != llm.RoleAssistant {
		t.Fatalf("expected projected assistant summary, got %+v", messages[1])
	}
	if !strings.Contains(messages[1].Text, "Loaded dynamic session skills via sfind: `release_flow`.") {
		t.Fatalf("unexpected projected summary: %q", messages[1].Text)
	}
	if !strings.Contains(messages[1].Text, "available now in the current user turn") {
		t.Fatalf("expected immediate-availability hint in projected summary, got %q", messages[1].Text)
	}
}

func TestSessionHistoryBuilder_KeepsToolSearchSearchSpanUnchanged(t *testing.T) {
	sess := session.NewSession("system")
	sess.AddMessage(llm.Message{
		Role: llm.RoleAssistant,
		ToolCalls: []llm.ToolCall{{
			ID:        "call-sfind-search",
			Name:      "sfind",
			Arguments: []byte(`{"action":"search","query":"web"}`),
		}},
	})
	sess.AddMessage(llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: "call-sfind-search",
		Text: agent.FormatToolResult(
			"sfind",
			"trace-sfind-search",
			`{"action":"search","kind":"skill","items":[{"name":"release_flow","summary":"Release workflow."}]}`,
			nil,
		),
	})

	builder := newSessionHistoryBuilder(
		bridgeconfig.ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"},
		"system",
		nil,
		3,
		true,
		"",
	)
	history := builder.BuildHistory(sess)
	messages := history.Messages()
	if len(messages) != 3 {
		t.Fatalf("unexpected message count: got %d want 3", len(messages))
	}
	if len(messages[1].ToolCalls) != 1 || messages[1].ToolCalls[0].Name != "sfind" {
		t.Fatalf("expected sfind tool call to remain in history, got %+v", messages[1])
	}
	if messages[2].Role != llm.RoleTool || messages[2].ToolCallID != "call-sfind-search" {
		t.Fatalf("expected tool result to remain in history, got %+v", messages[2])
	}
}

func TestSessionHistoryBuilder_DropsOrphanToolMessageAfterCompletedAnswer(t *testing.T) {
	sess := session.NewSession("system")
	sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: "看看影视飓风的粉丝数"})
	sess.AddMessage(llm.Message{
		Role: llm.RoleAssistant,
		ToolCalls: []llm.ToolCall{{
			ID:        "call-search-1",
			Name:      "web_search",
			Arguments: json.RawMessage(`{"query":"影视飓风 B站 粉丝数"}`),
		}},
	})
	sess.AddMessage(llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: "call-search-1",
		Text:       agent.FormatToolResult("web_search", "trace-search-1", "ok", nil),
	})
	sess.AddMessage(llm.Message{
		Role: llm.RoleAssistant,
		Text: "粉丝数是 1612.6 万。",
	})
	sess.AddMessage(llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: "call-orphan-1",
		Text:       agent.FormatToolResult("bash_exec", "trace-orphan-1", "orphan", nil),
	})

	builder := newSessionHistoryBuilder(
		bridgeconfig.ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"},
		"system",
		nil,
		3,
		false,
		"",
	)
	history := builder.BuildHistory(sess)
	messages := history.Messages()

	if len(messages) != 5 {
		t.Fatalf("unexpected sanitized message count: got %d want %d", len(messages), 5)
	}
	last := messages[len(messages)-1]
	if last.Role != llm.RoleAssistant || last.Text != "粉丝数是 1612.6 万。" {
		t.Fatalf("expected orphan tool message to be removed, got %+v", last)
	}
	if messages[2].Role != llm.RoleAssistant || len(messages[2].ToolCalls) != 1 {
		t.Fatalf("expected valid tool-call assistant message to remain intact, got %+v", messages[2])
	}
	if messages[3].Role != llm.RoleTool || messages[3].ToolCallID != "call-search-1" {
		t.Fatalf("expected matching tool result to remain intact, got %+v", messages[3])
	}
}
