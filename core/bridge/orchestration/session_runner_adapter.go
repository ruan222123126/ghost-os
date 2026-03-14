package orchestration

import (
	"context"
	"errors"

	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

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

func (a *sessionTurnRunnerAdapter) RunTurnStream(ctx context.Context, message string, sessionID string, traceID string, sink streaming.Sink) (string, string, error) {
	if a == nil {
		return "", "", errors.New("agent runner is not configured")
	}
	if a.streamExecutor != nil {
		return a.streamExecutor(ctx, message, sessionID, traceID, a.store, a.sessionStore, sink)
	}
	return a.RunTurn(ctx, message, sessionID, traceID)
}
