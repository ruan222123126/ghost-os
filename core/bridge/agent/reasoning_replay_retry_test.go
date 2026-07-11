package agent

import (
	"context"
	"testing"

	"ghost-os/bridge/llm"
)

func TestRunRetriesFinalAssistantReasoningReplayError(t *testing.T) {
	tool := newStaticTool("echo", "ok")
	completer := newFakeCompleter(
		newReasoningToolCallsResponse("first plan", newToolCall("call-1", "echo", `{}`)),
		newStopResponse("done"),
		withReasoning("final plan", newStopResponse("done")),
	)

	runAgent := newTestAgent(completer, newFakeToolCatalog(tool), 4)
	got, err := runAgent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("expected retry success, got error: %v", err)
	}
	if got != "done" {
		t.Fatalf("unexpected output: got %q want %q", got, "done")
	}
	if tool.callCount != 1 {
		t.Fatalf("unexpected tool call count: got %d want %d", tool.callCount, 1)
	}
	if len(completer.requests) != 3 {
		t.Fatalf("unexpected completion request count: got %d want %d", len(completer.requests), 3)
	}
}

func TestRunRetriesToolCallReasoningReplayError(t *testing.T) {
	tool := newStaticTool("echo", "ok")
	completer := newFakeCompleter(
		newReasoningToolCallsResponse("first plan", newToolCall("call-1", "echo", `{}`)),
		newToolCallsResponse(newToolCall("call-2", "echo", `{}`)),
		newReasoningToolCallsResponse("second plan", newToolCall("call-2", "echo", `{}`)),
		withReasoning("final plan", newStopResponse("done")),
	)

	runAgent := newTestAgent(completer, newFakeToolCatalog(tool), 5)
	got, err := runAgent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("expected retry success, got error: %v", err)
	}
	if got != "done" {
		t.Fatalf("unexpected output: got %q want %q", got, "done")
	}
	if tool.callCount != 2 {
		t.Fatalf("unexpected tool call count: got %d want %d", tool.callCount, 2)
	}
	if len(completer.requests) != 4 {
		t.Fatalf("unexpected completion request count: got %d want %d", len(completer.requests), 4)
	}
}

func withReasoning(reasoning string, resp *llm.CompletionResponse) *llm.CompletionResponse {
	if resp == nil {
		return nil
	}
	raw := mustReasoningRaw(reasoning)
	cloned := *resp
	cloned.Message = resp.Message
	cloned.Message.ReasoningContent = raw
	return &cloned
}
