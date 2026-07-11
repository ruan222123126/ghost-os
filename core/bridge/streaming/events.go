package streaming

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// EventType is the shared stream contract consumed by agent, transport, and session push layers.
type EventType string

const (
	EventRunStarted       EventType = "run_started"
	EventCompletionDelta  EventType = "completion_delta"
	EventToolCallStarted  EventType = "tool_call_started"
	EventToolCallFinished EventType = "tool_call_finished"
	EventAwaitingHuman    EventType = "awaiting_human"
	EventMessage          EventType = "message"
	EventDone             EventType = "done"
	EventError            EventType = "error"
)

type Event struct {
	ID        string    `json:"id"`
	StepID    string    `json:"step_id"`
	TraceID   string    `json:"trace_id"`
	SessionID string    `json:"session_id,omitempty"`
	Turn      int       `json:"turn"`
	Type      EventType `json:"type"`
	Payload   any       `json:"payload"`
	At        time.Time `json:"at"`
}

type Sink interface {
	Emit(context.Context, Event) (Event, error)
}

type NopSink struct{}

func (NopSink) Emit(_ context.Context, event Event) (Event, error) { return event, nil }

func NewEvent(traceID string, sessionID string, turn int, stepID string, eventType EventType, payload any) (Event, error) {
	if err := validateNonNegative("turn", turn); err != nil {
		return Event{}, err
	}
	return Event{
		StepID:    strings.TrimSpace(stepID),
		TraceID:   strings.TrimSpace(traceID),
		SessionID: strings.TrimSpace(sessionID),
		Turn:      turn,
		Type:      eventType,
		Payload:   payload,
		At:        time.Now().UTC(),
	}, nil
}

func FormatEventID(traceID string, sequence int) (string, error) {
	if err := validateNonNegative("sequence", sequence); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%06d", strings.TrimSpace(traceID), sequence), nil
}

func AssistantStepID(turn int) (string, error) {
	if err := validateNonNegative("turn", turn); err != nil {
		return "", err
	}
	return fmt.Sprintf("turn-%04d-assistant", turn), nil
}

func ToolStepID(turn int, toolIndex int) (string, error) {
	if err := errors.Join(
		validateNonNegative("turn", turn),
		validateNonNegative("tool_index", toolIndex),
	); err != nil {
		return "", err
	}
	return fmt.Sprintf("turn-%04d-tool-%04d", turn, toolIndex), nil
}

func validateNonNegative(name string, value int) error {
	if value < 0 {
		return fmt.Errorf("%s must be >= 0: got %d", name, value)
	}
	return nil
}
