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

func TestToOpenAIRequestSanitizesTopLevelToolSchemaCombinators(t *testing.T) {
	request, err := toOpenAIRequest("gpt-4o", CompletionRequest{
		Messages: []Message{{Role: RoleUser, Text: "hello"}},
		Tools: []ToolDef{{
			Name: "bash_exec",
			Parameters: json.RawMessage(`{
				"type":"object",
				"properties":{"command":{"type":"string"}},
				"required":["command"],
				"additionalProperties":false,
				"allOf":[{"if":{"properties":{"interactive":{"const":true}}},"then":{"not":{"required":["login"]}}}]
			}`),
		}},
	})
	if err != nil {
		t.Fatalf("toOpenAIRequest returned error: %v", err)
	}

	if len(request.Tools) != 1 {
		t.Fatalf("unexpected tool count: got %d want %d", len(request.Tools), 1)
	}

	var schema map[string]any
	if err := json.Unmarshal(request.Tools[0].Function.Parameters, &schema); err != nil {
		t.Fatalf("decode sanitized schema: %v", err)
	}
	if schema["type"] != "object" {
		t.Fatalf("unexpected top-level type: %#v", schema["type"])
	}
	if _, exists := schema["allOf"]; exists {
		t.Fatalf("unexpected top-level allOf: %+v", schema)
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected properties payload: %+v", schema["properties"])
	}
	if _, ok := properties["command"]; !ok {
		t.Fatalf("sanitized schema lost command property: %+v", properties)
	}
}

func TestToOpenAIRequestRejectsNonObjectToolSchema(t *testing.T) {
	_, err := toOpenAIRequest("gpt-4o", CompletionRequest{
		Messages: []Message{{Role: RoleUser, Text: "hello"}},
		Tools: []ToolDef{{
			Name:       "broken_tool",
			Parameters: json.RawMessage(`{"type":"string"}`),
		}},
	})
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), `top-level tool schema type "string" is not supported`) {
		t.Fatalf("unexpected error: %v", err)
	}
}
