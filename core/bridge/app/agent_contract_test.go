package app

import "testing"

func TestNewAgentResponsePayloadBuildsNormalSessionResponse(t *testing.T) {
	payload, err := newAgentResponsePayload("final answer", "session-1", nil)
	if err != nil {
		t.Fatalf("newAgentResponsePayload returned error: %v", err)
	}
	if payload.Message != "final answer" {
		t.Fatalf("unexpected message: got %q want %q", payload.Message, "final answer")
	}
	if payload.SessionID != "session-1" {
		t.Fatalf("unexpected session id: got %q want %q", payload.SessionID, "session-1")
	}
	if payload.SessionEnded {
		t.Fatal("session_ended should be false for normal response")
	}
	if payload.SessionEnd != nil {
		t.Fatalf("session_end should be nil for normal response: %+v", payload.SessionEnd)
	}
}

func TestValidateAgentResponsePayloadRejectsInconsistentSessionEnd(t *testing.T) {
	err := validateAgentResponsePayload(agentResponse{
		Message:      "final answer",
		SessionID:    "session-1",
		SessionEnded: true,
		SessionEnd: &assistantSessionEndSignalPayload{
			Signal:  busAssistantSessionEndSignal,
			Message: "different",
		},
	})
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}
