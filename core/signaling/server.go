package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	messagePCRegister    = "pc.register"
	messageMobileConnect = "mobile.connect"
	messageOffer         = "offer"
	messageAnswer        = "answer"
	messageICE           = "ice"
	messageBusy          = "busy"
	messageHeartbeat     = "heartbeat"
	messageError         = "error"
	messageRegistered    = "registered"

	clientKindPC     = "pc"
	clientKindMobile = "mobile"
)

type Config struct {
	BindAddr         string
	Token            string
	HeartbeatTimeout time.Duration
}

type Server struct {
	token            string
	heartbeatTimeout time.Duration
	upgrader         websocket.Upgrader

	mu      sync.Mutex
	clients map[*client]struct{}
	pcs     map[string]*client
	mobiles map[string]*client
}

type client struct {
	server *Server
	conn   *websocket.Conn
	mu     sync.Mutex

	kind     string
	pcID     string
	mobileID string
	lastSeen time.Time
}

type signalMessage struct {
	Type      string          `json:"type"`
	PCID      string          `json:"pc_id,omitempty"`
	MobileID  string          `json:"mobile_id,omitempty"`
	DeviceID  string          `json:"device_id,omitempty"`
	SDP       string          `json:"sdp,omitempty"`
	Candidate json.RawMessage `json:"candidate,omitempty"`
	Error     string          `json:"error,omitempty"`
}

func NewServer(config Config) (*Server, error) {
	token := strings.TrimSpace(config.Token)
	if token == "" {
		return nil, errors.New("signaling token is required")
	}
	timeout := config.HeartbeatTimeout
	if timeout <= 0 {
		timeout = 45 * time.Second
	}
	return &Server{
		token:            token,
		heartbeatTimeout: timeout,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(*http.Request) bool { return true },
		},
		clients: make(map[*client]struct{}),
		pcs:     make(map[string]*client),
		mobiles: make(map[string]*client),
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}

func (s *Server) Run(ctx context.Context, bindAddr string) error {
	server := &http.Server{
		Addr:              bindAddr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go s.closeStaleClients(ctx)
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	err := server.ListenAndServe()
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := &client{server: s, conn: conn, lastSeen: time.Now()}
	s.addClient(c)
	c.readLoop()
}

func (s *Server) authorized(r *http.Request) bool {
	provided := strings.TrimSpace(r.URL.Query().Get("token"))
	if provided == "" {
		provided = parseBearerToken(r.Header.Get("Authorization"))
	}
	return provided == s.token
}

func parseBearerToken(header string) string {
	parts := strings.Fields(strings.TrimSpace(header))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func (s *Server) addClient(c *client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[c] = struct{}{}
}

func (s *Server) removeClient(c *client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, c)
	if c.kind == clientKindPC && s.pcs[c.pcID] == c {
		delete(s.pcs, c.pcID)
	}
	if c.kind == clientKindMobile && s.mobiles[c.mobileID] == c {
		delete(s.mobiles, c.mobileID)
	}
}

func (c *client) readLoop() {
	defer func() {
		c.server.removeClient(c)
		_ = c.conn.Close()
	}()

	for {
		var msg signalMessage
		if err := c.conn.ReadJSON(&msg); err != nil {
			return
		}
		c.touch()
		if err := c.server.handleMessage(c, msg); err != nil {
			_ = c.write(signalMessage{Type: messageError, Error: err.Error()})
		}
	}
}

func (s *Server) handleMessage(c *client, msg signalMessage) error {
	switch strings.TrimSpace(msg.Type) {
	case messagePCRegister:
		return s.registerPC(c, msg)
	case messageMobileConnect:
		return s.connectMobile(c, msg)
	case messageOffer:
		return s.routeToMobile(msg)
	case messageAnswer:
		return s.routeToPC(msg)
	case messageICE:
		return s.routeICE(c, msg)
	case messageBusy:
		return s.routeToMobile(msg)
	case messageHeartbeat:
		return c.write(signalMessage{Type: messageHeartbeat})
	default:
		return fmt.Errorf("unsupported signaling message type: %s", strings.TrimSpace(msg.Type))
	}
}

func (s *Server) registerPC(c *client, msg signalMessage) error {
	pcID := strings.TrimSpace(msg.PCID)
	if pcID == "" {
		return errors.New("pc_id is required")
	}

	var previous *client
	s.mu.Lock()
	previous = s.pcs[pcID]
	c.kind = clientKindPC
	c.pcID = pcID
	s.pcs[pcID] = c
	s.mu.Unlock()

	if previous != nil && previous != c {
		_ = previous.conn.Close()
	}
	return c.write(signalMessage{Type: messageRegistered, PCID: pcID})
}

func (s *Server) connectMobile(c *client, msg signalMessage) error {
	pcID := strings.TrimSpace(msg.PCID)
	mobileID := strings.TrimSpace(firstNonEmpty(msg.MobileID, msg.DeviceID))
	if pcID == "" {
		return errors.New("pc_id is required")
	}
	if mobileID == "" {
		return errors.New("mobile_id is required")
	}

	var pc *client
	var previous *client
	s.mu.Lock()
	pc = s.pcs[pcID]
	previous = s.mobiles[mobileID]
	c.kind = clientKindMobile
	c.pcID = pcID
	c.mobileID = mobileID
	s.mobiles[mobileID] = c
	s.mu.Unlock()

	if previous != nil && previous != c {
		_ = previous.conn.Close()
	}
	if pc == nil {
		return errors.New("pc is offline")
	}
	return pc.write(signalMessage{
		Type:     messageMobileConnect,
		PCID:     pcID,
		MobileID: mobileID,
		DeviceID: firstNonEmpty(msg.DeviceID, mobileID),
	})
}

func (s *Server) routeToMobile(msg signalMessage) error {
	mobileID := strings.TrimSpace(firstNonEmpty(msg.MobileID, msg.DeviceID))
	if mobileID == "" {
		return errors.New("mobile_id is required")
	}
	target := s.mobileClient(mobileID)
	if target == nil {
		return fmt.Errorf("mobile is offline: %s", mobileID)
	}
	msg.MobileID = mobileID
	return target.write(msg)
}

func (s *Server) routeToPC(msg signalMessage) error {
	pcID := strings.TrimSpace(msg.PCID)
	if pcID == "" {
		return errors.New("pc_id is required")
	}
	target := s.pcClient(pcID)
	if target == nil {
		return fmt.Errorf("pc is offline: %s", pcID)
	}
	return target.write(msg)
}

func (s *Server) routeICE(c *client, msg signalMessage) error {
	if c.kind == clientKindPC {
		return s.routeToMobile(msg)
	}
	return s.routeToPC(msg)
}

func (s *Server) pcClient(pcID string) *client {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pcs[strings.TrimSpace(pcID)]
}

func (s *Server) mobileClient(mobileID string) *client {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mobiles[strings.TrimSpace(mobileID)]
}

func (s *Server) closeStaleClients(ctx context.Context) {
	ticker := time.NewTicker(s.heartbeatTimeout / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			for _, stale := range s.staleClients(now) {
				_ = stale.conn.Close()
			}
		}
	}
}

func (s *Server) staleClients(now time.Time) []*client {
	s.mu.Lock()
	defer s.mu.Unlock()
	stale := make([]*client, 0)
	for c := range s.clients {
		if now.Sub(c.lastSeen) > s.heartbeatTimeout {
			stale = append(stale, c)
		}
	}
	return stale
}

func (c *client) touch() {
	c.server.mu.Lock()
	defer c.server.mu.Unlock()
	c.lastSeen = time.Now()
}

func (c *client) write(msg signalMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(msg)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
