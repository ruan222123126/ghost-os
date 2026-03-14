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

// 场景：tool_calls -> 工具成功 -> 下一轮 stop。
func TestRunToolCallsThenStopWithSuccessEnvelope(t *testing.T) {
	tool := newStaticTool("echo", "tool-ok")
	completer := newFakeCompleter(
		newToolCallsResponse(newToolCall("call-1", "echo", `{"input":"hi"}`)),
		newStopResponse("final"),
	)
	catalog := newFakeToolCatalog(tool)

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
	completer := newFakeCompleter(
		newToolCallsResponse(newToolCall("call-1", "missing_tool", `{}`)),
		newStopResponse("done"),
	)
	catalog := newFakeToolCatalog()

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
	tool := newErrorTool("boom", errors.New("boom"))
	completer := newFakeCompleter(
		newToolCallsResponse(newToolCall("call-1", "boom", `{}`)),
		newStopResponse("done"),
	)
	catalog := newFakeToolCatalog(tool)

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
	if !strings.Contains(toolResult.Error, `tool "boom" error: boom`) {
		t.Fatalf("unexpected error text: %q", toolResult.Error)
	}
}

// 场景：模型连续返回不可执行 tool_call 时，提前退出并给出明确错误，避免跑满 max turns。
func TestRunRepeatedNonExecutableToolCallsReturnsErrorEarly(t *testing.T) {
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
							Arguments: json.RawMessage(``),
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
							ID:        "call-3",
							Name:      "echo",
							Arguments: json.RawMessage(`{}`),
						},
					},
				},
				FinishReason: llm.FinishToolCalls,
			},
		},
	}
	catalog := &fakeToolCatalog{}

	agent := newTestAgent(completer, catalog, 20)
	_, err := agent.Run(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "repeated non-executable tool_calls") {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), "max turns exceeded") {
		t.Fatalf("should fail early before max turns: %v", err)
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

	assistantMsg := lastMessage(t, completer.requests[1])
	if assistantMsg.Role != llm.RoleAssistant {
		t.Fatalf("unexpected last role: got %q want %q", assistantMsg.Role, llm.RoleAssistant)
	}
	if len(assistantMsg.ToolCalls) != 0 {
		t.Fatalf("invalid tool call should not stay in history: %+v", assistantMsg.ToolCalls)
	}
	if strings.Contains(assistantMsg.Text, `"status":"error"`) {
		t.Fatalf("invalid tool call should not be rewritten as tool result envelope: %q", assistantMsg.Text)
	}
	if !strings.Contains(assistantMsg.Text, "must be a JSON object") {
		t.Fatalf("unexpected assistant error note: %q", assistantMsg.Text)
	}
}

// 场景：arguments 为空时，不执行工具并写 error envelope。
func TestRunEmptyToolArgumentsWritesErrorEnvelope(t *testing.T) {
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
							Arguments: nil,
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
		t.Fatalf("tool should not be called for empty arguments, got %d calls", tool.callCount)
	}

	assistantMsg := lastMessage(t, completer.requests[1])
	if assistantMsg.Role != llm.RoleAssistant {
		t.Fatalf("unexpected last role: got %q want %q", assistantMsg.Role, llm.RoleAssistant)
	}
	if len(assistantMsg.ToolCalls) != 0 {
		t.Fatalf("empty tool call should not stay in history: %+v", assistantMsg.ToolCalls)
	}
	if !strings.Contains(assistantMsg.Text, "tool_call.arguments is empty") {
		t.Fatalf("unexpected assistant error note: %q", assistantMsg.Text)
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

func TestRunMissingToolCallIDWritesErrorEnvelope(t *testing.T) {
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
		t.Fatalf("tool should not be called when id is missing, got %d calls", tool.callCount)
	}

	assistantMsg := lastMessage(t, completer.requests[1])
	if assistantMsg.Role != llm.RoleAssistant {
		t.Fatalf("unexpected last role: got %q want %q", assistantMsg.Role, llm.RoleAssistant)
	}
	if len(assistantMsg.ToolCalls) != 0 {
		t.Fatalf("missing-id tool call should not stay in history: %+v", assistantMsg.ToolCalls)
	}
	if !strings.Contains(assistantMsg.Text, "tool_call.id is empty") {
		t.Fatalf("unexpected assistant error note: %q", assistantMsg.Text)
	}
}

func TestRunMixedValidAndInvalidToolCallsOnlyReplaysValidCalls(t *testing.T) {
	validTool := newStaticTool("echo", "ok")
	completer := newFakeCompleter(
		newToolCallsResponse(
			newToolCall("", "echo", `{}`),
			newToolCall("call-2", "echo", `{"input":"hi"}`),
		),
		newStopResponse("done"),
	)
	agent := newTestAgent(completer, newFakeToolCatalog(validTool), 3)

	got, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got != "done" {
		t.Fatalf("unexpected output: got %q want %q", got, "done")
	}
	if validTool.callCount != 1 {
		t.Fatalf("unexpected tool call count: got %d want %d", validTool.callCount, 1)
	}

	requestMessages := completer.requests[1].Messages
	if len(requestMessages) < 2 {
		t.Fatalf("unexpected replay message count: got %d", len(requestMessages))
	}
	assistantMsg := requestMessages[len(requestMessages)-2]
	if assistantMsg.Role != llm.RoleAssistant {
		t.Fatalf("unexpected replay assistant message: %+v", assistantMsg)
	}
	if len(assistantMsg.ToolCalls) != 1 || assistantMsg.ToolCalls[0].ID != "call-2" {
		t.Fatalf("unexpected replayed tool calls: %+v", assistantMsg.ToolCalls)
	}
	toolMsg := requestMessages[len(requestMessages)-1]
	if toolMsg.Role != llm.RoleTool || toolMsg.ToolCallID != "call-2" {
		t.Fatalf("unexpected replay tool message: %+v", toolMsg)
	}
}

func TestRunAskHumanReturnsAwaitingError(t *testing.T) {
	tool := newAwaitingHumanTool("approval_gate", "q-123", "Which database?")
	completer := newFakeCompleter(
		newToolCallsResponse(newToolCall("call-ask-1", "approval_gate", `{"prompt":"Which database?"}`)),
	)
	catalog := newFakeToolCatalog(tool)

	agent := newTestAgent(completer, catalog, 3)
	_, err := agent.Run(context.Background(), "pick db")
	if err == nil {
		t.Fatal("expected awaiting-human error")
	}
	var awaitingErr *ErrAwaitingHuman
	if !errors.As(err, &awaitingErr) {
		t.Fatalf("expected ErrAwaitingHuman, got: %v", err)
	}
	if awaitingErr.QuestionID != "q-123" {
		t.Fatalf("unexpected question id: got %q want %q", awaitingErr.QuestionID, "q-123")
	}
	if awaitingErr.Prompt != "Which database?" {
		t.Fatalf("unexpected prompt: got %q want %q", awaitingErr.Prompt, "Which database?")
	}

	newMessages := agent.GetNewMessages()
	if len(newMessages) != 2 {
		t.Fatalf("awaiting-human turn should commit round messages: got %d want %d", len(newMessages), 2)
	}
	if newMessages[0].Role != llm.RoleUser || newMessages[0].Text != "pick db" {
		t.Fatalf("unexpected committed user message: %+v", newMessages[0])
	}
	if newMessages[1].Role != llm.RoleAssistant || len(newMessages[1].ToolCalls) != 1 {
		t.Fatalf("unexpected committed assistant message: %+v", newMessages[1])
	}
}

func TestRunBrowserScreenshotKeepsToolRoleWithImageContent(t *testing.T) {
	tool := &fakeTool{
		name: "visual_probe",
		execute: func(_ context.Context, _ json.RawMessage) (string, error) {
			return `{"action":"screenshot","artifact":{"type":"image","vision_path":"/tmp/s.png","vision_mime_type":"image/png","width":100,"height":50,"sha256":"hash","vision_bytes":256}}`, nil
		},
		interpret: func(_ string) tools.ExecuteMeta {
			return tools.ExecuteMeta{
				Content: []llm.ContentPart{
					{
						Type: llm.ContentTypeImage,
						Image: &llm.ImageContent{
							Path:     "/tmp/s.png",
							MimeType: "image/png",
							Width:    100,
							Height:   50,
							SHA256:   "hash",
							Bytes:    256,
						},
					},
				},
			}
		},
	}
	completer := &fakeCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{
						{
							ID:        "call-shot-1",
							Name:      "visual_probe",
							Arguments: json.RawMessage(`{"action":"screenshot"}`),
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
			"visual_probe": tool,
		},
	}

	agent := newTestAgent(completer, catalog, 3)
	got, err := agent.Run(context.Background(), "check")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got != "done" {
		t.Fatalf("unexpected output: got %q want %q", got, "done")
	}
	if len(completer.requests) != 2 {
		t.Fatalf("unexpected complete call count: got %d want %d", len(completer.requests), 2)
	}

	toolMsg := lastMessage(t, completer.requests[1])
	if toolMsg.Role != llm.RoleTool {
		t.Fatalf("unexpected role: got %q want %q", toolMsg.Role, llm.RoleTool)
	}
	if len(toolMsg.Content) != 1 {
		t.Fatalf("unexpected tool content count: got %d want %d", len(toolMsg.Content), 1)
	}
	if toolMsg.Content[0].Image == nil {
		t.Fatal("expected image content on tool message")
	}
	if toolMsg.Content[0].Image.Path != "/tmp/s.png" {
		t.Fatalf("unexpected image path: got %q want %q", toolMsg.Content[0].Image.Path, "/tmp/s.png")
	}
}
