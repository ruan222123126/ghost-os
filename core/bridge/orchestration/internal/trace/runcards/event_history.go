package runcards

import (
	"strings"
	"time"

	"ghost-os/bridge/streaming"
	bridgeTasks "ghost-os/bridge/tasks"
)

const completionDeltaPersistInterval = 300 * time.Millisecond

type cardPersistState struct {
	lastPersistedAt time.Time
}

func newRunCardSourceEvent(
	event streaming.Event,
	fallbackTime time.Time,
) bridgeTasks.RunCardSourceEvent {
	at := event.At.UTC()
	if at.IsZero() {
		at = fallbackTime.UTC()
	}
	return bridgeTasks.RunCardSourceEvent{
		ID:        strings.TrimSpace(event.ID),
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
	if len(events) == 0 {
		return []bridgeTasks.RunCardSourceEvent{next}
	}

	lastIndex := len(events) - 1
	merged, ok := mergeRunCardSourceEvent(events[lastIndex], next)
	if !ok {
		return append(events, next)
	}
	events[lastIndex] = merged
	return events
}

func mergeRunCardSourceEvent(
	current bridgeTasks.RunCardSourceEvent,
	next bridgeTasks.RunCardSourceEvent,
) (bridgeTasks.RunCardSourceEvent, bool) {
	if current.Type != string(streaming.EventCompletionDelta) || next.Type != string(streaming.EventCompletionDelta) {
		return bridgeTasks.RunCardSourceEvent{}, false
	}
	if current.TraceID != next.TraceID || current.StepID != next.StepID || current.Turn != next.Turn {
		return bridgeTasks.RunCardSourceEvent{}, false
	}

	payload, ok := mergeCompletionDeltaPayload(current.Payload, next.Payload)
	if !ok {
		return bridgeTasks.RunCardSourceEvent{}, false
	}
	current.Payload = payload
	if current.SessionID == "" {
		current.SessionID = next.SessionID
	}
	if !next.At.IsZero() {
		current.At = next.At.UTC()
	}
	return current, true
}

func mergeCompletionDeltaPayload(
	current map[string]any,
	next map[string]any,
) (map[string]any, bool) {
	kind := payloadString(current, "kind")
	if kind == "" || kind != payloadString(next, "kind") {
		return nil, false
	}

	switch kind {
	case "text":
		return mergeStringDeltaPayload(current, next, "text")
	case "thinking":
		return mergeStringDeltaPayload(current, next, "thinking")
	case "tool_call_delta":
		if !sameToolCallIndex(current, next) {
			return nil, false
		}
		return mergeStringDeltaPayload(current, next, "arguments_fragment")
	default:
		return nil, false
	}
}

func mergeStringDeltaPayload(
	current map[string]any,
	next map[string]any,
	key string,
) (map[string]any, bool) {
	merged := payloadRecord(current)
	merged[key] = payloadString(current, key) + payloadString(next, key)
	return merged, true
}

func sameToolCallIndex(current map[string]any, next map[string]any) bool {
	currentIndex, ok := payloadInt(current, "tool_call_index")
	if !ok {
		return false
	}
	nextIndex, ok := payloadInt(next, "tool_call_index")
	if !ok {
		return false
	}
	return currentIndex == nextIndex
}

func payloadInt(payload map[string]any, key string) (int, bool) {
	value, ok := payload[key]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case int:
		return typed, true
	case float64:
		return int(typed), true
	default:
		return 0, false
	}
}

func payloadString(payload map[string]any, key string) string {
	value, _ := payload[key].(string)
	return strings.TrimSpace(value)
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
