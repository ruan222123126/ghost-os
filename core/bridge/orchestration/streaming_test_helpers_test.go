package orchestration

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
