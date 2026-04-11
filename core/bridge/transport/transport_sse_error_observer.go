package transport

import (
	"context"
	"strings"
	"sync"

	bridgeorchestration "ghost-os/bridge/orchestration"
	"ghost-os/bridge/streaming"
)

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
