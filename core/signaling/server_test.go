package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestSignalingRejectsUnauthorizedWebSocket(t *testing.T) {
	t.Parallel()

	server := newTestHTTPServer(t)
	defer server.Close()

	_, response, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err == nil {
		t.Fatal("expected unauthorized dial to fail")
	}
	if response == nil || response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected HTTP 401, got response=%v error=%v", response, err)
	}
}

func TestSignalingRoutesOfferAnswerICEAndBusy(t *testing.T) {
	t.Parallel()

	server := newTestHTTPServer(t)
	defer server.Close()

	pc := dialClient(t, server.URL)
	defer pc.Close()
	writeSignal(t, pc, signalMessage{Type: messagePCRegister, PCID: "pc-1"})
	if got := readSignal(t, pc); got.Type != messageRegistered {
		t.Fatalf("expected registered ack, got %+v", got)
	}

	mobile := dialClient(t, server.URL)
	defer mobile.Close()
	writeSignal(t, mobile, signalMessage{
		Type:     messageMobileConnect,
		PCID:     "pc-1",
		MobileID: "mobile-1",
		DeviceID: "device-1",
	})
	if got := readSignal(t, pc); got.Type != messageMobileConnect || got.MobileID != "mobile-1" {
		t.Fatalf("expected mobile.connect routed to pc, got %+v", got)
	}

	writeSignal(t, pc, signalMessage{Type: messageOffer, PCID: "pc-1", MobileID: "mobile-1", SDP: "offer-sdp"})
	if got := readSignal(t, mobile); got.Type != messageOffer || got.SDP != "offer-sdp" {
		t.Fatalf("expected offer routed to mobile, got %+v", got)
	}

	writeSignal(t, mobile, signalMessage{Type: messageAnswer, PCID: "pc-1", MobileID: "mobile-1", SDP: "answer-sdp"})
	if got := readSignal(t, pc); got.Type != messageAnswer || got.SDP != "answer-sdp" {
		t.Fatalf("expected answer routed to pc, got %+v", got)
	}

	writeSignal(t, pc, signalMessage{Type: messageBusy, PCID: "pc-1", MobileID: "mobile-1", Error: "busy"})
	if got := readSignal(t, mobile); got.Type != messageBusy || got.Error != "busy" {
		t.Fatalf("expected busy routed to mobile, got %+v", got)
	}
}

func TestSignalingReportsOfflinePC(t *testing.T) {
	t.Parallel()

	server := newTestHTTPServer(t)
	defer server.Close()

	mobile := dialClient(t, server.URL)
	defer mobile.Close()
	writeSignal(t, mobile, signalMessage{Type: messageMobileConnect, PCID: "pc-missing", MobileID: "mobile-1"})
	if got := readSignal(t, mobile); got.Type != messageError || got.Error != "pc is offline" {
		t.Fatalf("expected offline pc error, got %+v", got)
	}
}

func TestSignalingFindsStaleClients(t *testing.T) {
	t.Parallel()

	server, err := NewServer(Config{Token: "token", HeartbeatTimeout: time.Millisecond})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	client := &client{lastSeen: time.Now().Add(-time.Second)}
	server.clients[client] = struct{}{}
	stale := server.staleClients(time.Now())
	if len(stale) != 1 || stale[0] != client {
		t.Fatalf("expected stale client, got %v", stale)
	}
}

func newTestHTTPServer(t *testing.T) *httptest.Server {
	t.Helper()
	signaling, err := NewServer(Config{Token: "token", HeartbeatTimeout: time.Second})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	return httptest.NewServer(signaling.Handler())
}

func dialClient(t *testing.T, rawURL string) *websocket.Conn {
	t.Helper()
	header := http.Header{}
	header.Set("Authorization", "Bearer token")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(rawURL), header)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	return conn
}

func writeSignal(t *testing.T, conn *websocket.Conn, msg signalMessage) {
	t.Helper()
	if err := conn.WriteJSON(msg); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
}

func readSignal(t *testing.T, conn *websocket.Conn) signalMessage {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}
	var msg signalMessage
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}
	return msg
}

func wsURL(raw string) string {
	return "ws" + raw[len("http"):] + "/ws"
}
