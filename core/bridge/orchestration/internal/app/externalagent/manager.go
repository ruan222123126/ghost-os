package externalagent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/orchestration/internal/contracts/api"
	internaltrace "ghost-os/bridge/orchestration/internal/trace"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

type Manager struct {
	ConfigStore   bridgeconfig.Store
	SessionStore  *session.Store
	ClientFactory ClientFactory

	mu       sync.Mutex
	sessions map[string]*runtimeSession
}

type runtimeSession struct {
	mu      sync.Mutex
	client  CodexClient
	active  *activeTurn
	pending map[string]chan string
}

type activeTurn struct {
	sessionID   string
	traceID     string
	turn        int
	sink        streaming.Sink
	done        chan struct{}
	result      turnDone
	finished    bool
	text        strings.Builder
	pendingText strings.Builder
}

type turnDone struct {
	aborted bool
	err     error
}

type preparedRun struct {
	request api.ExternalAgentRequest
	session *session.Session
	cfg     bridgeconfig.Config
	policy  ExecutionPolicy
	cwd     string
}

func NewManager(configStore bridgeconfig.Store, sessionStore *session.Store) *Manager {
	return &Manager{
		ConfigStore:   configStore,
		SessionStore:  sessionStore,
		ClientFactory: DefaultClientFactory,
		sessions:      make(map[string]*runtimeSession),
	}
}

func (m *Manager) ExecuteStream(
	ctx context.Context,
	req api.ExternalAgentRequest,
	traceID string,
	sink streaming.Sink,
	forceStart bool,
) (string, string, error) {
	prepared, err := m.prepareRun(req, forceStart)
	if err != nil {
		return "", strings.TrimSpace(req.SessionID), err
	}
	sessionID := prepared.session.ID
	runtime, err := m.runtimeForSession(sessionID, prepared.cfg, prepared.cwd)
	if err != nil {
		return "", sessionID, err
	}
	runSink := internaltrace.NewSessionDraftStoreCheckpointSink(sink, m.SessionStore, sessionID)
	if err := runtime.beginTurn(sessionID, traceID, prepared.session.TurnIndex, runSink); err != nil {
		return "", sessionID, err
	}
	defer runtime.clearActive()

	runtime.client.SetEventHandler(func(event CodexEvent) {
		m.handleEvent(ctx, runtime, event)
	})
	runtime.client.SetApprovalHandler(func(ctx context.Context, approval ApprovalRequest) (string, error) {
		return m.handleApproval(ctx, runtime, approval)
	})

	if err := runtime.client.Connect(ctx); err != nil {
		_ = m.finishWithError(ctx, prepared.session.ID, traceID, prepared.session.TurnIndex, runSink, err)
		return "", sessionID, err
	}
	threadResult, err := m.ensureThread(ctx, runtime.client, prepared)
	if err != nil {
		_ = m.finishWithError(ctx, sessionID, traceID, prepared.session.TurnIndex, runSink, err)
		return "", sessionID, err
	}
	threadID := strings.TrimSpace(threadResult.ThreadID)
	if err := runtime.client.SetCollaborationMode(ctx, CollaborationModeOptions{
		ThreadID: threadID,
		Mode:     prepared.request.Mode,
		Model:    resolveCollaborationModeModel(prepared.request.Model, threadResult.Model),
		Effort:   prepared.request.Effort,
	}); err != nil {
		_ = m.finishWithError(ctx, sessionID, traceID, prepared.session.TurnIndex, runSink, err)
		return "", sessionID, err
	}
	if err := m.updateRuntimeState(sessionID, func(ext *session.ExternalRuntime) {
		ext.ThreadID = threadID
		ext.Status = StatusRunning
	}); err != nil {
		return "", sessionID, err
	}
	if err := emit(ctx, runSink, traceID, sessionID, prepared.session.TurnIndex, "", streaming.EventRunStarted, map[string]any{
		"session_id": sessionID,
		"provider":   ProviderCodex,
		"thread_id":  threadID,
	}); err != nil {
		return "", sessionID, err
	}
	turnID, err := runtime.client.StartTurn(ctx, TurnOptions{
		ThreadID:       threadID,
		Message:        prepared.request.Message,
		Model:          prepared.request.Model,
		CWD:            prepared.cwd,
		ApprovalPolicy: prepared.policy.ApprovalPolicy,
		Sandbox:        prepared.policy.Sandbox,
		Effort:         prepared.request.Effort,
	})
	if err != nil {
		_ = m.finishWithError(ctx, sessionID, traceID, prepared.session.TurnIndex, runSink, err)
		return "", sessionID, err
	}
	if strings.TrimSpace(turnID) != "" && runtime.activeTurnRunning() {
		_ = m.updateRuntimeState(sessionID, func(ext *session.ExternalRuntime) {
			ext.TurnID = turnID
			ext.Status = StatusRunning
		})
	}
	active, activeDone := runtime.activeDoneState()
	select {
	case <-ctx.Done():
		_ = runtime.client.InterruptTurn(context.Background(), threadID, turnID)
		_ = m.updateRuntimeState(sessionID, func(ext *session.ExternalRuntime) {
			ext.Status = StatusIdle
			ext.TurnID = ""
			ext.PendingApprovals = nil
		})
		return "", sessionID, ctx.Err()
	case <-activeDone:
		done := runtime.activeResult(active)
		if done.err != nil {
			return "", sessionID, done.err
		}
		_ = m.updateRuntimeState(sessionID, func(ext *session.ExternalRuntime) {
			ext.Status = StatusIdle
			ext.TurnID = ""
			ext.PendingApprovals = nil
		})
		final := runtime.finalText()
		return final, sessionID, nil
	}
}

func (m *Manager) ensureThread(ctx context.Context, client CodexClient, prepared preparedRun) (ThreadResult, error) {
	ext := prepared.session.ExternalRuntime
	opts := ThreadOptions{
		Model:          prepared.request.Model,
		CWD:            prepared.cwd,
		ApprovalPolicy: prepared.policy.ApprovalPolicy,
		Sandbox:        prepared.policy.Sandbox,
	}
	if ext != nil && strings.TrimSpace(ext.ThreadID) != "" {
		opts.ThreadID = ext.ThreadID
		return client.ResumeThread(ctx, opts)
	}
	return client.StartThread(ctx, opts)
}

func normalizeCodexMode(raw string) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(raw))
	switch mode {
	case "":
		return "", nil
	case CodexModeDefault:
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported external codex mode: %q", mode)
	}
}

func resolveCollaborationModeModel(requestModel string, threadModel string) string {
	if model := strings.TrimSpace(requestModel); model != "" {
		return model
	}
	return strings.TrimSpace(threadModel)
}

func (m *Manager) Stop(ctx context.Context, req api.ExternalAgentStopParams) (api.ExternalAgentResponse, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return api.ExternalAgentResponse{}, ErrSessionRequired
	}
	runtime := m.lookupRuntime(sessionID)
	if runtime == nil {
		return api.ExternalAgentResponse{}, ErrExternalRunMissing
	}
	ext, err := m.externalRuntime(sessionID)
	if err != nil {
		return api.ExternalAgentResponse{}, err
	}
	if ext == nil || strings.TrimSpace(ext.ThreadID) == "" {
		return api.ExternalAgentResponse{}, ErrExternalRunMissing
	}
	turnID := strings.TrimSpace(ext.TurnID)
	if turnID == "" {
		return api.ExternalAgentResponse{}, ErrExternalRunMissing
	}
	active, done := runtime.activeDoneState()
	if err := runtime.client.InterruptTurn(ctx, ext.ThreadID, turnID); err != nil {
		return api.ExternalAgentResponse{}, err
	}
	select {
	case <-ctx.Done():
		return api.ExternalAgentResponse{}, ctx.Err()
	case <-done:
		result := runtime.activeResult(active)
		if result.err != nil {
			return api.ExternalAgentResponse{}, result.err
		}
	}
	_ = m.updateRuntimeState(sessionID, func(ext *session.ExternalRuntime) {
		ext.Status = StatusIdle
		ext.TurnID = ""
		ext.PendingApprovals = nil
	})
	return api.ExternalAgentResponse{
		Status:    StatusIdle,
		Provider:  ProviderCodex,
		SessionID: sessionID,
		ThreadID:  ext.ThreadID,
	}, nil
}

func (m *Manager) Approve(req api.ExternalAgentApprovalParams) (api.ExternalAgentApprovalResponse, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	approvalID := strings.TrimSpace(req.ApprovalID)
	decision, err := ValidateApprovalDecision(req.Decision)
	if err != nil {
		return api.ExternalAgentApprovalResponse{}, err
	}
	runtime := m.lookupRuntime(sessionID)
	if runtime == nil {
		return api.ExternalAgentApprovalResponse{}, ErrApprovalNotFound
	}
	if err := runtime.resolveApproval(approvalID, decision); err != nil {
		return api.ExternalAgentApprovalResponse{}, err
	}
	return api.ExternalAgentApprovalResponse{
		SessionID:  sessionID,
		ApprovalID: approvalID,
		Decision:   decision,
		Accepted:   true,
	}, nil
}

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, runtime := range m.sessions {
		runtime.mu.Lock()
		if runtime.client != nil {
			_ = runtime.client.Close()
		}
		runtime.mu.Unlock()
	}
	m.sessions = make(map[string]*runtimeSession)
}
