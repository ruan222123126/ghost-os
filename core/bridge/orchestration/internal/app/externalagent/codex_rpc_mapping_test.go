package externalagent

import (
	"encoding/json"
	"testing"
)

func TestEventFromNotificationMapsAgentMessageDelta(t *testing.T) {
	event, ok := eventFromNotification("item/agentMessage/delta", json.RawMessage(`{"delta":"hello"}`))
	if !ok {
		t.Fatal("expected agent message delta notification to map")
	}
	if event.Type != "agent_message" || stringValue(event.Payload["message"]) != "hello" {
		t.Fatalf("unexpected event: %+v", event)
	}
}

func TestEventFromNotificationNormalizesCodexEventMessageDelta(t *testing.T) {
	event, ok := eventFromNotification("codex/event", json.RawMessage(`{
		"msg": {
			"type": "agent_message_content_delta",
			"delta": "hello"
		}
	}`))
	if !ok {
		t.Fatal("expected codex/event message delta to map")
	}
	if event.Type != "agent_message" || stringValue(event.Payload["message"]) != "hello" {
		t.Fatalf("unexpected event: %+v", event)
	}
}

func TestEventFromNotificationMapsCompletedAgentMessageAsSnapshot(t *testing.T) {
	event, ok := eventFromNotification("item/completed", json.RawMessage(`{
		"item": {
			"type": "agentMessage",
			"text": "hello"
		}
	}`))
	if !ok {
		t.Fatal("expected completed agent message notification to map")
	}
	if event.Type != codexEventAgentMessageSnapshot || stringValue(event.Payload["message"]) != "hello" {
		t.Fatalf("unexpected event: %+v", event)
	}
}

func TestEventFromNotificationMapsCodexAgentMessageAsSnapshot(t *testing.T) {
	event, ok := eventFromNotification("codex/event", json.RawMessage(`{
		"msg": {
			"type": "agent_message",
			"message": "hello"
		}
	}`))
	if !ok {
		t.Fatal("expected codex/event agent message notification to map")
	}
	if event.Type != codexEventAgentMessageSnapshot || stringValue(event.Payload["message"]) != "hello" {
		t.Fatalf("unexpected event: %+v", event)
	}
}

func TestEventFromNotificationNormalizesCodexReasoningDelta(t *testing.T) {
	event, ok := eventFromNotification("codex/event", json.RawMessage(`{
		"msg": {
			"type": "reasoning_content_delta",
			"delta": "thinking"
		}
	}`))
	if !ok {
		t.Fatal("expected codex/event reasoning delta to map")
	}
	if event.Type != "agent_reasoning_delta" || stringValue(event.Payload["text"]) != "thinking" {
		t.Fatalf("unexpected event: %+v", event)
	}
}
