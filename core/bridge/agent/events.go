package agent

import (
	"context"
	"fmt"
	"strings"
	"time"
)

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

type AgentEvent struct {
	ID      string    `json:"id"`
	StepID  string    `json:"step_id"`
	TraceID string    `json:"trace_id"`
	Turn    int       `json:"turn"`
	Type    EventType `json:"type"`
	Payload any       `json:"payload"`
	At      time.Time `json:"at"`
}

type EventSink interface {
	Emit(context.Context, AgentEvent) error
}

type nopSink struct{}

func (nopSink) Emit(context.Context, AgentEvent) error { return nil }

func NewEvent(traceID string, turn int, stepID string, eventType EventType, payload any) AgentEvent {
	return AgentEvent{
		StepID:  strings.TrimSpace(stepID),
		TraceID: strings.TrimSpace(traceID),
		Turn:    max(turn, 0),
		Type:    eventType,
		Payload: payload,
		At:      time.Now().UTC(),
	}
}

func FormatEventID(traceID string, sequence int) string {
	return fmt.Sprintf("%s:%06d", strings.TrimSpace(traceID), max(sequence, 0))
}

func AssistantStepID(turn int) string {
	return fmt.Sprintf("turn-%04d-assistant", max(turn, 0))
}

func ToolStepID(turn int, toolIndex int) string {
	return fmt.Sprintf("turn-%04d-tool-%04d", max(turn, 0), max(toolIndex, 0))
}

func max(value int, floor int) int {
	if value < floor {
		return floor
	}
	return value
}
