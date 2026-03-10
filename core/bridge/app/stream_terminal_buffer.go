package app

import (
	"context"

	"ghost-os/bridge/streaming"
)

type streamTerminalBuffer struct {
	sink   streaming.Sink
	events []streaming.Event
}

func newStreamTerminalBuffer(sink streaming.Sink) *streamTerminalBuffer {
	return &streamTerminalBuffer{sink: ensureEventSink(sink)}
}

func (b *streamTerminalBuffer) Emit(ctx context.Context, event streaming.Event) (streaming.Event, error) {
	if b == nil {
		return event, nil
	}
	if isTerminalStreamEvent(event.Type) {
		b.events = append(b.events, event)
		return event, nil
	}
	return b.sink.Emit(ctx, event)
}

func (b *streamTerminalBuffer) Flush(ctx context.Context) error {
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

func (b *streamTerminalBuffer) Discard() {
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
