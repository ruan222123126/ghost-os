package agent

import (
	"context"
	"testing"

	"ghost-os/bridge/llm"
)

func TestLLMDeltaBridgeConvertsTextDelta(t *testing.T) {
	sink := &recordingEventSink{}
	bridge := newLLMDeltaBridge(sink, "trace-bridge", 2)

	err := bridge.OnDelta(context.Background(), llm.LLMDelta{
		Kind: llm.DeltaKindText,
		Text: "Hello",
	})
	if err != nil {
		t.Fatalf("OnDelta returned error: %v", err)
	}
	if len(sink.events) != 1 {
		t.Fatalf("unexpected event count: got %d want %d", len(sink.events), 1)
	}
	event := sink.events[0]
	if event.Type != EventCompletionDelta {
		t.Fatalf("unexpected event type: got %q want %q", event.Type, EventCompletionDelta)
	}
	if event.StepID != AssistantStepID(2) {
		t.Fatalf("unexpected step id: got %q want %q", event.StepID, AssistantStepID(2))
	}
	payload := event.Payload.(map[string]any)
	if payload["kind"] != string(llm.DeltaKindText) || payload["text"] != "Hello" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestLLMDeltaBridgeConvertsToolCallDelta(t *testing.T) {
	sink := &recordingEventSink{}
	bridge := newLLMDeltaBridge(sink, "trace-bridge", 1)

	err := bridge.OnDelta(context.Background(), llm.LLMDelta{
		Kind:              llm.DeltaKindToolCallDelta,
		ToolCallIndex:     0,
		ArgumentsFragment: `{"query":`,
	})
	if err != nil {
		t.Fatalf("OnDelta returned error: %v", err)
	}
	payload := sink.events[0].Payload.(map[string]any)
	if payload["kind"] != string(llm.DeltaKindToolCallDelta) {
		t.Fatalf("unexpected kind: got %v want %q", payload["kind"], llm.DeltaKindToolCallDelta)
	}
	if payload["tool_call_index"] != 0 {
		t.Fatalf("unexpected tool_call_index: got %v want %d", payload["tool_call_index"], 0)
	}
	if payload["arguments_fragment"] != `{"query":` {
		t.Fatalf("unexpected arguments fragment: got %v want %q", payload["arguments_fragment"], `{"query":`)
	}
}
