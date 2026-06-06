package agent

import (
	"context"
	"errors"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/orchestration/internal/app/agentturn"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

type ExecutorFunc func(
	ctx context.Context,
	message string,
	sessionID string,
	traceID string,
	store bridgeconfig.Store,
	sessionStore *session.Store,
) (string, string, error)

type StreamExecutorFunc func(
	ctx context.Context,
	message string,
	sessionID string,
	traceID string,
	store bridgeconfig.Store,
	sessionStore *session.Store,
	sink streaming.Sink,
) (string, string, error)

type ExecutorRunner struct {
	store          bridgeconfig.Store
	sessionStore   *session.Store
	executor       ExecutorFunc
	streamExecutor StreamExecutorFunc
}

func NewExecutorRunner(
	store bridgeconfig.Store,
	sessionStore *session.Store,
	executor ExecutorFunc,
	streamExecutor StreamExecutorFunc,
) *ExecutorRunner {
	if executor == nil && streamExecutor == nil {
		return nil
	}
	return &ExecutorRunner{
		store:          store,
		sessionStore:   sessionStore,
		executor:       executor,
		streamExecutor: streamExecutor,
	}
}

func (r *ExecutorRunner) RunTurn(
	ctx context.Context,
	message string,
	sessionID string,
	traceID string,
) (string, string, error) {
	if r == nil || r.executor == nil {
		return "", "", errors.New("agent runner is not configured")
	}
	return r.executor(ctx, message, sessionID, traceID, r.store, r.sessionStore)
}

func (r *ExecutorRunner) RunTurnStream(
	ctx context.Context,
	message string,
	sessionID string,
	traceID string,
	sink streaming.Sink,
) (string, string, error) {
	if r == nil {
		return "", "", errors.New("agent runner is not configured")
	}
	if r.streamExecutor != nil {
		return r.streamExecutor(ctx, message, sessionID, traceID, r.store, r.sessionStore, sink)
	}
	return r.RunTurn(ctx, message, sessionID, traceID)
}

func (r *ExecutorRunner) WithRequestRuntimeOptions(options *agentturn.RequestRuntimeOptions) any {
	if r == nil || options == nil {
		return r
	}
	cloned := *r
	cloned.store = bridgeconfig.WithProjectRootOverride(r.store, options.ProjectRoot)
	return &cloned
}
