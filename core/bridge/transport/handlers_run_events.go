package transport

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	bridgeorchestration "ghost-os/bridge/orchestration"
)

type resumedAgentStreamEvent struct {
	ID        string                                   `json:"id"`
	StepID    string                                   `json:"step_id"`
	TraceID   string                                   `json:"trace_id"`
	SessionID string                                   `json:"session_id,omitempty"`
	Turn      int                                      `json:"turn"`
	Type      bridgeorchestration.SessionPushEventType `json:"type"`
	Payload   any                                      `json:"payload"`
	At        time.Time                                `json:"at,omitempty"`
}

func (t *transport) handleRunEvents(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	traceID, err := parseRunEventsTraceID(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported", traceID)
		return
	}
	hub := t.usecases.streams.SessionPushHub()
	if hub == nil {
		writeError(w, http.StatusServiceUnavailable, "session push hub is not available", traceID)
		return
	}

	lastEventID := strings.TrimSpace(r.Header.Get("Last-Event-ID"))
	_, replay, notifications, unsubscribe, found := hub.SubscribeTrace(traceID, lastEventID)
	if !found {
		writeError(w, http.StatusNotFound, "run stream is not available", traceID)
		return
	}
	defer unsubscribe()

	prepareSSEHeaders(w, traceID)
	flusher.Flush()
	lastEventID, terminal, err := writeResumedAgentEvents(w, flusher, replay, lastEventID)
	if err != nil || terminal {
		return
	}

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case _, open := <-notifications:
			if !open {
				return
			}
			events := hub.ReplayTraceAfter(traceID, lastEventID)
			var terminal bool
			lastEventID, terminal, err = writeResumedAgentEvents(w, flusher, events, lastEventID)
			if err != nil || terminal {
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

func parseRunEventsTraceID(path string) (string, error) {
	raw := strings.TrimSpace(strings.TrimPrefix(path, "/api/runs/"))
	if !strings.HasSuffix(raw, "/events") {
		return "", errors.New("invalid run events path")
	}
	traceID := strings.TrimSpace(strings.TrimSuffix(raw, "/events"))
	if traceID == "" || strings.Contains(traceID, "/") {
		return "", errors.New("trace id is required")
	}
	return traceID, nil
}

func prepareSSEHeaders(w http.ResponseWriter, traceID string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("X-Trace-ID", traceID)
}

func writeResumedAgentEvents(
	w http.ResponseWriter,
	flusher http.Flusher,
	events []bridgeorchestration.SessionPushEvent,
	lastEventID string,
) (string, bool, error) {
	for _, event := range events {
		streamEvent, ok := toResumedAgentStreamEvent(event)
		if !ok {
			continue
		}
		data, err := json.Marshal(streamEvent)
		if err != nil {
			return lastEventID, false, err
		}
		if _, err := fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", streamEvent.ID, streamEvent.Type, data); err != nil {
			return lastEventID, false, err
		}
		flusher.Flush()
		lastEventID = streamEvent.ID
		if isResumedAgentTerminalEvent(streamEvent.Type) {
			return lastEventID, true, nil
		}
	}
	return lastEventID, false, nil
}

func toResumedAgentStreamEvent(event bridgeorchestration.SessionPushEvent) (resumedAgentStreamEvent, bool) {
	eventType := event.Type
	switch eventType {
	case bridgeorchestration.SessionPushRunStarted,
		bridgeorchestration.SessionPushCompletionDelta,
		bridgeorchestration.SessionPushToolCallStarted,
		bridgeorchestration.SessionPushToolCallFinished,
		bridgeorchestration.SessionPushAwaitingHuman,
		bridgeorchestration.SessionPushError,
		bridgeorchestration.SessionPushDone:
	default:
		return resumedAgentStreamEvent{}, false
	}

	return resumedAgentStreamEvent{
		ID:        event.ID,
		StepID:    "",
		TraceID:   event.TraceID,
		SessionID: event.SessionID,
		Turn:      0,
		Type:      eventType,
		Payload:   event.Payload,
		At:        event.At,
	}, true
}

func isResumedAgentTerminalEvent(eventType bridgeorchestration.SessionPushEventType) bool {
	return eventType == bridgeorchestration.SessionPushAwaitingHuman ||
		eventType == bridgeorchestration.SessionPushError ||
		eventType == bridgeorchestration.SessionPushDone
}
