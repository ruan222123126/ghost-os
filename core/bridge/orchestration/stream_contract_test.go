package orchestration

import (
	"encoding/json"
	"testing"
	"time"

	"ghost-os/bridge/streaming"
)

func TestStreamingEventMatchesSharedContract(t *testing.T) {
	event := streaming.Event{
		ID:        "trace-123:000001",
		StepID:    "turn-0001-assistant",
		TraceID:   "trace-123",
		SessionID: "session-123",
		Turn:      1,
		Type:      streaming.EventMessage,
		Payload: map[string]any{
			"text":       "done",
			"session_id": "session-123",
		},
		At: time.Unix(42, 0).UTC(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal streaming event: %v", err)
	}

	var contract agentStreamEventContract
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatalf("unmarshal shared contract: %v", err)
	}
	if contract.Type != string(streaming.EventMessage) || contract.TraceID != "trace-123" {
		t.Fatalf("unexpected contract envelope: %+v", contract)
	}
	if contract.Payload["text"] != "done" || contract.At == "" {
		t.Fatalf("unexpected contract payload: %+v", contract)
	}
}

func TestSessionPushEventMatchesSharedContract(t *testing.T) {
	event := sessionPushEvent{
		ID:        "session-123:000001",
		Type:      sessionPushAssistantMessage,
		TraceID:   "trace-123",
		SessionID: "session-123",
		Payload: assistantMessagePushPayload{
			Message:      "sent to phone",
			SessionEnded: false,
		},
		At: time.Unix(52, 0).UTC(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal session push event: %v", err)
	}

	var contract sessionPushEventContract
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatalf("unmarshal shared contract: %v", err)
	}
	if contract.Type != string(sessionPushAssistantMessage) || contract.SessionID != "session-123" {
		t.Fatalf("unexpected contract envelope: %+v", contract)
	}
	if contract.Payload["message"] != "sent to phone" || contract.At == "" {
		t.Fatalf("unexpected contract payload: %+v", contract)
	}
}
