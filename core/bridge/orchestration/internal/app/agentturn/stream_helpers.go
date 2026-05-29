package agentturn

import (
	"context"

	internaltrace "ghost-os/bridge/orchestration/internal/trace"
	"ghost-os/bridge/streaming"
)

type eventTurnTracker struct {
	inner *internaltrace.EventTurnTracker
}

func newEventTurnTracker(sink streaming.Sink) *eventTurnTracker {
	return &eventTurnTracker{inner: internaltrace.NewEventTurnTracker(sink)}
}

func (t *eventTurnTracker) Emit(ctx context.Context, event streaming.Event) (streaming.Event, error) {
	return t.inner.Emit(ctx, event)
}

func (t *eventTurnTracker) finalAssistantTurn() int {
	if t == nil || t.inner == nil {
		return 0
	}
	return t.inner.FinalAssistantTurn()
}

func emitStreamEvent(ctx context.Context, sink streaming.Sink, event streaming.Event) error {
	return internaltrace.EmitStreamEvent(ctx, sink, event)
}

func emitStreamErrorEvent(
	ctx context.Context,
	sink streaming.Sink,
	traceID string,
	turn int,
	stepID string,
	sessionID string,
	statusCode int,
	err error,
) error {
	return internaltrace.EmitStreamErrorEvent(ctx, sink, traceID, turn, stepID, sessionID, statusCode, err)
}
