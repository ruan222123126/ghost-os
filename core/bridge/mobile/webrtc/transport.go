package mobilewebrtc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/mobile"
	bridgeorchestration "ghost-os/bridge/orchestration"
	"ghost-os/bridge/streaming"
)

const dataChannelLabel = "ghost-os.bus"

type Transport struct {
	cfg     bridgeconfig.MobileWebRTCConfig
	service *bridgeorchestration.Service
	store   *mobile.CredentialStore
	gate    *mobile.ActiveConnectionGate

	ctx       context.Context
	cancel    context.CancelFunc
	signaling *signalingConn

	peersMu sync.Mutex
	peers   map[string]*peerSession
}

func Start(
	ctx context.Context,
	cfg bridgeconfig.MobileWebRTCConfig,
	service *bridgeorchestration.Service,
) (*Transport, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if service == nil {
		return nil, fmt.Errorf("orchestration service is required")
	}
	signaling, err := dialSignaling(ctx, cfg.SignalingURL, cfg.SignalingToken, cfg.PCID)
	if err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithCancel(ctx)
	t := &Transport{
		cfg:       cfg,
		service:   service,
		store:     mobile.NewCredentialStore(cfg.CredentialStorePath),
		gate:      mobile.NewActiveConnectionGate(),
		ctx:       runCtx,
		cancel:    cancel,
		signaling: signaling,
		peers:     make(map[string]*peerSession),
	}
	go signaling.readLoop(runCtx, t.handleSignal)
	go signaling.heartbeatLoop(runCtx, signalingHeartbeatEvery, func(err error) {
		log.Printf("mobile_webrtc signaling heartbeat failed: %v", err)
	})
	return t, nil
}

func (t *Transport) Close() {
	if t == nil {
		return
	}
	t.cancel()
	t.signaling.close()
	t.closePeers()
}

func (t *Transport) handleSignal(msg signalMessage) {
	switch strings.TrimSpace(msg.Type) {
	case signalMobileConnect:
		if err := t.startPeer(msg); err != nil {
			log.Printf("mobile_webrtc start peer failed mobile_id=%s error=%v", msg.MobileID, err)
			t.sendSignal(signalMessage{
				Type:     signalBusy,
				PCID:     t.cfg.PCID,
				MobileID: firstNonEmpty(msg.MobileID, msg.DeviceID),
				DeviceID: firstNonEmpty(msg.DeviceID, msg.MobileID),
				Error:    err.Error(),
			})
		}
	case signalAnswer:
		if err := t.applyAnswer(msg); err != nil {
			log.Printf("mobile_webrtc apply answer failed mobile_id=%s error=%v", msg.MobileID, err)
		}
	case signalICE:
		if err := t.addRemoteICE(msg); err != nil {
			log.Printf("mobile_webrtc add ice failed mobile_id=%s error=%v", msg.MobileID, err)
		}
	case signalError:
		log.Printf("mobile_webrtc signaling error: %s", strings.TrimSpace(msg.Error))
	case signalHeartbeat:
	default:
		log.Printf("mobile_webrtc ignored signaling message type=%s", strings.TrimSpace(msg.Type))
	}
}

func (t *Transport) startPeer(msg signalMessage) error {
	mobileID := firstNonEmpty(msg.MobileID, msg.DeviceID)
	deviceID := firstNonEmpty(msg.DeviceID, msg.MobileID)
	if strings.TrimSpace(mobileID) == "" {
		return fmt.Errorf("mobile_id is required")
	}
	if active, ok := t.gate.Snapshot(); ok && active.DeviceID != strings.TrimSpace(deviceID) {
		return fmt.Errorf("busy: active_device_id=%s", active.DeviceID)
	}

	peer, err := t.newPeerSession(mobileID, deviceID)
	if err != nil {
		return err
	}
	t.replacePeer(mobileID, peer)
	if err := peer.createOffer(); err != nil {
		peer.close()
		t.removePeer(mobileID, peer)
		return err
	}
	return nil
}

func (t *Transport) newPeerSession(mobileID string, deviceID string) (*peerSession, error) {
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{ICEServers: toPionICEServers(t.cfg.ICEServers)})
	if err != nil {
		return nil, fmt.Errorf("create peer connection: %w", err)
	}
	ctx, cancel := context.WithCancel(t.ctx)
	peer := &peerSession{
		transport:      t,
		pc:             pc,
		mobileID:       strings.TrimSpace(mobileID),
		deviceID:       strings.TrimSpace(deviceID),
		ctx:            ctx,
		cancel:         cancel,
		requests:       make(map[string]context.CancelFunc),
		streamRequests: make(map[string]context.CancelFunc),
	}
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		peer.sendICE(candidate)
	})
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateFailed ||
			state == webrtc.PeerConnectionStateClosed ||
			state == webrtc.PeerConnectionStateDisconnected {
			peer.close()
		}
	})

	dc, err := pc.CreateDataChannel(dataChannelLabel, nil)
	if err != nil {
		_ = pc.Close()
		cancel()
		return nil, fmt.Errorf("create data channel: %w", err)
	}
	peer.attachDataChannel(dc)
	return peer, nil
}

func (p *peerSession) createOffer() error {
	offer, err := p.pc.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("create offer: %w", err)
	}
	if err := p.pc.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("set local offer: %w", err)
	}
	return p.transport.sendSignal(signalMessage{
		Type:     signalOffer,
		PCID:     p.transport.cfg.PCID,
		MobileID: p.mobileID,
		DeviceID: p.deviceID,
		SDP:      offer.SDP,
	})
}

func (t *Transport) applyAnswer(msg signalMessage) error {
	peer := t.peer(firstNonEmpty(msg.MobileID, msg.DeviceID))
	if peer == nil {
		return fmt.Errorf("peer not found")
	}
	if strings.TrimSpace(msg.SDP) == "" {
		return fmt.Errorf("answer sdp is required")
	}
	return peer.applyAnswer(msg.SDP)
}

func (t *Transport) addRemoteICE(msg signalMessage) error {
	peer := t.peer(firstNonEmpty(msg.MobileID, msg.DeviceID))
	if peer == nil {
		return fmt.Errorf("peer not found")
	}
	var candidate webrtc.ICECandidateInit
	if err := json.Unmarshal(msg.Candidate, &candidate); err != nil {
		return fmt.Errorf("decode ice candidate: %w", err)
	}
	if strings.TrimSpace(candidate.Candidate) == "" {
		return nil
	}
	return peer.addRemoteICE(candidate)
}

func (t *Transport) sendSignal(msg signalMessage) error {
	return t.signaling.write(msg)
}

func (t *Transport) replacePeer(mobileID string, peer *peerSession) {
	t.peersMu.Lock()
	previous := t.peers[strings.TrimSpace(mobileID)]
	t.peers[strings.TrimSpace(mobileID)] = peer
	t.peersMu.Unlock()
	if previous != nil {
		previous.close()
	}
}

func (t *Transport) peer(mobileID string) *peerSession {
	t.peersMu.Lock()
	defer t.peersMu.Unlock()
	return t.peers[strings.TrimSpace(mobileID)]
}

func (t *Transport) removePeer(mobileID string, peer *peerSession) {
	t.peersMu.Lock()
	defer t.peersMu.Unlock()
	if t.peers[strings.TrimSpace(mobileID)] == peer {
		delete(t.peers, strings.TrimSpace(mobileID))
	}
}

func (t *Transport) closePeers() {
	t.peersMu.Lock()
	peers := make([]*peerSession, 0, len(t.peers))
	for _, peer := range t.peers {
		peers = append(peers, peer)
	}
	t.peers = make(map[string]*peerSession)
	t.peersMu.Unlock()
	for _, peer := range peers {
		peer.close()
	}
}

func toPionICEServers(raw []bridgeconfig.MobileICEServerConfig) []webrtc.ICEServer {
	servers := make([]webrtc.ICEServer, 0, len(raw))
	for _, server := range raw {
		if len(server.URLs) == 0 {
			continue
		}
		servers = append(servers, webrtc.ICEServer{
			URLs:       append([]string(nil), server.URLs...),
			Username:   strings.TrimSpace(server.Username),
			Credential: strings.TrimSpace(server.Credential),
		})
	}
	return servers
}

type peerSession struct {
	transport *Transport
	pc        *webrtc.PeerConnection
	dc        *webrtc.DataChannel
	mobileID  string
	deviceID  string

	ctx    context.Context
	cancel context.CancelFunc

	mu             sync.Mutex
	authenticated  bool
	challenge      string
	requests       map[string]context.CancelFunc
	streamRequests map[string]context.CancelFunc
	answerSet      bool
	remoteICE      []webrtc.ICECandidateInit
	closed         bool
}

func (p *peerSession) attachDataChannel(dc *webrtc.DataChannel) {
	p.dc = dc
	dc.OnOpen(func() {
		p.sendAuthChallenge()
	})
	dc.OnMessage(func(message webrtc.DataChannelMessage) {
		p.handleDataMessage(message.Data)
	})
	dc.OnClose(func() {
		p.close()
	})
}

func (p *peerSession) sendICE(candidate *webrtc.ICECandidate) {
	if candidate == nil {
		return
	}
	data, err := json.Marshal(candidate.ToJSON())
	if err != nil {
		log.Printf("mobile_webrtc encode ice failed mobile_id=%s error=%v", p.mobileID, err)
		return
	}
	if err := p.transport.sendSignal(signalMessage{
		Type:      signalICE,
		PCID:      p.transport.cfg.PCID,
		MobileID:  p.mobileID,
		DeviceID:  p.deviceID,
		Candidate: data,
	}); err != nil {
		log.Printf("mobile_webrtc send ice failed mobile_id=%s error=%v", p.mobileID, err)
	}
}

func (p *peerSession) applyAnswer(sdp string) error {
	if err := p.pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  sdp,
	}); err != nil {
		return err
	}
	return p.flushRemoteICE()
}

func (p *peerSession) addRemoteICE(candidate webrtc.ICECandidateInit) error {
	p.mu.Lock()
	if !p.answerSet {
		p.remoteICE = append(p.remoteICE, candidate)
		p.mu.Unlock()
		return nil
	}
	p.mu.Unlock()
	return p.pc.AddICECandidate(candidate)
}

func (p *peerSession) flushRemoteICE() error {
	p.mu.Lock()
	p.answerSet = true
	candidates := p.remoteICE
	p.remoteICE = nil
	p.mu.Unlock()

	for _, candidate := range candidates {
		if err := p.pc.AddICECandidate(candidate); err != nil {
			return fmt.Errorf("add queued ice candidate: %w", err)
		}
	}
	return nil
}

func (p *peerSession) sendAuthChallenge() {
	challenge, err := mobile.NewChallenge()
	if err != nil {
		p.sendFrame(mobile.Frame{Type: mobile.FrameAuthError, Error: err.Error()})
		p.close()
		return
	}
	p.mu.Lock()
	p.challenge = challenge
	p.mu.Unlock()
	p.sendFrame(mobile.Frame{Type: mobile.FrameAuthChallenge, Challenge: challenge})
}

func (p *peerSession) handleDataMessage(data []byte) {
	frame, err := mobile.DecodeFrame(data)
	if err != nil {
		p.sendFrame(mobile.Frame{Type: mobile.FrameAuthError, Error: err.Error()})
		return
	}
	if !p.isAuthenticated() {
		p.handleUnauthenticatedFrame(frame)
		return
	}

	switch frame.Type {
	case mobile.FrameRequest:
		go p.handleRequest(frame)
	case mobile.FrameStreamStart:
		go p.handleStream(frame)
	case mobile.FrameCancel:
		p.cancelRequest(frame.RequestID)
	default:
		p.sendFrame(mobile.Frame{
			Type:      mobile.FrameResponse,
			RequestID: frame.RequestID,
			Status:    "error",
			Error:     "unsupported authenticated frame type",
		})
	}
}

func (p *peerSession) handleUnauthenticatedFrame(frame mobile.Frame) {
	if frame.Type != mobile.FrameAuthResponse {
		p.sendFrame(mobile.Frame{Type: mobile.FrameAuthError, Error: "authentication is required"})
		return
	}
	p.mu.Lock()
	challenge := p.challenge
	p.mu.Unlock()

	authenticator := mobile.NewAuthenticator(p.transport.store)
	device, err := authenticator.Verify(frame.DeviceID, challenge, frame.Signature)
	if err != nil {
		p.sendFrame(mobile.Frame{Type: mobile.FrameAuthError, Error: err.Error()})
		p.close()
		return
	}
	acquired := p.transport.gate.Acquire(mobile.ActiveConnection{
		DeviceID:     device.DeviceID,
		ConnectionID: p.mobileID,
		StartedAt:    time.Now(),
		Cancel:       p.close,
	})
	if acquired.Busy {
		p.sendFrame(mobile.Frame{Type: mobile.FrameAuthError, Error: "busy"})
		p.close()
		return
	}
	_ = p.transport.store.MarkConnected(device.DeviceID, time.Now())
	p.mu.Lock()
	p.authenticated = true
	p.deviceID = device.DeviceID
	p.mu.Unlock()
	p.sendFrame(mobile.Frame{Type: mobile.FrameAuthOK})
}

func (p *peerSession) handleRequest(frame mobile.Frame) {
	ctx, cancel := context.WithCancel(p.ctx)
	p.trackRequest(frame.RequestID, cancel)
	defer p.finishRequest(frame.RequestID)

	result, err := p.transport.service.DispatchAction(ctx, strings.ToUpper(frame.Action), mobile.RequestRawParams(frame), frame.TraceID)
	if err != nil {
		p.sendFrame(mobile.Frame{
			Type:      mobile.FrameResponse,
			RequestID: frame.RequestID,
			Status:    "error",
			Payload:   map[string]any{},
			Error:     err.Error(),
		})
		return
	}
	p.sendFrame(mobile.Frame{
		Type:      mobile.FrameResponse,
		RequestID: frame.RequestID,
		Status:    "success",
		Payload:   result.Payload,
		Error:     "",
	})
}

func (p *peerSession) handleStream(frame mobile.Frame) {
	ctx, cancel := context.WithCancel(p.transport.ctx)
	p.trackStreamRequest(frame.RequestID, cancel)
	defer cancel()
	defer p.finishStreamRequest(frame.RequestID)

	action := strings.ToUpper(strings.TrimSpace(frame.Action))
	switch action {
	case bridgeorchestration.BusActionAgentSend:
		p.handleAgentStream(ctx, frame)
	case bridgeorchestration.BusActionExternalAgentStart, bridgeorchestration.BusActionExternalAgentSend:
		p.handleExternalAgentStream(ctx, frame, action == bridgeorchestration.BusActionExternalAgentStart)
	default:
		p.sendStreamEnd(frame.RequestID, "error", "unsupported stream_start action: "+action, map[string]any{})
	}
}

func (p *peerSession) handleAgentStream(ctx context.Context, frame mobile.Frame) {
	var params bridgeorchestration.AgentParams
	if err := json.Unmarshal(mobile.RequestRawParams(frame), &params); err != nil {
		p.sendStreamEnd(frame.RequestID, "error", fmt.Sprintf("decode AGENT_SEND params: %v", err), map[string]any{})
		return
	}
	prepared, _, err := p.transport.service.PrepareAgentStreamAction(ctx, params, frame.TraceID)
	if err != nil {
		p.sendStreamEnd(frame.RequestID, "error", err.Error(), map[string]any{})
		return
	}
	p.runPreparedStream(ctx, frame, prepared, params.SessionID)
}

func (p *peerSession) handleExternalAgentStream(ctx context.Context, frame mobile.Frame, forceStart bool) {
	var params bridgeorchestration.ExternalAgentRequest
	if err := json.Unmarshal(mobile.RequestRawParams(frame), &params); err != nil {
		p.sendStreamEnd(frame.RequestID, "error", fmt.Sprintf("decode %s params: %v", frame.Action, err), map[string]any{})
		return
	}
	prepared, _, err := p.transport.service.PrepareExternalAgentStreamAction(params, frame.TraceID, forceStart)
	if err != nil {
		p.sendStreamEnd(frame.RequestID, "error", err.Error(), map[string]any{})
		return
	}
	p.runPreparedStream(ctx, frame, prepared, params.SessionID)
}

func (p *peerSession) runPreparedStream(
	ctx context.Context,
	frame mobile.Frame,
	prepared bridgeorchestration.PreparedAgentStream,
	inputSessionID string,
) {
	sink := dataChannelStreamSink{peer: p, requestID: frame.RequestID}
	_, sessionID, err := prepared.Run(ctx, sink)
	if err != nil {
		p.sendStreamEnd(frame.RequestID, "error", err.Error(), map[string]any{"session_id": firstNonEmpty(sessionID, inputSessionID)})
		return
	}
	p.sendStreamEnd(frame.RequestID, "success", "", map[string]any{"session_id": sessionID})
}

func (p *peerSession) trackRequest(requestID string, cancel context.CancelFunc) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.requests[strings.TrimSpace(requestID)] = cancel
}

func (p *peerSession) finishRequest(requestID string) {
	p.mu.Lock()
	cancel := p.requests[strings.TrimSpace(requestID)]
	delete(p.requests, strings.TrimSpace(requestID))
	p.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (p *peerSession) trackStreamRequest(requestID string, cancel context.CancelFunc) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.streamRequests[strings.TrimSpace(requestID)] = cancel
}

func (p *peerSession) finishStreamRequest(requestID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.streamRequests, strings.TrimSpace(requestID))
}

func (p *peerSession) cancelRequest(requestID string) {
	cancel := p.takeRequestCancel(requestID)
	if cancel != nil {
		cancel()
	}
}

func (p *peerSession) takeRequestCancel(requestID string) context.CancelFunc {
	key := strings.TrimSpace(requestID)
	p.mu.Lock()
	defer p.mu.Unlock()
	if cancel := p.requests[key]; cancel != nil {
		delete(p.requests, key)
		return cancel
	}
	cancel := p.streamRequests[key]
	delete(p.streamRequests, key)
	return cancel
}

func (p *peerSession) sendStreamEnd(requestID string, status string, errText string, payload any) {
	p.sendFrame(mobile.Frame{
		Type:      mobile.FrameStreamEnd,
		RequestID: requestID,
		Status:    status,
		Payload:   payload,
		Error:     errText,
	})
}

func (p *peerSession) sendFrame(frame mobile.Frame) {
	data, err := mobile.EncodeFrame(frame)
	if err != nil {
		log.Printf("mobile_webrtc encode frame failed mobile_id=%s error=%v", p.mobileID, err)
		return
	}
	p.mu.Lock()
	dc := p.dc
	closed := p.closed
	p.mu.Unlock()
	if dc == nil || closed {
		return
	}
	if err := dc.SendText(string(data)); err != nil {
		log.Printf("mobile_webrtc send frame failed mobile_id=%s error=%v", p.mobileID, err)
	}
}

func (p *peerSession) isAuthenticated() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.authenticated
}

func (p *peerSession) close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	requests := p.requests
	p.requests = make(map[string]context.CancelFunc)
	p.streamRequests = make(map[string]context.CancelFunc)
	authenticated := p.authenticated
	deviceID := p.deviceID
	p.mu.Unlock()

	for _, cancel := range requests {
		cancel()
	}
	p.cancel()
	if p.dc != nil {
		_ = p.dc.Close()
	}
	if p.pc != nil {
		_ = p.pc.Close()
	}
	if authenticated {
		p.transport.gate.Release(deviceID, p.mobileID)
	}
	p.transport.removePeer(p.mobileID, p)
}

type dataChannelStreamSink struct {
	peer      *peerSession
	requestID string
}

func (s dataChannelStreamSink) Emit(ctx context.Context, event streaming.Event) (streaming.Event, error) {
	select {
	case <-ctx.Done():
		return event, ctx.Err()
	default:
	}
	s.peer.sendFrame(mobile.Frame{
		Type:      mobile.FrameStreamEvent,
		RequestID: s.requestID,
		Event:     string(event.Type),
		Payload:   event,
	})
	return event, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
