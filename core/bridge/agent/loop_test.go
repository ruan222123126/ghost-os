package agent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

type fakeCompleter struct {
	responses []*llm.CompletionResponse
	requests  []llm.CompletionRequest
}

// fakeCompleter 通过预置响应驱动 Agent 循环，避免真实网络依赖。
func (f *fakeCompleter) Complete(_ context.Context, request llm.CompletionRequest) (*llm.CompletionResponse, error) {
	f.requests = append(f.requests, cloneCompletionRequest(request))
	if len(f.responses) == 0 {
		return nil, errors.New("unexpected complete call")
	}

	response := f.responses[0]
	f.responses = f.responses[1:]
	return response, nil
}

func cloneCompletionRequest(request llm.CompletionRequest) llm.CompletionRequest {
	clonedTools := make([]llm.ToolDef, len(request.Tools))
	for i, tool := range request.Tools {
		clonedTools[i] = llm.ToolDef{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  cloneRawJSON(tool.Parameters),
		}
	}

	return llm.CompletionRequest{
		Messages: llm.CloneMessages(request.Messages),
		Tools:    clonedTools,
	}
}

func cloneRawJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}

	out := make([]byte, len(raw))
	copy(out, raw)
	return out
}

type fakeToolCatalog struct {
	defs      []llm.ToolDef
	toolByKey map[string]tools.Tool
}

func (f *fakeToolCatalog) ToolDefs() []llm.ToolDef {
	out := make([]llm.ToolDef, len(f.defs))
	for i, tool := range f.defs {
		out[i] = llm.ToolDef{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  cloneRawJSON(tool.Parameters),
		}
	}
	return out
}

func (f *fakeToolCatalog) Get(name string) tools.Tool {
	if f.toolByKey == nil {
		return nil
	}
	return f.toolByKey[name]
}

type fakeTool struct {
	name      string
	execute   func(context.Context, json.RawMessage) (string, error)
	executeV2 func(context.Context, json.RawMessage, string) (string, error)
	callCount int
	lastArgs  json.RawMessage
	lastTrace string
}

func (f *fakeTool) Name() string {
	return f.name
}

func (f *fakeTool) Description() string {
	return "fake tool"
}

func (f *fakeTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}

func (f *fakeTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	f.callCount++
	f.lastArgs = cloneRawJSON(argsJSON)
	f.lastTrace = traceID
	if f.executeV2 != nil {
		return f.executeV2(ctx, argsJSON, traceID)
	}
	if f.execute == nil {
		return "", nil
	}
	return f.execute(ctx, argsJSON)
}

type toolResultPayload struct {
	Status  string `json:"status"`
	Tool    string `json:"tool"`
	TraceID string `json:"trace_id"`
	Output  string `json:"output"`
	Error   string `json:"error"`
}

// decodeToolResult 用于校验 tool_result 的 JSON envelope 结构。
func decodeToolResult(t *testing.T, raw string) toolResultPayload {
	t.Helper()

	var result toolResultPayload
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("decode tool result JSON: %v, raw=%s", err, raw)
	}
	return result
}

func lastMessage(t *testing.T, request llm.CompletionRequest) llm.Message {
	t.Helper()

	if len(request.Messages) == 0 {
		t.Fatal("request has no messages")
	}
	return request.Messages[len(request.Messages)-1]
}

func newTestAgent(completer *fakeCompleter, catalog *fakeToolCatalog, maxTurns int) *Agent {
	return NewAgent(completer, catalog, "system prompt", maxTurns)
}

func TestGetNewMessagesWithPreloadedHistory(t *testing.T) {
	completer := &fakeCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					Text: "follow-up",
				},
				FinishReason: llm.FinishStop,
			},
		},
	}
	catalog := &fakeToolCatalog{}
	preloaded := NewHistoryFromMessages([]llm.Message{
		{Role: llm.RoleSystem, Text: "system prompt"},
		{Role: llm.RoleUser, Text: "old user message"},
		{Role: llm.RoleAssistant, Text: "old assistant message"},
	})

	agent := NewAgentWithHistory(completer, catalog, preloaded, 3)
	if _, err := agent.Run(context.Background(), "new user message"); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	newMessages := agent.GetNewMessages()
	if len(newMessages) != 2 {
		t.Fatalf("unexpected new message count: got %d want %d", len(newMessages), 2)
	}
	if newMessages[0].Role != llm.RoleUser || newMessages[0].Text != "new user message" {
		t.Fatalf("unexpected first new message: %+v", newMessages[0])
	}
	if newMessages[1].Role != llm.RoleAssistant || newMessages[1].Text != "follow-up" {
		t.Fatalf("unexpected second new message: %+v", newMessages[1])
	}
}

// 场景：模型直接 stop，Agent 返回 assistant 文本。
func TestRunStop(t *testing.T) {
	completer := &fakeCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					Text: "done",
				},
				FinishReason: llm.FinishStop,
			},
		},
	}
	catalog := &fakeToolCatalog{}

	agent := newTestAgent(completer, catalog, 3)
	got, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got != "done" {
		t.Fatalf("unexpected output: got %q want %q", got, "done")
	}
	if len(completer.requests) != 1 {
		t.Fatalf("unexpected complete call count: got %d want %d", len(completer.requests), 1)
	}
}

// 场景：tool_calls -> 工具成功 -> 下一轮 stop。
func TestRunToolCallsThenStopWithSuccessEnvelope(t *testing.T) {
	tool := &fakeTool{
		name: "echo",
		execute: func(_ context.Context, _ json.RawMessage) (string, error) {
			return "tool-ok", nil
		},
	}
	completer := &fakeCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{
						{
							ID:        "call-1",
							Name:      "echo",
							Arguments: json.RawMessage(`{"input":"hi"}`),
						},
					},
				},
				FinishReason: llm.FinishToolCalls,
			},
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					Text: "final",
				},
				FinishReason: llm.FinishStop,
			},
		},
	}
	catalog := &fakeToolCatalog{
		defs: []llm.ToolDef{
			{
				Name:       "echo",
				Parameters: json.RawMessage(`{"type":"object"}`),
			},
		},
		toolByKey: map[string]tools.Tool{
			"echo": tool,
		},
	}

	agent := newTestAgent(completer, catalog, 3)
	got, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got != "final" {
		t.Fatalf("unexpected output: got %q want %q", got, "final")
	}
	if tool.callCount != 1 {
		t.Fatalf("unexpected tool call count: got %d want %d", tool.callCount, 1)
	}

	if len(completer.requests) != 2 {
		t.Fatalf("unexpected complete call count: got %d want %d", len(completer.requests), 2)
	}
	toolMsg := lastMessage(t, completer.requests[1])
	if toolMsg.Role != llm.RoleTool {
		t.Fatalf("unexpected last role: got %q want %q", toolMsg.Role, llm.RoleTool)
	}
	if toolMsg.ToolCallID != "call-1" {
		t.Fatalf("unexpected tool call id: got %q want %q", toolMsg.ToolCallID, "call-1")
	}

	toolResult := decodeToolResult(t, toolMsg.Text)
	if toolResult.Status != "success" {
		t.Fatalf("unexpected status: got %q want %q", toolResult.Status, "success")
	}
	if toolResult.Tool != "echo" {
		t.Fatalf("unexpected tool: got %q want %q", toolResult.Tool, "echo")
	}
	if toolResult.Output != "tool-ok" {
		t.Fatalf("unexpected output: got %q want %q", toolResult.Output, "tool-ok")
	}
	if toolResult.Error != "" {
		t.Fatalf("unexpected error: got %q want empty", toolResult.Error)
	}
	if strings.TrimSpace(toolResult.TraceID) == "" {
		t.Fatal("trace_id should not be empty")
	}
}

// 场景：工具不存在时写 error envelope，并继续下一轮。
func TestRunToolNotFoundWritesErrorEnvelope(t *testing.T) {
	completer := &fakeCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{
						{
							ID:        "call-1",
							Name:      "missing_tool",
							Arguments: json.RawMessage(`{}`),
						},
					},
				},
				FinishReason: llm.FinishToolCalls,
			},
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					Text: "done",
				},
				FinishReason: llm.FinishStop,
			},
		},
	}
	catalog := &fakeToolCatalog{}

	agent := newTestAgent(completer, catalog, 3)
	got, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got != "done" {
		t.Fatalf("unexpected output: got %q want %q", got, "done")
	}

	toolMsg := lastMessage(t, completer.requests[1])
	toolResult := decodeToolResult(t, toolMsg.Text)
	if toolResult.Status != "error" {
		t.Fatalf("unexpected status: got %q want %q", toolResult.Status, "error")
	}
	if toolResult.Tool != "missing_tool" {
		t.Fatalf("unexpected tool: got %q want %q", toolResult.Tool, "missing_tool")
	}
	if !strings.Contains(toolResult.Error, "not found") {
		t.Fatalf("unexpected error text: %q", toolResult.Error)
	}
}

// 场景：工具执行报错时写 error envelope，并继续下一轮。
func TestRunToolExecuteErrorWritesErrorEnvelope(t *testing.T) {
	tool := &fakeTool{
		name: "boom",
		execute: func(_ context.Context, _ json.RawMessage) (string, error) {
			return "", errors.New("boom")
		},
	}
	completer := &fakeCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{
						{
							ID:        "call-1",
							Name:      "boom",
							Arguments: json.RawMessage(`{}`),
						},
					},
				},
				FinishReason: llm.FinishToolCalls,
			},
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					Text: "done",
				},
				FinishReason: llm.FinishStop,
			},
		},
	}
	catalog := &fakeToolCatalog{
		toolByKey: map[string]tools.Tool{
			"boom": tool,
		},
	}

	agent := newTestAgent(completer, catalog, 3)
	got, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got != "done" {
		t.Fatalf("unexpected output: got %q want %q", got, "done")
	}

	toolMsg := lastMessage(t, completer.requests[1])
	toolResult := decodeToolResult(t, toolMsg.Text)
	if toolResult.Status != "error" {
		t.Fatalf("unexpected status: got %q want %q", toolResult.Status, "error")
	}
	if !strings.Contains(toolResult.Error, "tool \"boom\" error: boom") {
		t.Fatalf("unexpected error text: %q", toolResult.Error)
	}
}

// 场景：未知 finish_reason 返回显式错误，并带 trace_id/turn。
func TestRunUnknownFinishReasonReturnsError(t *testing.T) {
	completer := &fakeCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					Text: "unknown",
				},
				FinishReason: llm.FinishReason("unknown_reason"),
			},
		},
	}
	catalog := &fakeToolCatalog{}

	agent := newTestAgent(completer, catalog, 1)
	_, err := agent.Run(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "unsupported finish_reason") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "trace_id=") {
		t.Fatalf("error should include trace_id: %v", err)
	}
	if !strings.Contains(err.Error(), "turn=0") {
		t.Fatalf("error should include turn: %v", err)
	}
}

// 场景：连续 tool_calls 超过 max turns 时退出。
func TestRunMaxTurnsExceeded(t *testing.T) {
	tool := &fakeTool{
		name: "echo",
		execute: func(_ context.Context, _ json.RawMessage) (string, error) {
			return "ok", nil
		},
	}
	completer := &fakeCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{
						{
							ID:        "call-1",
							Name:      "echo",
							Arguments: json.RawMessage(`{}`),
						},
					},
				},
				FinishReason: llm.FinishToolCalls,
			},
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{
						{
							ID:        "call-2",
							Name:      "echo",
							Arguments: json.RawMessage(`{}`),
						},
					},
				},
				FinishReason: llm.FinishToolCalls,
			},
		},
	}
	catalog := &fakeToolCatalog{
		toolByKey: map[string]tools.Tool{
			"echo": tool,
		},
	}

	agent := newTestAgent(completer, catalog, 2)
	_, err := agent.Run(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "max turns exceeded: 2") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "trace_id=") {
		t.Fatalf("error should include trace_id: %v", err)
	}
}

// 场景：arguments 非 JSON object 时，不执行工具并写 error envelope。
func TestRunInvalidToolArgumentsWritesErrorEnvelope(t *testing.T) {
	tool := &fakeTool{name: "echo"}
	completer := &fakeCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{
						{
							ID:        "call-1",
							Name:      "echo",
							Arguments: json.RawMessage(`["not-an-object"]`),
						},
					},
				},
				FinishReason: llm.FinishToolCalls,
			},
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					Text: "done",
				},
				FinishReason: llm.FinishStop,
			},
		},
	}
	catalog := &fakeToolCatalog{
		toolByKey: map[string]tools.Tool{
			"echo": tool,
		},
	}

	agent := newTestAgent(completer, catalog, 2)
	got, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got != "done" {
		t.Fatalf("unexpected output: got %q want %q", got, "done")
	}
	if tool.callCount != 0 {
		t.Fatalf("tool should not be called for invalid arguments, got %d calls", tool.callCount)
	}

	toolMsg := lastMessage(t, completer.requests[1])
	toolResult := decodeToolResult(t, toolMsg.Text)
	if toolResult.Status != "error" {
		t.Fatalf("unexpected status: got %q want %q", toolResult.Status, "error")
	}
	if !strings.Contains(toolResult.Error, "must be a JSON object") {
		t.Fatalf("unexpected error: %q", toolResult.Error)
	}
}

func TestRunWithTraceIDPassesTraceToTool(t *testing.T) {
	tool := &fakeTool{
		name: "echo",
		executeV2: func(_ context.Context, _ json.RawMessage, traceID string) (string, error) {
			if traceID != "trace-propagation" {
				t.Fatalf("unexpected trace id: got %q want %q", traceID, "trace-propagation")
			}
			return "ok", nil
		},
	}
	completer := &fakeCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{
						{
							ID:        "call-1",
							Name:      "echo",
							Arguments: json.RawMessage(`{}`),
						},
					},
				},
				FinishReason: llm.FinishToolCalls,
			},
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					Text: "done",
				},
				FinishReason: llm.FinishStop,
			},
		},
	}
	catalog := &fakeToolCatalog{
		toolByKey: map[string]tools.Tool{
			"echo": tool,
		},
	}

	agent := newTestAgent(completer, catalog, 3)
	got, err := agent.RunWithTraceID(context.Background(), "hello", "trace-propagation")
	if err != nil {
		t.Fatalf("RunWithTraceID returned error: %v", err)
	}
	if got != "done" {
		t.Fatalf("unexpected output: got %q want %q", got, "done")
	}
}

func TestRunMissingToolCallIDReturnsError(t *testing.T) {
	tool := &fakeTool{name: "echo"}
	completer := &fakeCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{
						{
							ID:        "",
							Name:      "echo",
							Arguments: json.RawMessage(`{}`),
						},
					},
				},
				FinishReason: llm.FinishToolCalls,
			},
		},
	}
	catalog := &fakeToolCatalog{
		toolByKey: map[string]tools.Tool{
			"echo": tool,
		},
	}

	agent := newTestAgent(completer, catalog, 2)
	_, err := agent.Run(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "tool_call.id is empty") {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.callCount != 0 {
		t.Fatalf("tool should not be called when id is missing, got %d calls", tool.callCount)
	}
}
