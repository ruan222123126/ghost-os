package mobilewebrtc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	signalPCRegister    = "pc.register"
	signalMobileConnect = "mobile.connect"
	signalOffer         = "offer"
	signalAnswer        = "answer"
	signalICE           = "ice"
	signalBusy          = "busy"
	signalHeartbeat     = "heartbeat"
	signalError         = "error"
	signalRegistered    = "registered"

	signalingRegisterTimeout = 5 * time.Second
	signalingHeartbeatEvery  = 15 * time.Second
)

type signalMessage struct {
	Type      string          `json:"type"`
	PCID      string          `json:"pc_id,omitempty"`
	MobileID  string          `json:"mobile_id,omitempty"`
	DeviceID  string          `json:"device_id,omitempty"`
	SDP       string          `json:"sdp,omitempty"`
	Candidate json.RawMessage `json:"candidate,omitempty"`
	Error     string          `json:"error,omitempty"`
}

type signalingConn struct {
	mu   sync.Mutex
	conn *websocket.Conn
}

func dialSignaling(ctx context.Context, rawURL string, token string, pcID string) (*signalingConn, error) {
	endpoint, err := normalizeWebSocketURL(rawURL)
	if err != nil {
		return nil, err
	}
	header := http.Header{}
	header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, endpoint, header)
	if err != nil {
		return nil, fmt.Errorf("connect signaling service: %w", err)
	}
	client := &signalingConn{conn: conn}
	if err := client.registerPC(pcID); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}

func normalizeWebSocketURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("invalid signaling_url: %w", err)
	}
	switch parsed.Scheme {
	case "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("invalid signaling_url scheme: %s", parsed.Scheme)
	}
	if parsed.Path == "" || parsed.Path == "/" {
		parsed.Path = "/ws"
	}
	return parsed.String(), nil
}

func (c *signalingConn) registerPC(pcID string) error {
	if err := c.write(signalMessage{Type: signalPCRegister, PCID: strings.TrimSpace(pcID)}); err != nil {
		return fmt.Errorf("register pc with signaling service: %w", err)
	}
	_ = c.conn.SetReadDeadline(time.Now().Add(signalingRegisterTimeout))
	defer func() { _ = c.conn.SetReadDeadline(time.Time{}) }()

	var ack signalMessage
	if err := c.conn.ReadJSON(&ack); err != nil {
		return fmt.Errorf("read signaling registration ack: %w", err)
	}
	if ack.Type == signalError {
		return fmt.Errorf("signaling registration failed: %s", strings.TrimSpace(ack.Error))
	}
	if ack.Type != signalRegistered {
		return fmt.Errorf("unexpected signaling registration ack: %s", strings.TrimSpace(ack.Type))
	}
	return nil
}

func (c *signalingConn) readLoop(ctx context.Context, handle func(signalMessage)) {
	defer c.close()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		var msg signalMessage
		if err := c.conn.ReadJSON(&msg); err != nil {
			return
		}
		handle(msg)
	}
}

func (c *signalingConn) heartbeatLoop(ctx context.Context, interval time.Duration, onError func(error)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.write(signalMessage{Type: signalHeartbeat}); err != nil {
				if onError != nil {
					onError(err)
				}
				return
			}
		}
	}
}

func (c *signalingConn) write(msg signalMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(msg)
}

func (c *signalingConn) close() {
	if c != nil && c.conn != nil {
		_ = c.conn.Close()
	}
}
