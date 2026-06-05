package tasks

import (
	"strings"
	"time"
)

type RunCardSourceEvent struct {
	ID        string         `json:"id"`
	StepID    string         `json:"step_id"`
	TraceID   string         `json:"trace_id"`
	SessionID string         `json:"session_id,omitempty"`
	Turn      int            `json:"turn"`
	Type      string         `json:"type"`
	Payload   map[string]any `json:"payload"`
	At        time.Time      `json:"at,omitempty"`
}

func CloneRunCardSourceEvents(input []RunCardSourceEvent) []RunCardSourceEvent {
	if len(input) == 0 {
		return nil
	}
	out := make([]RunCardSourceEvent, len(input))
	for index, item := range input {
		out[index] = normalizeRunCardSourceEvent(item)
	}
	return out
}

func normalizeRunCardSourceEvent(input RunCardSourceEvent) RunCardSourceEvent {
	out := RunCardSourceEvent{
		ID:        strings.TrimSpace(input.ID),
		StepID:    strings.TrimSpace(input.StepID),
		TraceID:   strings.TrimSpace(input.TraceID),
		SessionID: strings.TrimSpace(input.SessionID),
		Turn:      input.Turn,
		Type:      strings.TrimSpace(input.Type),
		Payload:   normalizeRunCardSourceEventPayload(input.Payload),
		At:        input.At,
	}
	if !out.At.IsZero() {
		out.At = out.At.UTC()
	}
	return out
}

func normalizeRunCardSourceEventPayload(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	cloned, ok := cloneJSONValue(input).(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return cloned
}
