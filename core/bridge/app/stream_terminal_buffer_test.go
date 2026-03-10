package app

import (
	"context"
	"testing"

	"ghost-os/bridge/agent"
)

type recordingAppEventSink struct {
	events []agent.AgentEvent
}

func (s *recordingAppEventSink) Emit(_ context.Context, event agent.AgentEvent) error {
	s.events = append(s.events, event)
	return nil
}

func TestStreamTerminalBufferBuffersTerminalEventsUntilFlush(t *testing.T) {
	recording := &recordingAppEventSink{}
	buffer := newStreamTerminalBuffer(recording)
	ctx := context.Background()

	if err := buffer.Emit(ctx, agent.NewEvent("trace-buffer", 0, "", agent.EventRunStarted, nil)); err != nil {
		t.Fatalf("emit run_started: %v", err)
	}
	if err := buffer.Emit(ctx, agent.NewEvent("trace-buffer", 0, agent.AssistantStepID(0), agent.EventMessage, map[string]any{"text": "done"})); err != nil {
		t.Fatalf("emit message: %v", err)
	}
	if err := buffer.Emit(ctx, agent.NewEvent("trace-buffer", 0, "", agent.EventDone, map[string]any{"session_ended": false})); err != nil {
		t.Fatalf("emit done: %v", err)
	}

	if len(recording.events) != 1 {
		t.Fatalf("unexpected pre-flush event count: got %d want %d", len(recording.events), 1)
	}
	if recording.events[0].Type != agent.EventRunStarted {
		t.Fatalf("unexpected pre-flush event type: got %q want %q", recording.events[0].Type, agent.EventRunStarted)
	}

	if err := buffer.Flush(ctx); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if len(recording.events) != 3 {
		t.Fatalf("unexpected flushed event count: got %d want %d", len(recording.events), 3)
	}
	if recording.events[1].Type != agent.EventMessage || recording.events[2].Type != agent.EventDone {
		t.Fatalf("unexpected flushed event order: got %q then %q", recording.events[1].Type, recording.events[2].Type)
	}
}

func TestStreamTerminalBufferDiscardDropsBufferedTerminalEvents(t *testing.T) {
	recording := &recordingAppEventSink{}
	buffer := newStreamTerminalBuffer(recording)

	if err := buffer.Emit(context.Background(), agent.NewEvent("trace-buffer", 0, agent.AssistantStepID(0), agent.EventMessage, map[string]any{"text": "done"})); err != nil {
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
