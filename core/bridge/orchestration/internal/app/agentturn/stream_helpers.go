package agentturn

import (
	"context"
	"strings"

	"ghost-os/bridge/streaming"
)

type eventTurnTracker struct {
	sink        streaming.Sink
	maxToolTurn int
}

func newEventTurnTracker(sink streaming.Sink) *eventTurnTracker {
	return &eventTurnTracker{sink: ensureSink(sink), maxToolTurn: -1}
}

func (t *eventTurnTracker) Emit(ctx context.Context, event streaming.Event) (streaming.Event, error) {
	if isToolProgressEvent(event.Type) && event.Turn > t.maxToolTurn {
		t.maxToolTurn = event.Turn
	}
	return t.sink.Emit(ctx, event)
}

func (t *eventTurnTracker) finalAssistantTurn() int {
	if t == nil || t.maxToolTurn < 0 {
		return 0
	}
	return t.maxToolTurn + 1
}

func isToolProgressEvent(eventType streaming.EventType) bool {
	switch eventType {
	case streaming.EventToolCallStarted, streaming.EventToolCallFinished, streaming.EventAwaitingHuman:
		return true
	default:
		return false
	}
}

func ensureSink(sink streaming.Sink) streaming.Sink {
	if sink == nil {
		return streaming.NopSink{}
	}
	return sink
}

func emitStreamEvent(ctx context.Context, sink streaming.Sink, event streaming.Event) error {
	_, err := ensureSink(sink).Emit(ctx, event)
	return err
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
	if err == nil {
		return nil
	}
	payload := map[string]any{"message": err.Error()}
	if trimmedSessionID := strings.TrimSpace(sessionID); trimmedSessionID != "" {
		payload["session_id"] = trimmedSessionID
	}
	if statusCode > 0 {
		payload["code"] = statusCode
	}
	event, newEventErr := streaming.NewEvent(traceID, sessionID, turn, stepID, streaming.EventError, payload)
	if newEventErr != nil {
		return newEventErr
	}
	return emitStreamEvent(ctx, sink, event)
}
