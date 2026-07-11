package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
)

func TestRunStreamEmitsToolEventsInOrder(t *testing.T) {
	tool := newStaticTool("echo", "ok")
	completer := newFakeCompleter(
		newToolCallsResponse(newToolCall("call-echo-1", "echo", `{"message":"hello"}`)),
		newStopResponse("done"),
	)
	catalog := newFakeToolCatalog(tool)
	sink := newRecordingEventSink()

	agent := newTestAgent(completer, catalog, 3)
	got, err := agent.RunMessageStreamWithTraceID(
		context.Background(),
		llm.Message{Role: llm.RoleUser, Text: "hello"},
		"trace-stream",
		sink,
	)
	if err != nil {
		t.Fatalf("RunMessageStreamWithTraceID returned error: %v", err)
	}
	if got != "done" {
		t.Fatalf("unexpected output: got %q want %q", got, "done")
	}
	if len(sink.events) != 5 {
		t.Fatalf("unexpected event count: got %d want %d", len(sink.events), 5)
	}
	wantOrder := []streaming.EventType{
		streaming.EventRunStarted,
		streaming.EventToolCallStarted,
		streaming.EventToolCallFinished,
		streaming.EventMessage,
		streaming.EventDone,
	}
	for index, want := range wantOrder {
		if sink.events[index].Type != want {
			t.Fatalf("unexpected event[%d]: got %q want %q", index, sink.events[index].Type, want)
		}
	}
	toolStepID, err := streaming.ToolStepID(0, 0)
	if err != nil {
		t.Fatalf("ToolStepID returned error: %v", err)
	}
	if sink.events[1].StepID != toolStepID || sink.events[2].StepID != toolStepID {
		t.Fatalf("unexpected step ids: got %q and %q", sink.events[1].StepID, sink.events[2].StepID)
	}
	payload, ok := sink.events[2].Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", sink.events[2].Payload)
	}
	if payload["status"] != "success" {
		t.Fatalf("unexpected finish status: got %v want %q", payload["status"], "success")
	}
}

func TestRunStreamEmitsAwaitingHumanEvent(t *testing.T) {
	tool := newAwaitingHumanTool("approval_gate", "q-123", "Which database?")
	completer := newFakeCompleter(
		newToolCallsResponse(newToolCall("call-ask-1", "approval_gate", `{"prompt":"Which database?"}`)),
	)
	catalog := newFakeToolCatalog(tool)
	sink := newRecordingEventSink()

	agent := newTestAgent(completer, catalog, 3)
	_, err := agent.RunMessageStreamWithTraceID(
		context.Background(),
		llm.Message{Role: llm.RoleUser, Text: "pick db"},
		"trace-await",
		sink,
	)
	if err == nil {
		t.Fatal("expected awaiting-human error")
	}
	var awaitingErr *ErrAwaitingHuman
	if !errors.As(err, &awaitingErr) {
		t.Fatalf("expected ErrAwaitingHuman, got %v", err)
	}
	if len(sink.events) != 4 {
		t.Fatalf("unexpected event count: got %d want %d", len(sink.events), 4)
	}
	if sink.events[3].Type != streaming.EventAwaitingHuman {
		t.Fatalf("unexpected final event type: got %q want %q", sink.events[3].Type, streaming.EventAwaitingHuman)
	}
	payload, ok := sink.events[3].Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", sink.events[3].Payload)
	}
	if payload["question_id"] != "q-123" {
		t.Fatalf("unexpected question id: got %v want %q", payload["question_id"], "q-123")
	}
}

func TestRunStreamMixedValidAndInvalidToolCallsKeepDistinctStepIDs(t *testing.T) {
	tool := newStaticTool("echo", "ok")
	completer := newFakeCompleter(
		newToolCallsResponse(
			newToolCall("", "echo", `{}`),
			newToolCall("call-2", "echo", `{"input":"hi"}`),
		),
		newStopResponse("done"),
	)
	sink := newRecordingEventSink()
	agent := newTestAgent(completer, newFakeToolCatalog(tool), 3)

	got, err := agent.RunMessageStreamWithTraceID(
		context.Background(),
		llm.Message{Role: llm.RoleUser, Text: "hello"},
		"trace-mixed",
		sink,
	)
	if err != nil {
		t.Fatalf("RunMessageStreamWithTraceID returned error: %v", err)
	}
	if got != "done" {
		t.Fatalf("unexpected output: got %q want %q", got, "done")
	}
	if len(sink.events) != 7 {
		t.Fatalf("unexpected event count: got %d want %d", len(sink.events), 7)
	}

	invalidStepID, err := streaming.ToolStepID(0, 0)
	if err != nil {
		t.Fatalf("ToolStepID returned error: %v", err)
	}
	validStepID, err := streaming.ToolStepID(0, 1)
	if err != nil {
		t.Fatalf("ToolStepID returned error: %v", err)
	}
	if sink.events[1].StepID != invalidStepID || sink.events[2].StepID != invalidStepID {
		t.Fatalf("unexpected invalid tool step ids: got %q and %q", sink.events[1].StepID, sink.events[2].StepID)
	}
	if sink.events[3].StepID != validStepID || sink.events[4].StepID != validStepID {
		t.Fatalf("unexpected valid tool step ids: got %q and %q", sink.events[3].StepID, sink.events[4].StepID)
	}
	if sink.events[1].StepID == sink.events[3].StepID {
		t.Fatalf("mixed valid/invalid tool events should not share step id: %q", sink.events[1].StepID)
	}
	if got := eventToolCallID(t, sink.events[3]); got != "call-2" {
		t.Fatalf("unexpected valid tool_call_started id: got %q want %q", got, "call-2")
	}
	if got := eventToolCallID(t, sink.events[4]); got != "call-2" {
		t.Fatalf("unexpected valid tool_call_finished id: got %q want %q", got, "call-2")
	}
}

func TestRunStreamEmitsErrorEventOnFatalFailure(t *testing.T) {
	completer := newFakeCompleter()
	catalog := newFakeToolCatalog()
	sink := newRecordingEventSink()

	agent := newTestAgent(completer, catalog, 1)
	_, err := agent.RunMessageStreamWithTraceID(
		context.Background(),
		llm.Message{Role: llm.RoleUser, Text: "hello"},
		"trace-error",
		sink,
	)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if len(sink.events) != 2 {
		t.Fatalf("unexpected event count: got %d want %d", len(sink.events), 2)
	}
	event := sink.events[1]
	if event.Type != streaming.EventError {
		t.Fatalf("unexpected event type: got %q want %q", event.Type, streaming.EventError)
	}
	assistantStepID, err := streaming.AssistantStepID(0)
	if err != nil {
		t.Fatalf("AssistantStepID returned error: %v", err)
	}
	if event.StepID != assistantStepID {
		t.Fatalf("unexpected step id: got %q want %q", event.StepID, assistantStepID)
	}
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", event.Payload)
	}
	if !strings.Contains(payload["message"].(string), "complete_once") {
		t.Fatalf("unexpected payload message: %q", payload["message"])
	}
}

func TestRunStreamDoesNotEmitErrorEventOnCancellation(t *testing.T) {
	completer := newFakeCompleter(newStopResponse("done"))
	sink := newRecordingEventSink()
	agent := newTestAgent(completer, newFakeToolCatalog(), 1)
	agent.SetBeforeCompletionHook(func(_ context.Context, _ int, _ *History) error {
		return context.Canceled
	})

	_, err := agent.RunMessageStreamWithTraceID(
		context.Background(),
		llm.Message{Role: llm.RoleUser, Text: "hello"},
		"trace-cancel",
		sink,
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("unexpected error: got %v want %v", err, context.Canceled)
	}
	if err.Error() != context.Canceled.Error() {
		t.Fatalf("cancellation should not keep completion wrapper: %v", err)
	}
	if len(sink.events) != 1 {
		t.Fatalf("unexpected event count: got %d want 1", len(sink.events))
	}
	if sink.events[0].Type != streaming.EventRunStarted {
		t.Fatalf("unexpected event type: got %q want %q", sink.events[0].Type, streaming.EventRunStarted)
	}
}

func TestRunStreamUsesStreamingCompleterAndEmitsCompletionDeltas(t *testing.T) {
	completer := &fakeStreamingCompleter{
		streamResponses: []*llm.CompletionResponse{newStopResponse("Hello world")},
		streamDeltas: [][]llm.LLMDelta{
			{
				{Kind: llm.DeltaKindText, Text: "Hello"},
				{Kind: llm.DeltaKindText, Text: " world"},
			},
		},
	}
	sink := newRecordingEventSink()
	agent := newTestAgent(completer, newFakeToolCatalog(), 3)

	got, err := agent.RunMessageStreamWithTraceID(
		context.Background(),
		llm.Message{Role: llm.RoleUser, Text: "hello"},
		"trace-streaming",
		sink,
	)
	if err != nil {
		t.Fatalf("RunMessageStreamWithTraceID returned error: %v", err)
	}
	if got != "Hello world" {
		t.Fatalf("unexpected output: got %q want %q", got, "Hello world")
	}
	if len(completer.streamRequests) != 1 || len(completer.completeRequests) != 0 {
		t.Fatalf("unexpected completer call counts: stream=%d complete=%d", len(completer.streamRequests), len(completer.completeRequests))
	}
	if len(sink.events) != 5 {
		t.Fatalf("unexpected event count: got %d want %d", len(sink.events), 5)
	}
	wantOrder := []streaming.EventType{
		streaming.EventRunStarted,
		streaming.EventCompletionDelta,
		streaming.EventCompletionDelta,
		streaming.EventMessage,
		streaming.EventDone,
	}
	for index, want := range wantOrder {
		if sink.events[index].Type != want {
			t.Fatalf("unexpected event[%d]: got %q want %q", index, sink.events[index].Type, want)
		}
	}
}

func TestRunFallsBackToCompleteWithoutStreamSink(t *testing.T) {
	completer := &fakeStreamingCompleter{
		completeResponses: []*llm.CompletionResponse{newStopResponse("done")},
	}
	agent := newTestAgent(completer, newFakeToolCatalog(), 3)

	got, err := agent.RunWithTraceID(context.Background(), "hello", "trace-sync")
	if err != nil {
		t.Fatalf("RunWithTraceID returned error: %v", err)
	}
	if got != "done" {
		t.Fatalf("unexpected output: got %q want %q", got, "done")
	}
	if len(completer.completeRequests) != 1 || len(completer.streamRequests) != 0 {
		t.Fatalf("unexpected completer call counts: complete=%d stream=%d", len(completer.completeRequests), len(completer.streamRequests))
	}
}

func TestRunStreamFallsBackToCompleteForNonStreamingCompleter(t *testing.T) {
	completer := newFakeCompleter(newStopResponse("fallback"))
	sink := newRecordingEventSink()
	agent := newTestAgent(completer, newFakeToolCatalog(), 3)

	got, err := agent.RunMessageStreamWithTraceID(
		context.Background(),
		llm.Message{Role: llm.RoleUser, Text: "hello"},
		"trace-fallback",
		sink,
	)
	if err != nil {
		t.Fatalf("RunMessageStreamWithTraceID returned error: %v", err)
	}
	if got != "fallback" {
		t.Fatalf("unexpected output: got %q want %q", got, "fallback")
	}
	if len(completer.requests) != 1 {
		t.Fatalf("unexpected complete call count: got %d want %d", len(completer.requests), 1)
	}
	if len(sink.events) != 3 {
		t.Fatalf("unexpected streamed event count: got %d want %d", len(sink.events), 3)
	}
	if sink.events[0].Type != streaming.EventRunStarted || sink.events[1].Type != streaming.EventMessage || sink.events[2].Type != streaming.EventDone {
		t.Fatalf("unexpected streamed events: %+v", sink.events)
	}
}

func eventToolCallID(t *testing.T, event streaming.Event) string {
	t.Helper()
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", event.Payload)
	}
	toolCallID, ok := payload["tool_call_id"].(string)
	if !ok {
		t.Fatalf("unexpected tool_call_id payload: %#v", payload["tool_call_id"])
	}
	return toolCallID
}
