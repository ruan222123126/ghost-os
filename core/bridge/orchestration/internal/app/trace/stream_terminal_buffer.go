package trace

import (
	"context"

	"ghost-os/bridge/streaming"
)

type StreamTerminalBuffer struct {
	sink   streaming.Sink
	events []streaming.Event
}

func NewStreamTerminalBuffer(sink streaming.Sink) *StreamTerminalBuffer {
	return &StreamTerminalBuffer{sink: EnsureEventSink(sink)}
}

func (b *StreamTerminalBuffer) Emit(ctx context.Context, event streaming.Event) (streaming.Event, error) {
	if b == nil {
		return event, nil
	}
	if isTerminalStreamEvent(event.Type) {
		b.events = append(b.events, event)
		return event, nil
	}
	return b.sink.Emit(ctx, event)
}

func (b *StreamTerminalBuffer) Flush(ctx context.Context) error {
	if b == nil {
		return nil
	}
	for _, event := range b.events {
		if _, err := b.sink.Emit(ctx, event); err != nil {
			return err
		}
	}
	b.events = nil
	return nil
}

func (b *StreamTerminalBuffer) Discard() {
	if b == nil {
		return
	}
	b.events = nil
}

func isTerminalStreamEvent(eventType streaming.EventType) bool {
	switch eventType {
	case streaming.EventMessage, streaming.EventDone:
		return true
	default:
		return false
	}
}
