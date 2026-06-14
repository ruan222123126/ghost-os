package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	bridgeorchestration "ghost-os/bridge/orchestration"
	"ghost-os/bridge/streaming"
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

func (s *sseEventSink) Emit(ctx context.Context, event streaming.Event) (streaming.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return event, ctx.Err()
	default:
	}

	if strings.TrimSpace(event.TraceID) == "" {
		event.TraceID = s.traceID
	}
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}

	s.sequence++
	eventID, err := streaming.FormatEventID(event.TraceID, s.sequence)
	if err != nil {
		return event, err
	}
	event.ID = eventID

	data, err := json.Marshal(event)
	if err != nil {
		return event, err
	}

	if _, err := fmt.Fprintf(s.w, "id: %s\n", event.ID); err != nil {
		return event, err
	}
	if _, err := fmt.Fprintf(s.w, "event: %s\n", event.Type); err != nil {
		return event, err
	}
	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", data); err != nil {
		return event, err
	}

	s.flusher.Flush()
	return event, nil
}

type observedSSEStreamSink struct {
	sink          streaming.Sink
	mu            sync.RWMutex
	hasErrorEvent bool
}

func newObservedSSEStreamSink(sink streaming.Sink) *observedSSEStreamSink {
	if sink == nil {
		sink = streaming.NopSink{}
	}
	return &observedSSEStreamSink{sink: sink}
}

func (s *observedSSEStreamSink) Emit(ctx context.Context, event streaming.Event) (streaming.Event, error) {
	emitted, err := s.sink.Emit(ctx, event)
	if err != nil {
		return emitted, err
	}
	if event.Type == streaming.EventError || emitted.Type == streaming.EventError {
		s.markErrorEvent()
	}
	return emitted, nil
}

func (s *observedSSEStreamSink) shouldEmitFallbackError(err error) bool {
	if err == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.hasErrorEvent
}

func (s *observedSSEStreamSink) markErrorEvent() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hasErrorEvent = true
}

func emitUnhandledStreamError(
	ctx context.Context,
	sink *observedSSEStreamSink,
	traceID string,
	sessionID string,
	err error,
) {
	if sink == nil || !sink.shouldEmitFallbackError(err) {
		return
	}
	event, buildErr := buildFallbackStreamErrorEvent(traceID, sessionID, err)
	if buildErr != nil {
		logAction(traceID, bridgeorchestration.BusActionAgentSend, "error", buildErr)
		return
	}
	if _, emitErr := sink.Emit(ctx, event); emitErr != nil {
		logAction(traceID, bridgeorchestration.BusActionAgentSend, "error", emitErr)
	}
}

func buildFallbackStreamErrorEvent(traceID string, sessionID string, err error) (streaming.Event, error) {
	payload := map[string]any{
		"message": err.Error(),
	}
	if normalizedSessionID := strings.TrimSpace(sessionID); normalizedSessionID != "" {
		payload["session_id"] = normalizedSessionID
	}
	return streaming.NewEvent(traceID, sessionID, 0, "", streaming.EventError, payload)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if normalized := strings.TrimSpace(value); normalized != "" {
			return normalized
		}
	}
	return ""
}

func (t *transport) handleSessionEvents(w http.ResponseWriter, r *http.Request, sessionID string) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	flusher, traceID, ok := prepareSessionEventsResponse(w, r)
	if !ok {
		return
	}
	if err := t.writePendingSessionPushEvent(w, flusher, sessionID); err != nil {
		return
	}

	hub := t.usecases.streams.SessionPushHub()
	if hub == nil {
		writeError(w, http.StatusInternalServerError, "session push hub is not available", traceID)
		return
	}
	ch, unsubscribe := hub.Subscribe(sessionID)
	defer unsubscribe()

	streamSessionPushEvents(r.Context(), w, flusher, ch)
}

func prepareSessionEventsResponse(w http.ResponseWriter, r *http.Request) (http.Flusher, string, bool) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported", "")
		return nil, "", false
	}

	traceID := resolveTraceID("", r)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("X-Trace-ID", traceID)
	return flusher, traceID, true
}

func (t *transport) writePendingSessionPushEvent(
	w http.ResponseWriter,
	flusher http.Flusher,
	sessionID string,
) error {
	if event, ok := t.usecases.streams.PendingQuestionSnapshot(sessionID); ok {
		return writeSessionPushEvent(w, flusher, event)
	}
	return nil
}

func streamSessionPushEvents(
	ctx context.Context,
	w http.ResponseWriter,
	flusher http.Flusher,
	ch <-chan bridgeorchestration.SessionPushEvent,
) {
	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			if err := writeSessionPushEvent(w, flusher, event); err != nil {
				return
			}
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": keep-alive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func writeSessionPushEvent(w http.ResponseWriter, flusher http.Flusher, event bridgeorchestration.SessionPushEvent) error {
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "id: %s\n", event.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\n", event.Type); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}
