package mobilewebrtc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestSignalingConnHeartbeatLoopSendsHeartbeat(t *testing.T) {
	t.Parallel()

	messages := make(chan signalMessage, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("Upgrade: %v", err)
			return
		}
		defer conn.Close()

		var registered signalMessage
		if err := conn.ReadJSON(&registered); err != nil {
			t.Errorf("ReadJSON register: %v", err)
			return
		}
		messages <- registered
		if err := conn.WriteJSON(signalMessage{Type: signalRegistered, PCID: registered.PCID}); err != nil {
			t.Errorf("WriteJSON registered: %v", err)
			return
		}

		var heartbeat signalMessage
		if err := conn.ReadJSON(&heartbeat); err != nil {
			t.Errorf("ReadJSON heartbeat: %v", err)
			return
		}
		messages <- heartbeat
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conn, err := dialSignaling(ctx, server.URL, "token", "pc-1")
	if err != nil {
		t.Fatalf("dialSignaling: %v", err)
	}
	defer conn.close()

	go conn.heartbeatLoop(ctx, 10*time.Millisecond, func(err error) {
		t.Errorf("heartbeatLoop error: %v", err)
	})

	if got := readTestSignal(t, messages); got.Type != signalPCRegister || got.PCID != "pc-1" {
		t.Fatalf("expected pc.register, got %+v", got)
	}
	if got := readTestSignal(t, messages); got.Type != signalHeartbeat {
		t.Fatalf("expected heartbeat, got %+v", got)
	}
	cancel()
}

func readTestSignal(t *testing.T, messages <-chan signalMessage) signalMessage {
	t.Helper()
	select {
	case msg := <-messages:
		return msg
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for signal message")
		return signalMessage{}
	}
}
