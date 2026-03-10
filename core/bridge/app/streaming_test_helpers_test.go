package app

import (
	"testing"

	"ghost-os/bridge/streaming"
)

func mustAppEvent(t *testing.T, traceID string, sessionID string, turn int, stepID string, eventType streaming.EventType, payload any) streaming.Event {
	t.Helper()

	event, err := streaming.NewEvent(traceID, sessionID, turn, stepID, eventType, payload)
	if err != nil {
		t.Fatalf("NewEvent returned error: %v", err)
	}
	return event
}

func mustAppAssistantStepID(t *testing.T, turn int) string {
	t.Helper()

	stepID, err := streaming.AssistantStepID(turn)
	if err != nil {
		t.Fatalf("AssistantStepID returned error: %v", err)
	}
	return stepID
}

func mustAppToolStepID(t *testing.T, turn int, toolIndex int) string {
	t.Helper()

	stepID, err := streaming.ToolStepID(turn, toolIndex)
	if err != nil {
		t.Fatalf("ToolStepID returned error: %v", err)
	}
	return stepID
}
