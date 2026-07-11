package trace

import (
	"context"
	"testing"

	"ghost-os/bridge/streaming"
)

func TestStreamTerminalBufferBuffersTerminalEventsUntilFlush(t *testing.T) {
	recording := &recordingAppEventSink{}
	buffer := NewStreamTerminalBuffer(recording)
	ctx := context.Background()

	if _, err := buffer.Emit(ctx, mustAppEvent(t, "trace-buffer", "", 0, "", streaming.EventRunStarted, nil)); err != nil {
		t.Fatalf("emit run_started: %v", err)
	}
	if _, err := buffer.Emit(ctx, mustAppEvent(t, "trace-buffer", "", 0, mustAppAssistantStepID(t, 0), streaming.EventMessage, map[string]any{"text": "done"})); err != nil {
		t.Fatalf("emit message: %v", err)
	}
	if _, err := buffer.Emit(ctx, mustAppEvent(t, "trace-buffer", "", 0, "", streaming.EventDone, map[string]any{"session_ended": false})); err != nil {
		t.Fatalf("emit done: %v", err)
	}

	if len(recording.events) != 1 {
		t.Fatalf("unexpected pre-flush event count: got %d want %d", len(recording.events), 1)
	}
	if recording.events[0].Type != streaming.EventRunStarted {
		t.Fatalf("unexpected pre-flush event type: got %q want %q", recording.events[0].Type, streaming.EventRunStarted)
	}

	if err := buffer.Flush(ctx); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if len(recording.events) != 3 {
		t.Fatalf("unexpected flushed event count: got %d want %d", len(recording.events), 3)
	}
	if recording.events[1].Type != streaming.EventMessage || recording.events[2].Type != streaming.EventDone {
		t.Fatalf("unexpected flushed event order: got %q then %q", recording.events[1].Type, recording.events[2].Type)
	}
}

func TestStreamTerminalBufferDiscardDropsBufferedTerminalEvents(t *testing.T) {
	recording := &recordingAppEventSink{}
	buffer := NewStreamTerminalBuffer(recording)

	if _, err := buffer.Emit(context.Background(), mustAppEvent(t, "trace-buffer", "", 0, mustAppAssistantStepID(t, 0), streaming.EventMessage, map[string]any{"text": "done"})); err != nil {
		t.Fatalf("emit message: %v", err)
	}

	buffer.Discard()
	if err := buffer.Flush(context.Background()); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if len(recording.events) != 0 {
		t.Fatalf("discard should drop buffered terminal events, got %d", len(recording.events))
	}
}

type recordingAppEventSink struct {
	events []streaming.Event
}

func (s *recordingAppEventSink) Emit(_ context.Context, event streaming.Event) (streaming.Event, error) {
	s.events = append(s.events, event)
	return event, nil
}

func mustAppEvent(
	t *testing.T,
	traceID string,
	sessionID string,
	turn int,
	stepID string,
	eventType streaming.EventType,
	payload any,
) streaming.Event {
	t.Helper()

	event, err := streaming.NewEvent(traceID, sessionID, turn, stepID, eventType, payload)
	if err != nil {
		t.Fatalf("NewEvent returned error: %v", err)
	}
	return event
}

func mustAppAssistantStepID(t *testing.T, turn int) string {
	t.Helper()

	stepID, err := streaming.AssistantStepID(turn)
	if err != nil {
		t.Fatalf("AssistantStepID returned error: %v", err)
	}
	return stepID
}
