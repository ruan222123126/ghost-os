package externalagent

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

func (m *Manager) prepareRun(req api.ExternalAgentRequest, forceStart bool) (preparedRun, error) {
	if m == nil || m.ConfigStore == nil || m.SessionStore == nil {
		return preparedRun{}, errors.New("external agent manager is not configured")
	}
	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	if provider == "" {
		provider = ProviderCodex
	}
	if provider == ProviderClaude {
		return preparedRun{}, fmt.Errorf("%w: claude", ErrNotImplemented)
	}
	if provider != ProviderCodex {
		return preparedRun{}, fmt.Errorf("unsupported external agent provider: %q", provider)
	}
	if strings.TrimSpace(req.Message) == "" {
		return preparedRun{}, errors.New("message is required")
	}
	cfg, err := m.ConfigStore.Config()
	if err != nil {
		return preparedRun{}, err
	}
	if strings.TrimSpace(req.PermissionMode) == "" {
		req.PermissionMode = cfg.ExternalCodexPermissionMode
	}
	policy, err := ResolveExecutionPolicy(req.PermissionMode)
	if err != nil {
		return preparedRun{}, err
	}
	req.Mode, err = normalizeCodexMode(req.Mode)
	if err != nil {
		return preparedRun{}, err
	}
	cwd := resolveCWD(req.ProjectRoot, cfg.ProjectRoot)
	sess, err := m.prepareSession(req, forceStart, cfg, policy, cwd)
	if err != nil {
		return preparedRun{}, err
	}
	return preparedRun{request: req, session: sess, cfg: cfg, policy: policy, cwd: cwd}, nil
}

func (m *Manager) prepareSession(
	req api.ExternalAgentRequest,
	forceStart bool,
	cfg bridgeconfig.Config,
	policy ExecutionPolicy,
	cwd string,
) (*session.Session, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	var (
		sess *session.Session
		err  error
	)
	if sessionID == "" {
		sess = session.NewSession("")
	} else {
		sess, err = m.SessionStore.Load(sessionID)
		if err != nil {
			return nil, err
		}
		if !forceStart && (sess.ExternalRuntime == nil || strings.TrimSpace(sess.ExternalRuntime.ThreadID) == "") {
			return nil, fmt.Errorf("external runtime is not initialized for session_id=%s", sessionID)
		}
	}
	sess.TurnIndex++
	sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: strings.TrimSpace(req.Message)})
	now := time.Now().UTC()
	ext := cloneOrNewRuntime(sess.ExternalRuntime)
	ext.Provider = ProviderCodex
	ext.Status = StatusRunning
	ext.PermissionMode = policy.PermissionMode
	ext.Mode = strings.TrimSpace(req.Mode)
	ext.Model = strings.TrimSpace(req.Model)
	ext.Effort = strings.TrimSpace(req.Effort)
	ext.CWD = cwd
	ext.ProjectRoot = strings.TrimSpace(req.ProjectRoot)
	if ext.ProjectRoot == "" {
		ext.ProjectRoot = cfg.ProjectRoot
	}
	if ext.StartedAt.IsZero() {
		ext.StartedAt = now
	}
	ext.UpdatedAt = now
	ext.PendingApprovals = nil
	sess.SetExternalRuntime(ext)
	if strings.TrimSpace(sess.Title) == "" {
		sess.Title = externalSessionTitle(req.Message)
	}
	if err := m.SessionStore.Save(sess); err != nil {
		return nil, err
	}
	return sess, nil
}

func (m *Manager) runtimeForSession(sessionID string, cfg bridgeconfig.Config, cwd string) (*runtimeSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sessions == nil {
		m.sessions = make(map[string]*runtimeSession)
	}
	if runtime := m.sessions[sessionID]; runtime != nil {
		return runtime, nil
	}
	factory := m.ClientFactory
	if factory == nil {
		factory = DefaultClientFactory
	}
	runtime := &runtimeSession{
		client: factory(ClientConfig{
			CodexPath: cfg.CodexCLIPath,
			NodePath:  cfg.NodeBinPath,
			CWD:       cwd,
		}),
		pending: make(map[string]chan string),
	}
	m.sessions[sessionID] = runtime
	return runtime, nil
}

func (m *Manager) lookupRuntime(sessionID string) *runtimeSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessions[strings.TrimSpace(sessionID)]
}

func (r *runtimeSession) beginTurn(sessionID string, traceID string, turn int, sink streaming.Sink) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active != nil {
		return ErrExternalRunActive
	}
	r.active = &activeTurn{
		sessionID: sessionID,
		traceID:   traceID,
		turn:      turn,
		sink:      ensureSink(sink),
		done:      make(chan struct{}),
	}
	return nil
}

func (r *runtimeSession) activeSnapshot() *activeTurn {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.active
}

func (r *runtimeSession) activeDoneState() (*activeTurn, <-chan struct{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active == nil {
		ch := make(chan struct{})
		close(ch)
		return nil, ch
	}
	return r.active, r.active.done
}

func (r *runtimeSession) activeResult(active *activeTurn) turnDone {
	r.mu.Lock()
	defer r.mu.Unlock()
	if active != nil {
		return active.result
	}
	if r.active == nil {
		return turnDone{}
	}
	return r.active.result
}

func (r *runtimeSession) clearActive() {
	r.mu.Lock()
	r.active = nil
	r.mu.Unlock()
}

func (r *runtimeSession) finalText() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active == nil {
		return ""
	}
	return strings.TrimSpace(r.active.text.String())
}

func (r *runtimeSession) appendText(text string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active != nil {
		r.active.text.WriteString(text)
		r.active.pendingText.WriteString(text)
	}
}

func (r *runtimeSession) pendingText() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active == nil {
		return ""
	}
	return r.active.pendingText.String()
}

func (r *runtimeSession) clearPendingText() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active == nil {
		return
	}
	r.active.pendingText.Reset()
}

func (r *runtimeSession) finish(done turnDone) {
	r.mu.Lock()
	active := r.active
	if active == nil {
		r.mu.Unlock()
		return
	}
	if !active.finished {
		active.result = done
		active.finished = true
		close(active.done)
	}
	r.mu.Unlock()
}

func resolveCWD(requestRoot string, configRoot string) string {
	root := strings.TrimSpace(requestRoot)
	if root == "" {
		root = strings.TrimSpace(configRoot)
	}
	if root == "" {
		return "."
	}
	if abs, err := filepath.Abs(root); err == nil {
		return abs
	}
	return root
}

func externalSessionTitle(message string) string {
	title := strings.TrimSpace(strings.ReplaceAll(message, "\n", " "))
	if title == "" {
		return "Codex"
	}
	if len(title) > 80 {
		return title[:80]
	}
	return title
}
