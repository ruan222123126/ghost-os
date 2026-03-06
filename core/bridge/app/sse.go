package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"ghost-os/bridge/agent"
)

type sseEventSink struct {
	w        http.ResponseWriter
	flusher  http.Flusher
	traceID  string
	sequence int
	mu       sync.Mutex
}

func newSSEEventSink(w http.ResponseWriter, flusher http.Flusher, traceID string) *sseEventSink {
	return &sseEventSink{
		w:       w,
		flusher: flusher,
		traceID: strings.TrimSpace(traceID),
	}
}

func (s *sseEventSink) Emit(ctx context.Context, event agent.AgentEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if strings.TrimSpace(event.TraceID) == "" {
		event.TraceID = s.traceID
	}
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}

	s.sequence++
	event.ID = agent.FormatEventID(event.TraceID, s.sequence)

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(s.w, "id: %s\n", event.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(s.w, "event: %s\n", event.Type); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", data); err != nil {
		return err
	}

	s.flusher.Flush()
	return nil
}

type nopEventSink struct{}

func (nopEventSink) Emit(context.Context, agent.AgentEvent) error { return nil }

func ensureEventSink(sink agent.EventSink) agent.EventSink {
	if sink == nil {
		return nopEventSink{}
	}
	return sink
}

type eventTurnTracker struct {
	sink        agent.EventSink
	maxToolTurn int
}

func newEventTurnTracker(sink agent.EventSink) *eventTurnTracker {
	return &eventTurnTracker{
		sink:        ensureEventSink(sink),
		maxToolTurn: -1,
	}
}

func (t *eventTurnTracker) Emit(ctx context.Context, event agent.AgentEvent) error {
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

func isToolProgressEvent(eventType agent.EventType) bool {
	switch eventType {
	case agent.EventToolCallStarted, agent.EventToolCallFinished, agent.EventAwaitingHuman:
		return true
	default:
		return false
	}
}

func emitStreamEvent(ctx context.Context, sink agent.EventSink, event agent.AgentEvent) error {
	return ensureEventSink(sink).Emit(ctx, event)
}

func emitStreamErrorEvent(
	ctx context.Context,
	sink agent.EventSink,
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

	payload := map[string]any{
		"message": err.Error(),
	}
	if trimmedSessionID := strings.TrimSpace(sessionID); trimmedSessionID != "" {
		payload["session_id"] = trimmedSessionID
	}
	if statusCode > 0 {
		payload["code"] = statusCode
	}

	return emitStreamEvent(ctx, sink, agent.NewEvent(traceID, turn, stepID, agent.EventError, payload))
}
