package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/memory"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

const agentWarmMemoryCapacity = 100

// SessionTurnRunner 为 service 层暴露瘦执行接口，隐藏运行时装配与会话编排细节。
type SessionTurnRunner interface {
	RunTurn(ctx context.Context, message string, sessionID string, traceID string) (string, string, error)
	RunTurnStream(ctx context.Context, message string, sessionID string, traceID string, sink agent.EventSink) (string, string, error)
}

// SessionAgentRunner 负责串起会话加载、历史恢复、Agent 执行与回合提交。
type SessionAgentRunner struct {
	runtimeFactory      AgentRuntimeFactory
	configStore         *ConfigStore
	sessionStore        *session.Store
	sharedMemoryManager *memory.MemoryManager
	runRegistry         *RunRegistry
	selectorFactory     func(Config) selectorEngine
}

type sessionTurnSetupError struct {
	sessionID  string
	statusCode int
	err        error
}

func (e *sessionTurnSetupError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *sessionTurnSetupError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

type sessionTurnState struct {
	sessionStore *session.Store
	deps         agentRuntimeDependencies
	persistence  *SessionTurnCommitter
	sess         *session.Session
	agent        *agent.Agent
	execCtx      context.Context
	traceID      string
	cleanup      func()
}

func (s *sessionTurnState) close() {
	if s == nil {
		return
	}
	if s.persistence != nil && s.sess != nil {
		s.persistence.ArchiveSession(s.sess.ID)
	}
	if s.cleanup != nil {
		s.cleanup()
	}
	s.deps.Close()
}

func (s *sessionTurnState) currentSessionID() string {
	if s == nil || s.sess == nil {
		return ""
	}
	return strings.TrimSpace(s.sess.ID)
}

func (s *sessionTurnState) persistedSessionID() string {
	if s == nil || s.sessionStore == nil {
		return ""
	}
	return s.currentSessionID()
}

func (s *sessionTurnState) persistNewMessages() error {
	if s == nil || s.persistence == nil || s.sess == nil || s.agent == nil {
		return nil
	}
	return s.persistence.PersistSessionMessages(s.sess, s.agent.GetNewMessages())
}

func (s *sessionTurnState) complete(
	response string,
	runErr error,
	onPersistErr func(err error, awaitingHuman bool) error,
) (string, string, error) {
	awaitingHuman := false
	if runErr != nil {
		var awaitingErr *agent.ErrAwaitingHuman
		if !errors.As(runErr, &awaitingErr) {
			return "", "", runErr
		}
		awaitingHuman = true
	}

	if saveErr := s.persistNewMessages(); saveErr != nil {
		if onPersistErr != nil {
			if emitErr := onPersistErr(saveErr, awaitingHuman); emitErr != nil {
				return "", "", emitErr
			}
		}
		return "", "", saveErr
	}
	if awaitingHuman {
		return "", s.persistedSessionID(), runErr
	}
	return response, s.persistedSessionID(), nil
}

func NewSessionAgentRunner(
	runtimeFactory AgentRuntimeFactory,
	configStore *ConfigStore,
	sessionStore *session.Store,
	sharedMemoryManager *memory.MemoryManager,
	runRegistry *RunRegistry,
) *SessionAgentRunner {
	if runtimeFactory == nil {
		runtimeFactory = newAgentRuntimeFactory()
	}
	return &SessionAgentRunner{
		runtimeFactory:      runtimeFactory,
		configStore:         configStore,
		sessionStore:        sessionStore,
		sharedMemoryManager: sharedMemoryManager,
		runRegistry:         runRegistry,
	}
}

func (r *SessionAgentRunner) prepareTurn(ctx context.Context, userMessage string, sessionID string, traceID string) (*sessionTurnState, error) {
	deps, err := r.runtimeFactory.Build(r.configStore)
	if err != nil {
		return nil, err
	}

	historyBuilder := newSessionHistoryBuilder(deps.cfg, deps.systemPrompt, r.sessionStore)
	persistence := newSessionTurnCommitter(r.sessionStore, r.memoryManager(deps.cfg), traceID)

	sess, err := historyBuilder.LoadOrCreateSession(sessionID)
	if err != nil {
		deps.Close()
		return nil, &sessionTurnSetupError{
			sessionID: strings.TrimSpace(sessionID),
			err:       err,
		}
	}

	execCtx, cleanup, err := r.registerRun(ctx, sess.ID, traceID)
	if err != nil {
		deps.Close()
		return nil, &sessionTurnSetupError{
			sessionID:  strings.TrimSpace(sess.ID),
			statusCode: http.StatusConflict,
			err:        err,
		}
	}

	askHumanContinuation := hasAnsweredHumanResponse(sess)
	history := historyBuilder.BuildHistory(sess)
	catalog, systemPrompt := r.selectToolsForTurn(execCtx, deps, history, userMessage, askHumanContinuation, traceID)
	if systemPrompt != "" {
		history.UpdateSystemPrompt(systemPrompt)
	}
	r.injectAutoRecall(history, userMessage, sess.ID, traceID, persistence.memoryManager)

	return &sessionTurnState{
		sessionStore: r.sessionStore,
		deps:         deps,
		persistence:  persistence,
		sess:         sess,
		agent:        agent.NewAgentWithHistory(deps.client, catalog, history, deps.cfg.MaxTurns),
		execCtx:      tools.WithSession(execCtx, sess),
		traceID:      strings.TrimSpace(traceID),
		cleanup:      cleanup,
	}, nil
}

func (r *SessionAgentRunner) RunTurn(ctx context.Context, userMessage string, sessionID string, traceID string) (string, string, error) {
	turn, err := r.prepareTurn(ctx, userMessage, sessionID, traceID)
	if err != nil {
		return "", "", err
	}
	defer turn.close()

	response, runErr := turn.agent.RunWithTraceID(turn.execCtx, userMessage, turn.traceID)
	return turn.complete(response, runErr, nil)
}

func (r *SessionAgentRunner) RunTurnStream(ctx context.Context, userMessage string, sessionID string, traceID string, sink agent.EventSink) (string, string, error) {
	streamSink := ensureEventSink(sink)
	turn, err := r.prepareTurn(ctx, userMessage, sessionID, traceID)
	if err != nil {
		var setupErr *sessionTurnSetupError
		if errors.As(err, &setupErr) {
			if emitErr := emitStreamErrorEvent(ctx, streamSink, traceID, 0, "", setupErr.sessionID, setupErr.statusCode, setupErr); emitErr != nil {
				return "", "", emitErr
			}
		}
		return "", "", err
	}
	defer turn.close()

	if emitErr := emitStreamEvent(ctx, streamSink, agent.NewEvent(turn.traceID, 0, "", agent.EventRunStarted, map[string]any{
		"session_id": turn.currentSessionID(),
	})); emitErr != nil {
		return "", "", emitErr
	}

	response, runErr := turn.agent.RunStreamWithTraceID(turn.execCtx, userMessage, turn.traceID, streamSink)
	return turn.complete(response, runErr, func(err error, awaitingHuman bool) error {
		turnNumber := turn.agent.LastTurn()
		stepID := agent.AssistantStepID(turnNumber)
		if awaitingHuman {
			stepID = ""
		}
		return emitStreamErrorEvent(ctx, streamSink, turn.traceID, turnNumber, stepID, turn.currentSessionID(), 0, err)
	})
}

func (r *SessionAgentRunner) selectToolsForTurn(
	ctx context.Context,
	deps agentRuntimeDependencies,
	history *agent.History,
	userMessage string,
	askHumanContinuation bool,
	traceID string,
) (tools.ToolCatalog, string) {
	if askHumanContinuation {
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=ask_human_continuation", strings.TrimSpace(traceID))
		return deps.registry, ""
	}
	if !deps.cfg.ToolSelectorEnabled {
		return deps.registry, ""
	}

	selector := r.newSelector(deps.cfg)
	if selector == nil {
		return deps.registry, ""
	}

	recentMessages := getRecentMessages(history, deps.cfg.ToolSelectorRecentMsgs)
	result := selector.SelectTools(ctx, userMessage, recentMessages, traceID)
	if deps.cfg.ToolSelectorShadow {
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=shadow mode=%s tools=%v confidence=%.2f fallback=%t reason=%q error=%v", strings.TrimSpace(traceID), result.Mode, result.Tools, result.Confidence, result.Fallback, result.Reason, result.Error)
		return deps.registry, ""
	}
	if result.Mode != "subset" || result.Fallback {
		return deps.registry, ""
	}

	scoped := tools.NewScopedCatalog(deps.registry, result.Tools)
	return scoped, buildSystemPromptForCatalog(deps.cfg, scoped)
}

func (r *SessionAgentRunner) newSelector(cfg Config) selectorEngine {
	if r != nil && r.selectorFactory != nil {
		return r.selectorFactory(cfg)
	}
	return newToolSelectorFromConfig(cfg)
}

func getRecentMessages(history *agent.History, limit int) []llm.Message {
	if history == nil || limit <= 0 {
		return nil
	}

	messages := history.Messages()
	filtered := make([]llm.Message, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case llm.RoleUser, llm.RoleAssistant:
			filtered = append(filtered, msg)
		}
	}
	if len(filtered) <= limit {
		return llm.CloneMessages(filtered)
	}
	return llm.CloneMessages(filtered[len(filtered)-limit:])
}

func hasAnsweredHumanResponse(sess *session.Session) bool {
	return sess != nil && len(sess.HumanAnswers) > 0
}

func (r *SessionAgentRunner) registerRun(ctx context.Context, sessionID string, traceID string) (context.Context, func(), error) {
	if r == nil || r.runRegistry == nil {
		return ctx, func() {}, nil
	}

	trimmedSessionID := strings.TrimSpace(sessionID)
	if trimmedSessionID == "" {
		return ctx, func() {}, nil
	}

	execCtx, cancel := context.WithCancel(ctx)
	if err := r.runRegistry.Register(trimmedSessionID, strings.TrimSpace(traceID), cancel); err != nil {
		cancel()
		return ctx, func() {}, err
	}

	return execCtx, func() {
		r.runRegistry.Unregister(trimmedSessionID)
		cancel()
	}, nil
}

func (r *SessionAgentRunner) memoryManager(cfg Config) *memory.MemoryManager {
	if r == nil {
		return nil
	}
	if r.sharedMemoryManager != nil {
		return r.sharedMemoryManager
	}

	return memory.NewMemoryManager(memoryManagerConfigFromAppConfig(cfg, r.sessionStore, newMemoryWorkerSummarizer(r.configStore)))
}

// injectAutoRecall 在执行前把 warm 层召回上下文注入为 system 消息。
func (r *SessionAgentRunner) injectAutoRecall(history *agent.History, userMessage string, sessionID string, traceID string, memoryManager *memory.MemoryManager) {
	if memoryManager == nil || history == nil {
		return
	}

	contextWindow, err := memoryManager.BuildContextWindowWithScope(memory.SessionScope{
		SessionID: sessionID,
		History:   history,
	}, userMessage)
	if err != nil {
		log.Printf(
			"trace_id=%s action=MEMORY_AUTO_RECALL status=error session_id=%s error=%v",
			strings.TrimSpace(traceID),
			strings.TrimSpace(sessionID),
			err,
		)
		return
	}
	for _, msg := range contextWindow {
		history.Append(msg)
	}
	if len(contextWindow) > 0 {
		log.Printf(
			"trace_id=%s action=MEMORY_AUTO_RECALL status=success session_id=%s injected=%d",
			strings.TrimSpace(traceID),
			strings.TrimSpace(sessionID),
			len(contextWindow),
		)
	}
}

type sessionTurnRunnerAdapter struct {
	store          *ConfigStore
	sessionStore   *session.Store
	executor       agentExecutorFunc
	streamExecutor agentStreamExecutorFunc
}

func newSessionTurnRunnerAdapter(
	store *ConfigStore,
	sessionStore *session.Store,
	executor agentExecutorFunc,
	streamExecutor agentStreamExecutorFunc,
) SessionTurnRunner {
	if executor == nil && streamExecutor == nil {
		return nil
	}
	return &sessionTurnRunnerAdapter{
		store:          store,
		sessionStore:   sessionStore,
		executor:       executor,
		streamExecutor: streamExecutor,
	}
}

func (a *sessionTurnRunnerAdapter) RunTurn(ctx context.Context, message string, sessionID string, traceID string) (string, string, error) {
	if a == nil || a.executor == nil {
		return "", "", errors.New("agent runner is not configured")
	}
	return a.executor(ctx, message, sessionID, traceID, a.store, a.sessionStore)
}

func (a *sessionTurnRunnerAdapter) RunTurnStream(ctx context.Context, message string, sessionID string, traceID string, sink agent.EventSink) (string, string, error) {
	if a == nil {
		return "", "", errors.New("agent runner is not configured")
	}
	if a.streamExecutor != nil {
		return a.streamExecutor(ctx, message, sessionID, traceID, a.store, a.sessionStore, sink)
	}
	return a.RunTurn(ctx, message, sessionID, traceID)
}
