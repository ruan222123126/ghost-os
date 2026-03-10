package llm

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestToOpenAIRequestRejectsInvalidAssistantToolCallHistory(t *testing.T) {
	_, err := toOpenAIRequest("gpt-4o", CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Text: "hello"},
			{
				Role: RoleAssistant,
				ToolCalls: []ToolCall{
					{ID: "", Name: "echo", Arguments: json.RawMessage(`{}`)},
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "tool_call.id is empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestToAnthropicRequestRejectsUnpairedToolResultHistory(t *testing.T) {
	_, err := toAnthropicRequest("claude-3-7-sonnet", 1024, CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Text: "hello"},
			{Role: RoleTool, ToolCallID: "call-1", Text: `{"status":"success"}`},
		},
	})
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), `unknown tool_call_id "call-1"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCodexClientRejectsDanglingAssistantToolCallHistory(t *testing.T) {
	client := NewClientWithOptions(ClientOptions{
		Provider: ProviderCodex,
		BaseURL:  "https://api.openai.com/v1",
		Model:    "codex-mini-latest",
	})

	_, err := client.buildProviderRequest(CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Text: "hello"},
			{
				Role: RoleAssistant,
				ToolCalls: []ToolCall{
					{ID: "call-1", Name: "echo", Arguments: json.RawMessage(`{}`)},
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "missing matching tool results") {
		t.Fatalf("unexpected error: %v", err)
	}
}
