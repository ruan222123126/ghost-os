package orchestration

import "testing"

func TestParseSessionEndSignalPassThroughPlainText(t *testing.T) {
	message, signal, err := parseSessionEndSignal("normal answer")
	if err != nil {
		t.Fatalf("parseSessionEndSignal returned error: %v", err)
	}
	if message != "normal answer" {
		t.Fatalf("unexpected message: got %q want %q", message, "normal answer")
	}
	if signal != nil {
		t.Fatalf("signal should be nil for plain text: %+v", signal)
	}
}

func TestParseSessionEndSignalStructured(t *testing.T) {
	message, signal, err := parseSessionEndSignal(`{"signal":"END_SESSION","message":"bye"}`)
	if err != nil {
		t.Fatalf("parseSessionEndSignal returned error: %v", err)
	}
	if message != "bye" {
		t.Fatalf("unexpected message: got %q want %q", message, "bye")
	}
	if signal == nil {
		t.Fatal("signal should not be nil")
	}
	if signal.Signal != busAssistantSessionEndSignal {
		t.Fatalf("unexpected signal: got %q want %q", signal.Signal, busAssistantSessionEndSignal)
	}
}

func TestParseSessionEndSignalRejectsInvalidEndPayload(t *testing.T) {
	_, _, err := parseSessionEndSignal(`{"signal":"END_SESSION","message":"","extra":"x"}`)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}
