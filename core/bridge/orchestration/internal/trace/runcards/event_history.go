package runcards

import (
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/streaming"
	bridgeTasks "ghost-os/bridge/tasks"
)

const completionDeltaPersistInterval = 300 * time.Millisecond

type cardPersistState struct {
	lastPersistedAt time.Time
	eventSeq        int
}

func newRunCardSourceEvent(
	event streaming.Event,
	fallbackTime time.Time,
	eventID string,
) bridgeTasks.RunCardSourceEvent {
	at := event.At.UTC()
	if at.IsZero() {
		at = fallbackTime.UTC()
	}
	return bridgeTasks.RunCardSourceEvent{
		ID:        strings.TrimSpace(eventID),
		StepID:    strings.TrimSpace(event.StepID),
		TraceID:   strings.TrimSpace(event.TraceID),
		SessionID: strings.TrimSpace(event.SessionID),
		Turn:      event.Turn,
		Type:      strings.TrimSpace(string(event.Type)),
		Payload:   payloadRecord(event.Payload),
		At:        at,
	}
}

func appendRunCardSourceEvent(
	events []bridgeTasks.RunCardSourceEvent,
	next bridgeTasks.RunCardSourceEvent,
) []bridgeTasks.RunCardSourceEvent {
	return append(events, next)
}

func nextRunCardSourceEventID(cardID string, state cardPersistState) string {
	return fmt.Sprintf("%s-event-%06d", strings.TrimSpace(cardID), state.eventSeq)
}

func shouldPersistRunCardEvent(
	eventType streaming.EventType,
	lastPersistedAt time.Time,
	now time.Time,
) bool {
	if eventType != streaming.EventCompletionDelta {
		return true
	}
	if lastPersistedAt.IsZero() {
		return true
	}
	return now.UTC().Sub(lastPersistedAt.UTC()) >= completionDeltaPersistInterval
}
