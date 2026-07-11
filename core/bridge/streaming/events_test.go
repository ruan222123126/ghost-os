package streaming

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestNopSinkEmitIsNoop(t *testing.T) {
	var sink Sink = NopSink{}
	event, err := sink.Emit(context.Background(), Event{TraceID: "trace-1"})
	if err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}
	if event.TraceID != "trace-1" {
		t.Fatalf("unexpected passthrough event trace_id: got %q want %q", event.TraceID, "trace-1")
	}
}

func TestNewEventNormalizesContractFields(t *testing.T) {
	payload := map[string]any{"message": "ok"}
	event, err := NewEvent(" trace-1 ", " session-1 ", 3, " step-1 ", EventMessage, payload)
	if err != nil {
		t.Fatalf("NewEvent returned error: %v", err)
	}

	if event.TraceID != "trace-1" {
		t.Fatalf("unexpected trace_id: got %q want %q", event.TraceID, "trace-1")
	}
	if event.SessionID != "session-1" {
		t.Fatalf("unexpected session_id: got %q want %q", event.SessionID, "session-1")
	}
	if event.Turn != 3 {
		t.Fatalf("unexpected turn: got %d want %d", event.Turn, 3)
	}
	if event.StepID != "step-1" {
		t.Fatalf("unexpected step_id: got %q want %q", event.StepID, "step-1")
	}
	if event.Type != EventMessage {
		t.Fatalf("unexpected event type: got %q want %q", event.Type, EventMessage)
	}
	gotPayload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", event.Payload)
	}
	if !reflect.DeepEqual(gotPayload, payload) {
		t.Fatalf("unexpected payload passthrough: got %#v want %#v", gotPayload, payload)
	}
	if event.At.IsZero() {
		t.Fatal("expected NewEvent to stamp event time")
	}
}

func TestNewEventRejectsNegativeTurn(t *testing.T) {
	_, err := NewEvent("trace-1", "session-1", -1, "", EventRunStarted, nil)
	if err == nil {
		t.Fatal("expected error for negative turn")
	}
	if !strings.Contains(err.Error(), "turn must be >= 0") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewEventAllowsEmptyTraceID(t *testing.T) {
	event, err := NewEvent("   ", "session-1", 0, "", EventRunStarted, nil)
	if err != nil {
		t.Fatalf("NewEvent returned error: %v", err)
	}
	if event.TraceID != "" {
		t.Fatalf("unexpected trace_id: got %q want empty", event.TraceID)
	}
}

func TestFormatEventID(t *testing.T) {
	got, err := FormatEventID("trace-123", 1)
	if err != nil {
		t.Fatalf("FormatEventID returned error: %v", err)
	}
	if want := "trace-123:000001"; got != want {
		t.Fatalf("unexpected event id: got %q want %q", got, want)
	}
}

func TestFormatEventIDRejectsNegativeSequence(t *testing.T) {
	_, err := FormatEventID("trace-123", -1)
	if err == nil {
		t.Fatal("expected error for negative sequence")
	}
	if !strings.Contains(err.Error(), "sequence must be >= 0") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStepIDFormatting(t *testing.T) {
	gotAssistant, err := AssistantStepID(1)
	if err != nil {
		t.Fatalf("AssistantStepID returned error: %v", err)
	}
	if want := "turn-0001-assistant"; gotAssistant != want {
		t.Fatalf("unexpected assistant step id: got %q want %q", gotAssistant, want)
	}
	gotTool, err := ToolStepID(1, 0)
	if err != nil {
		t.Fatalf("ToolStepID returned error: %v", err)
	}
	if want := "turn-0001-tool-0000"; gotTool != want {
		t.Fatalf("unexpected tool step id: got %q want %q", gotTool, want)
	}
}

func TestStepIDFormattingRejectsNegativeIndices(t *testing.T) {
	_, err := AssistantStepID(-1)
	if err == nil {
		t.Fatal("expected error for negative turn")
	}
	if !strings.Contains(err.Error(), "turn must be >= 0") {
		t.Fatalf("unexpected assistant error: %v", err)
	}

	_, err = ToolStepID(1, -1)
	if err == nil {
		t.Fatal("expected error for negative tool index")
	}
	if !strings.Contains(err.Error(), "tool_index must be >= 0") {
		t.Fatalf("unexpected tool error: %v", err)
	}

	_, err = ToolStepID(-1, 0)
	if err == nil {
		t.Fatal("expected error for negative turn")
	}
	if !strings.Contains(err.Error(), "turn must be >= 0") {
		t.Fatalf("unexpected tool error: %v", err)
	}
}
