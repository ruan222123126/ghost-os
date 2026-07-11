package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
)

func TestRunRejectsToolReplayWhenReasoningDisappearsMidTurn(t *testing.T) {
	tool := newStaticTool("echo", "ok")
	completer := newFakeCompleter(
		newReasoningToolCallsResponse("first plan", newToolCall("call-1", "echo", `{}`)),
		newToolCallsResponse(newToolCall("call-2", "echo", `{}`)),
		newToolCallsResponse(newToolCall("call-2", "echo", `{}`)),
	)

	agent := newTestAgent(completer, newFakeToolCatalog(tool), 4)
	_, err := agent.Run(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected reasoning replay error but got nil")
	}
	if !strings.Contains(err.Error(), "reasoning_content") {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.callCount != 1 {
		t.Fatalf("unexpected tool call count: got %d want %d", tool.callCount, 1)
	}
	if len(completer.requests) != 3 {
		t.Fatalf("unexpected completion request count: got %d want %d", len(completer.requests), 3)
	}
}

func TestRunRejectsFinalAssistantTurnWhenReasoningDisappears(t *testing.T) {
	tool := newStaticTool("echo", "ok")
	completer := newFakeCompleter(
		newReasoningToolCallsResponse("first plan", newToolCall("call-1", "echo", `{}`)),
		newStopResponse("done"),
		newStopResponse("done"),
	)

	agent := newTestAgent(completer, newFakeToolCatalog(tool), 4)
	_, err := agent.Run(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected reasoning replay error but got nil")
	}
	if !strings.Contains(err.Error(), "reasoning_content") {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.callCount != 1 {
		t.Fatalf("unexpected tool call count: got %d want %d", tool.callCount, 1)
	}
	if len(completer.requests) != 3 {
		t.Fatalf("unexpected completion request count: got %d want %d", len(completer.requests), 3)
	}
}

func newReasoningToolCallsResponse(reasoning string, toolCalls ...llm.ToolCall) *llm.CompletionResponse {
	resp := newToolCallsResponse(toolCalls...)
	resp.Message.ReasoningContent = mustReasoningRaw(reasoning)
	return resp
}

func mustReasoningRaw(reasoning string) json.RawMessage {
	raw, err := json.Marshal(reasoning)
	if err != nil {
		panic(err)
	}
	return raw
}
