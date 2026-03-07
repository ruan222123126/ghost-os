package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/memory"
	"ghost-os/bridge/session"
)

const agentWarmMemoryCapacity = 100

// SessionTurnRunner 为 service 层暴露瘦执行接口，隐藏运行时装配与会话编排细节。
type SessionTurnRunner interface {
	RunTurn(ctx context.Context, message string, sessionID string, traceID string) (string, string, error)
	RunTurnStream(ctx context.Context, message string, sessionID string, traceID string, sink agent.EventSink) (string, string, error)
}

// SessionAgentRunner 负责执行单轮 agent；运行时装配下沉到 sessionTurnPreparer。
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
	sessionStore      *session.Store
	deps              agentRuntimeDependencies
	persistence       *SessionTurnCommitter
	sess              *session.Session
	agent             *agent.Agent
	execCtx           context.Context
	traceID           string
	userMessage       string
	preTurnMessages   []llm.Message
	answeredQuestions []memory.DecisionAnsweredQuestion
	turnStartedAt     time.Time
	environment       memory.DecisionEnvFingerprint
	cleanup           func()
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
	s.captureDecisionTurn(response, awaitingHuman)
	if awaitingHuman {
		return "", s.persistedSessionID(), runErr
	}
	return response, s.persistedSessionID(), nil
}

func (s *sessionTurnState) captureDecisionTurn(response string, awaitingHuman bool) {
	if s == nil || s.persistence == nil || s.persistence.memoryManager == nil || s.agent == nil {
		return
	}
	_, sessionEndSignal, err := parseSessionEndSignal(response)
	if err != nil {
		log.Printf(
			"trace_id=%s action=MEMORY_DECISION_CAPTURE status=skip_invalid_session_end session_id=%s error=%v",
			strings.TrimSpace(s.traceID),
			strings.TrimSpace(s.currentSessionID()),
			err,
		)
		return
	}
	outcome := memory.DecisionOutcomeSuccess
	if awaitingHuman {
		outcome = memory.DecisionOutcomeAwaitingHuman
	}
	input := memory.DecisionCaptureInput{
		Namespace:         strings.TrimSpace(s.environment.GraphNamespace),
		SessionID:         s.currentSessionID(),
		TraceID:           strings.TrimSpace(s.traceID),
		TurnID:            buildDecisionTurnID(s.agent.LastTurn(), s.traceID),
		UserMessage:       strings.TrimSpace(s.userMessage),
		RecentHistory:     llm.CloneMessages(s.preTurnMessages),
		NewMessages:       s.agent.GetNewMessages(),
		Outcome:           outcome,
		AnsweredQuestions: append([]memory.DecisionAnsweredQuestion(nil), s.answeredQuestions...),
		SessionEnded:      sessionEndSignal != nil,
		TurnStartedAt:     s.turnStartedAt,
		TurnFinishedAt:    time.Now().UTC(),
		Environment:       s.environment,
	}
	if err := s.persistence.memoryManager.CaptureDecisionTurn(input); err != nil {
		log.Printf(
			"trace_id=%s action=MEMORY_DECISION_CAPTURE status=error session_id=%s error=%v",
			strings.TrimSpace(s.traceID),
			strings.TrimSpace(s.currentSessionID()),
			err,
		)
	}
}

func buildDecisionTurnID(turn int, traceID string) string {
	trimmedTraceID := strings.TrimSpace(traceID)
	if trimmedTraceID == "" {
		return fmt.Sprintf("turn-%d", turn)
	}
	return fmt.Sprintf("turn-%d-%s", turn, trimmedTraceID)
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
	return r.turnPreparer().prepare(ctx, userMessage, sessionID, traceID)
}

func (r *SessionAgentRunner) turnPreparer() *sessionTurnPreparer {
	if r == nil {
		return newSessionTurnPreparer(nil, nil, nil, nil, nil, nil)
	}
	return newSessionTurnPreparer(
		r.runtimeFactory,
		r.configStore,
		r.sessionStore,
		r.sharedMemoryManager,
		r.runRegistry,
		r.selectorFactory,
	)
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
