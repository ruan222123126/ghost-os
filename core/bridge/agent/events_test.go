package agent

import (
	"context"
	"testing"
)

func TestNopSinkEmitIsNoop(t *testing.T) {
	var sink EventSink = nopSink{}
	if err := sink.Emit(context.Background(), AgentEvent{}); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}
}

func TestFormatEventID(t *testing.T) {
	if got, want := FormatEventID("trace-123", 1), "trace-123:000001"; got != want {
		t.Fatalf("unexpected event id: got %q want %q", got, want)
	}
}

func TestStepIDFormatting(t *testing.T) {
	if got, want := AssistantStepID(1), "turn-0001-assistant"; got != want {
		t.Fatalf("unexpected assistant step id: got %q want %q", got, want)
	}
	if got, want := ToolStepID(1, 0), "turn-0001-tool-0000"; got != want {
		t.Fatalf("unexpected tool step id: got %q want %q", got, want)
	}
}
