package agent

import (
	"context"
	"testing"

	"ghost-os/bridge/llm"
)

func TestRunStreamRetriesReasoningReplayErrorWithoutCompletionDelta(t *testing.T) {
	completer := &fakeStreamingCompleter{
		streamResponses: []*llm.CompletionResponse{
			newReasoningToolCallsResponse("first plan", newToolCall("call-1", "echo", `{}`)),
			newStopResponse("done"),
			withReasoning("final plan", newStopResponse("done")),
		},
		streamDeltas: [][]llm.LLMDelta{{}, {}, {}},
	}
	sink := newRecordingEventSink()
	runAgent := newTestAgent(completer, newFakeToolCatalog(newStaticTool("echo", "ok")), 4)

	got, err := runAgent.RunMessageStreamWithTraceID(context.Background(), llm.Message{Role: llm.RoleUser, Text: "hello"}, "trace-stream", sink)
	if err != nil {
		t.Fatalf("expected retry success, got error: %v", err)
	}
	if got != "done" {
		t.Fatalf("unexpected output: got %q want %q", got, "done")
	}
	if len(completer.streamRequests) != 3 {
		t.Fatalf("unexpected stream request count: got %d want %d", len(completer.streamRequests), 3)
	}
}

func TestRunStreamDoesNotRetryReasoningReplayErrorAfterCompletionDelta(t *testing.T) {
	completer := &fakeStreamingCompleter{
		streamResponses: []*llm.CompletionResponse{
			newReasoningToolCallsResponse("first plan", newToolCall("call-1", "echo", `{}`)),
			newStopResponse("done"),
			withReasoning("final plan", newStopResponse("done")),
		},
		streamDeltas: [][]llm.LLMDelta{
			{},
			{{Kind: llm.DeltaKindText, Text: "done"}},
			{},
		},
	}
	sink := newRecordingEventSink()
	runAgent := newTestAgent(completer, newFakeToolCatalog(newStaticTool("echo", "ok")), 4)

	if _, err := runAgent.RunMessageStreamWithTraceID(context.Background(), llm.Message{Role: llm.RoleUser, Text: "hello"}, "trace-stream", sink); err == nil {
		t.Fatal("expected reasoning replay error without retry")
	}
	if len(completer.streamRequests) != 2 {
		t.Fatalf("unexpected stream request count: got %d want %d", len(completer.streamRequests), 2)
	}
}
