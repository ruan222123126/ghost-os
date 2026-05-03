package llm

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateReasoningReplayRejectsMissingReasoningAfterThinkingToolTurn(t *testing.T) {
	messages := []Message{
		{Role: RoleSystem, Text: "system prompt"},
		{Role: RoleUser, Text: "hello"},
		{
			Role:             RoleAssistant,
			ReasoningContent: mustRawJSON(t, "first plan"),
			ToolCalls:        []ToolCall{{ID: "call-1", Name: "echo", Arguments: json.RawMessage(`{}`)}},
		},
		{Role: RoleTool, ToolCallID: "call-1", Text: `{"status":"success"}`},
		{
			Role:      RoleAssistant,
			ToolCalls: []ToolCall{{ID: "call-2", Name: "echo", Arguments: json.RawMessage(`{}`)}},
		},
	}

	err := ValidateReasoningReplay(messages)
	if err == nil {
		t.Fatal("expected reasoning replay validation error")
	}
	if !strings.Contains(err.Error(), "reasoning_content") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReasoningReplayRejectsMissingReasoningOnFinalAssistantTurn(t *testing.T) {
	messages := []Message{
		{Role: RoleSystem, Text: "system prompt"},
		{Role: RoleUser, Text: "hello"},
		{
			Role:             RoleAssistant,
			ReasoningContent: mustRawJSON(t, "first plan"),
			ToolCalls:        []ToolCall{{ID: "call-1", Name: "echo", Arguments: json.RawMessage(`{}`)}},
		},
		{Role: RoleTool, ToolCallID: "call-1", Text: `{"status":"success"}`},
		{Role: RoleAssistant, Text: "done"},
	}

	err := ValidateReasoningReplay(messages)
	if err == nil {
		t.Fatal("expected reasoning replay validation error")
	}
	if !strings.Contains(err.Error(), "reasoning_content") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReasoningReplayResetsAfterNewUserTurn(t *testing.T) {
	messages := []Message{
		{Role: RoleSystem, Text: "system prompt"},
		{Role: RoleUser, Text: "hello"},
		{
			Role:             RoleAssistant,
			ReasoningContent: mustRawJSON(t, "first plan"),
			ToolCalls:        []ToolCall{{ID: "call-1", Name: "echo", Arguments: json.RawMessage(`{}`)}},
		},
		{Role: RoleTool, ToolCallID: "call-1", Text: `{"status":"success"}`},
		{Role: RoleAssistant, Text: "done", ReasoningContent: mustRawJSON(t, "final plan")},
		{Role: RoleUser, Text: "next question"},
		{
			Role:      RoleAssistant,
			ToolCalls: []ToolCall{{ID: "call-2", Name: "echo", Arguments: json.RawMessage(`{}`)}},
		},
	}

	if err := ValidateReasoningReplay(messages); err != nil {
		t.Fatalf("expected new user turn to reset replay validation, got %v", err)
	}
}

func mustRawJSON(t *testing.T, value string) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal reasoning content: %v", err)
	}
	return raw
}
