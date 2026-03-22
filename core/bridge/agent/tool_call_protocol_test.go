package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

type protocolFakeCompleter struct {
	responses []*llm.CompletionResponse
	requests  []llm.CompletionRequest
}

func (f *protocolFakeCompleter) Complete(_ context.Context, request llm.CompletionRequest) (*llm.CompletionResponse, error) {
	f.requests = append(f.requests, protocolCloneRequest(request))
	response := f.responses[0]
	f.responses = f.responses[1:]
	return response, nil
}

type protocolToolCatalog struct {
	tool tools.Tool
}

func (c *protocolToolCatalog) ToolDefs() []llm.ToolDef {
	if c.tool == nil {
		return nil
	}
	return []llm.ToolDef{tools.ToolDefFromTool(c.tool)}
}

func (c *protocolToolCatalog) Get(name string) tools.Tool {
	if c.tool == nil || c.tool.Name() != name {
		return nil
	}
	return c.tool
}

type protocolTool struct {
	name      string
	callCount int
}

func (t *protocolTool) Name() string { return t.name }

func (t *protocolTool) Description() string { return "protocol test tool" }

func (t *protocolTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }

func (t *protocolTool) Execute(context.Context, json.RawMessage, string) (string, error) {
	t.callCount++
	return "ok", nil
}

func (t *protocolTool) InterpretResult(string) tools.ExecuteMeta { return tools.ExecuteMeta{} }

func protocolCloneRequest(request llm.CompletionRequest) llm.CompletionRequest {
	clonedTools := make([]llm.ToolDef, len(request.Tools))
	for i, tool := range request.Tools {
		clonedTools[i] = llm.ToolDef{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  append(json.RawMessage(nil), tool.Parameters...),
			Semantics:   tool.Semantics,
		}
	}
	return llm.CompletionRequest{
		Messages:          llm.CloneMessages(request.Messages),
		Tools:             clonedTools,
		ConversationState: request.ConversationState,
	}
}

func TestRunInvalidToolCallDoesNotReplayBrokenProtocol(t *testing.T) {
	completer := &protocolFakeCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{{
						ID:        "",
						Name:      "echo",
						Arguments: json.RawMessage(`{}`),
					}},
				},
				FinishReason: llm.FinishToolCalls,
			},
			{
				Message:      llm.Message{Role: llm.RoleAssistant, Text: "done"},
				FinishReason: llm.FinishStop,
			},
		},
	}

	agent := NewAgentWithHistory(completer, &protocolToolCatalog{}, NewHistory("system prompt"), 3)
	got, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got != "done" {
		t.Fatalf("unexpected output: got %q want %q", got, "done")
	}
	if len(completer.requests) != 2 {
		t.Fatalf("unexpected request count: got %d want %d", len(completer.requests), 2)
	}

	last := completer.requests[1].Messages[len(completer.requests[1].Messages)-1]
	if last.Role != llm.RoleAssistant {
		t.Fatalf("unexpected replay role: got %q want %q", last.Role, llm.RoleAssistant)
	}
	if len(last.ToolCalls) != 0 {
		t.Fatalf("invalid tool_calls should not survive replay: %+v", last.ToolCalls)
	}
	if !strings.Contains(last.Text, "tool_call.id is empty") {
		t.Fatalf("unexpected replay note: %q", last.Text)
	}
}

func TestRunInvalidToolCallClearsConversationStateBeforeReplay(t *testing.T) {
	completer := &protocolFakeCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{{
						ID:        "",
						Name:      "echo",
						Arguments: json.RawMessage(`{}`),
					}},
				},
				FinishReason: llm.FinishToolCalls,
				ConversationState: llm.ConversationState{
					Provider:           llm.ProviderCodex,
					BaseURL:            "https://api.openai.com/v1",
					Model:              "codex-mini-latest",
					PreviousResponseID: "resp_invalid",
				},
			},
			{
				Message:      llm.Message{Role: llm.RoleAssistant, Text: "done"},
				FinishReason: llm.FinishStop,
			},
		},
	}
	history := NewHistoryFromMessages([]llm.Message{
		{Role: llm.RoleSystem, Text: "system prompt"},
		{Role: llm.RoleUser, Text: "old"},
	})
	history.SetConversationState(llm.ConversationState{
		Provider:           llm.ProviderCodex,
		BaseURL:            "https://api.openai.com/v1",
		Model:              "codex-mini-latest",
		PreviousResponseID: "resp_prev",
	})

	agent := NewAgentWithHistory(completer, &protocolToolCatalog{}, history, 3)
	if _, err := agent.Run(context.Background(), "hello"); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got := completer.requests[0].ConversationState.PreviousResponseID; got != "resp_prev" {
		t.Fatalf("unexpected first previous_response_id: got %q want %q", got, "resp_prev")
	}
	if got := completer.requests[1].ConversationState.PreviousResponseID; got != "" {
		t.Fatalf("expected rewritten replay to clear conversation state, got %q", got)
	}
}

func TestProtocolRunMixedValidAndInvalidToolCallsOnlyReplaysValidCalls(t *testing.T) {
	tool := &protocolTool{name: "echo"}
	completer := &protocolFakeCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{
						{ID: "", Name: "echo", Arguments: json.RawMessage(`{}`)},
						{ID: "call-2", Name: "echo", Arguments: json.RawMessage(`{"input":"hi"}`)},
					},
				},
				FinishReason: llm.FinishToolCalls,
			},
			{
				Message:      llm.Message{Role: llm.RoleAssistant, Text: "done"},
				FinishReason: llm.FinishStop,
			},
		},
	}

	agent := NewAgentWithHistory(completer, &protocolToolCatalog{tool: tool}, NewHistory("system prompt"), 3)
	if _, err := agent.Run(context.Background(), "hello"); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if tool.callCount != 1 {
		t.Fatalf("unexpected tool call count: got %d want %d", tool.callCount, 1)
	}

	requestMessages := completer.requests[1].Messages
	assistantMsg := requestMessages[len(requestMessages)-2]
	if assistantMsg.Role != llm.RoleAssistant {
		t.Fatalf("unexpected assistant replay message: %+v", assistantMsg)
	}
	if len(assistantMsg.ToolCalls) != 1 || assistantMsg.ToolCalls[0].ID != "call-2" {
		t.Fatalf("unexpected replayed tool_calls: %+v", assistantMsg.ToolCalls)
	}
	toolMsg := requestMessages[len(requestMessages)-1]
	if toolMsg.Role != llm.RoleTool || toolMsg.ToolCallID != "call-2" {
		t.Fatalf("unexpected replayed tool result: %+v", toolMsg)
	}
}
