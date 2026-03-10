package app

import (
	"context"

	"ghost-os/bridge/agent"
)

type streamTerminalBuffer struct {
	sink   agent.EventSink
	events []agent.AgentEvent
}

func newStreamTerminalBuffer(sink agent.EventSink) *streamTerminalBuffer {
	return &streamTerminalBuffer{sink: ensureEventSink(sink)}
}

func (b *streamTerminalBuffer) Emit(ctx context.Context, event agent.AgentEvent) error {
	if b == nil {
		return nil
	}
	if isTerminalStreamEvent(event.Type) {
		b.events = append(b.events, event)
		return nil
	}
	return b.sink.Emit(ctx, event)
}

func (b *streamTerminalBuffer) Flush(ctx context.Context) error {
	if b == nil {
		return nil
	}
	for _, event := range b.events {
		if err := b.sink.Emit(ctx, event); err != nil {
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

func isTerminalStreamEvent(eventType agent.EventType) bool {
	switch eventType {
	case agent.EventMessage, agent.EventDone:
		return true
	default:
		return false
	}
}
