package tools

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"ghost-os/bridge/tools/internal/toolparams"

	"github.com/chromedp/chromedp"
)

func (t *BrowserControlTool) sessionFromParams(params map[string]any) (*browserSession, error) {
	sessionID := toolparams.OptionalString(params, "session_id", "")
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	session := t.getSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("browser session %q not found", sessionID)
	}
	return session, nil
}

func (t *BrowserControlTool) getSession(sessionID string) *browserSession {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.sessions[sessionID]
}

func (t *BrowserControlTool) popSession(sessionID string) *browserSession {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.sessions == nil {
		return nil
	}
	session := t.sessions[sessionID]
	delete(t.sessions, sessionID)
	return session
}

func (t *BrowserControlTool) storeSession(session *browserSession) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if existing := t.sessions[session.id]; existing != nil {
		existing.close()
	}
	t.sessions[session.id] = session
}

func (t *BrowserControlTool) replaceSession(sessionID string, wsEndpoint string) error {
	session, err := newRemoteBrowserSession(sessionID, wsEndpoint)
	if err != nil {
		return err
	}
	t.storeSession(session)
	return nil
}

func newRemoteBrowserSession(sessionID string, wsEndpoint string) (*browserSession, error) {
	allocCtx, allocStop := chromedp.NewRemoteAllocator(context.Background(), wsEndpoint)
	ctx, stop := chromedp.NewContext(allocCtx)
	return &browserSession{
		id:         sessionID,
		wsEndpoint: wsEndpoint,
		allocCtx:   allocCtx,
		allocStop:  allocStop,
		ctx:        ctx,
		stop:       stop,
		lastUsed:   time.Now(),
	}, nil
}

func (s *browserSession) run(timeout time.Duration, actions ...chromedp.Action) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastUsed = time.Now()

	runCtx := s.ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(runCtx, timeout)
		defer cancel()
	}
	return chromedp.Run(runCtx, actions...)
}

func (s *browserSession) close() {
	if s.stop != nil {
		s.stop()
	}
	if s.allocStop != nil {
		s.allocStop()
	}
}

func newBrowserSessionID() string {
	var raw [6]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("browser-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("browser-%x", raw)
}
