package trace

import (
	"context"
	"strings"
	"sync"

	"ghost-os/bridge/streaming"
)

type SessionStreamBroadcastSink struct {
	sink streaming.Sink
	hub  *SessionPushHub
	mu   sync.Mutex
}

func NewSessionStreamBroadcastSink(sink streaming.Sink, hub *SessionPushHub) streaming.Sink {
	if hub == nil {
		return EnsureEventSink(sink)
	}
	return &SessionStreamBroadcastSink{
		sink: EnsureEventSink(sink),
		hub:  hub,
	}
}

func (s *SessionStreamBroadcastSink) Emit(ctx context.Context, event streaming.Event) (streaming.Event, error) {
	event, err := s.sink.Emit(ctx, event)
	if err != nil {
		return event, err
	}
	s.publish(event)
	return event, nil
}

func (s *SessionStreamBroadcastSink) publish(event streaming.Event) {
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

func newSessionPushEventFromAgentEvent(event streaming.Event) (SessionPushEvent, bool) {
	trimmedSessionID := strings.TrimSpace(event.SessionID)
	if trimmedSessionID == "" {
		return SessionPushEvent{}, false
	}

	eventType, ok := mapSessionPushEventType(event.Type)
	if !ok {
		return SessionPushEvent{}, false
	}

	return SessionPushEvent{
		ID:        strings.TrimSpace(event.ID),
		Type:      eventType,
		TraceID:   strings.TrimSpace(event.TraceID),
		SessionID: trimmedSessionID,
		Payload:   event.Payload,
		At:        event.At,
	}, true
}

func mapSessionPushEventType(eventType streaming.EventType) (SessionPushEventType, bool) {
	switch eventType {
	case streaming.EventRunStarted:
		return SessionPushRunStarted, true
	case streaming.EventCompletionDelta:
		return SessionPushCompletionDelta, true
	case streaming.EventToolCallStarted:
		return SessionPushToolCallStarted, true
	case streaming.EventToolCallFinished:
		return SessionPushToolCallFinished, true
	case streaming.EventError:
		return SessionPushError, true
	case streaming.EventDone:
		return SessionPushDone, true
	default:
		return "", false
	}
}
