package mobile

import (
	"strings"
	"testing"
	"time"
)

func TestCredentialStorePairAuthAndRevoke(t *testing.T) {
	t.Parallel()

	store := NewCredentialStore(t.TempDir() + "/devices.json")
	result, err := store.Pair(PairingOptions{
		Label:        "phone",
		PCID:         "pc-1",
		SignalingURL: "wss://signal.example/ws",
		Now:          time.Unix(10, 0),
	})
	if err != nil {
		t.Fatalf("Pair: %v", err)
	}
	if !strings.Contains(result.URI, "ghost-os://mobile-pair") {
		t.Fatalf("unexpected pairing uri: %s", result.URI)
	}

	challenge, err := NewChallenge()
	if err != nil {
		t.Fatalf("NewChallenge: %v", err)
	}
	secret, _, err := store.SecretFor(result.Device.DeviceID)
	if err != nil {
		t.Fatalf("SecretFor: %v", err)
	}
	signature, err := SignChallenge(secret, challenge)
	if err != nil {
		t.Fatalf("SignChallenge: %v", err)
	}
	if _, err := NewAuthenticator(store).Verify(result.Device.DeviceID, challenge, signature); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if err := store.Revoke(result.Device.DeviceID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if _, err := NewAuthenticator(store).Verify(result.Device.DeviceID, challenge, signature); err == nil {
		t.Fatal("expected revoked device to fail auth")
	}
}

func TestFrameValidationRejectsOversizeAndMissingFields(t *testing.T) {
	t.Parallel()

	if _, err := DecodeFrame(make([]byte, MaxFrameBytes+1)); err == nil {
		t.Fatal("expected oversize frame error")
	}
	if _, err := EncodeFrame(Frame{Type: FrameRequest, RequestID: "req-1"}); err == nil {
		t.Fatal("expected missing action/trace_id error")
	}
	encoded, err := EncodeFrame(Frame{
		Type:      FrameRequest,
		RequestID: "req-1",
		Action:    "CONFIG_GET",
		TraceID:   "trace-1",
		Params:    []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("EncodeFrame: %v", err)
	}
	if _, err := DecodeFrame(encoded); err != nil {
		t.Fatalf("DecodeFrame: %v", err)
	}
}

func TestActiveConnectionGateTakeoverAndBusy(t *testing.T) {
	t.Parallel()

	gate := NewActiveConnectionGate()
	cancelled := false
	first := gate.Acquire(ActiveConnection{
		DeviceID:     "device-a",
		ConnectionID: "conn-1",
		Cancel: func() {
			cancelled = true
		},
	})
	if !first.Accepted || first.Busy {
		t.Fatalf("expected first acquire accepted, got %+v", first)
	}
	busy := gate.Acquire(ActiveConnection{DeviceID: "device-b", ConnectionID: "conn-2"})
	if !busy.Busy || busy.Accepted {
		t.Fatalf("expected different device busy, got %+v", busy)
	}
	takeover := gate.Acquire(ActiveConnection{DeviceID: "device-a", ConnectionID: "conn-3"})
	if !takeover.Accepted || !takeover.Takeover {
		t.Fatalf("expected same device takeover, got %+v", takeover)
	}
	if !cancelled {
		t.Fatal("expected previous connection cancel")
	}
	gate.Release("device-a", "conn-3")
	if _, ok := gate.Snapshot(); ok {
		t.Fatal("expected release to clear active connection")
	}
}
