package orchestration

import (
	"context"
	"strings"
	"sync"

	"ghost-os/bridge/streaming"
)

type sessionStreamBroadcastSink struct {
	sink streaming.Sink
	hub  *sessionPushHub
	mu   sync.Mutex
}

func newSessionStreamBroadcastSink(sink streaming.Sink, hub *sessionPushHub) streaming.Sink {
	if hub == nil {
		return ensureEventSink(sink)
	}
	return &sessionStreamBroadcastSink{
		sink: ensureEventSink(sink),
		hub:  hub,
	}
}

func (s *sessionStreamBroadcastSink) Emit(ctx context.Context, event streaming.Event) (streaming.Event, error) {
	event, err := s.sink.Emit(ctx, event)
	if err != nil {
		return event, err
	}
	s.publish(event)
	return event, nil
}

func (s *sessionStreamBroadcastSink) publish(event streaming.Event) {
	if s == nil || s.hub == nil {
		return
	}

	pushEvent, ok := newSessionPushEventFromAgentEvent(event)
	if !ok {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.hub.Publish(pushEvent)
}

func newSessionPushEventFromAgentEvent(event streaming.Event) (sessionPushEvent, bool) {
	trimmedSessionID := strings.TrimSpace(event.SessionID)
	if trimmedSessionID == "" {
		return sessionPushEvent{}, false
	}

	eventType, ok := mapSessionPushEventType(event.Type)
	if !ok {
		return sessionPushEvent{}, false
	}

	return sessionPushEvent{
		ID:        strings.TrimSpace(event.ID),
		Type:      eventType,
		TraceID:   strings.TrimSpace(event.TraceID),
		SessionID: trimmedSessionID,
		Payload:   event.Payload,
		At:        event.At,
	}, true
}

func mapSessionPushEventType(eventType streaming.EventType) (sessionPushEventType, bool) {
	switch eventType {
	case streaming.EventRunStarted:
		return sessionPushRunStarted, true
	case streaming.EventCompletionDelta:
		return sessionPushCompletionDelta, true
	case streaming.EventToolCallStarted:
		return sessionPushToolCallStarted, true
	case streaming.EventToolCallFinished:
		return sessionPushToolCallFinished, true
	case streaming.EventError:
		return sessionPushError, true
	case streaming.EventDone:
		return sessionPushDone, true
	default:
		return "", false
	}
}
