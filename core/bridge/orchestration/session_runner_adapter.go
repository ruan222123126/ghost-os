package orchestration

import (
	"context"
	"errors"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/orchestration/internal/app/agentturn"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

type requestRuntimeOptions = agentturn.RequestRuntimeOptions

type requestRuntimeAwareRunner interface {
	withRequestRuntimeOptions(*requestRuntimeOptions) SessionTurnRunner
}

func normalizeRequestRuntimeOptions(rawProjectRoot string) (*requestRuntimeOptions, error) {
	return agentturn.NormalizeRequestRuntimeOptions(rawProjectRoot)
}

func applyRequestRuntimeOptionsToStore(
	store bridgeconfig.Store,
	options *requestRuntimeOptions,
) bridgeconfig.Store {
	if options == nil {
		return store
	}
	return bridgeconfig.WithProjectRootOverride(store, options.ProjectRoot)
}

func cloneRequestRuntimeOptions(input *requestRuntimeOptions) *requestRuntimeOptions {
	return agentturn.CloneRequestRuntimeOptions(input)
}

type sessionTurnRunnerAdapter struct {
	store          bridgeconfig.Store
	sessionStore   *session.Store
	executor       agentExecutorFunc
	streamExecutor agentStreamExecutorFunc
}

func newSessionTurnRunnerAdapter(
	store bridgeconfig.Store,
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

func (a *sessionTurnRunnerAdapter) withRequestRuntimeOptions(options *requestRuntimeOptions) SessionTurnRunner {
	if a == nil || options == nil {
		return a
	}
	cloned := *a
	cloned.store = applyRequestRuntimeOptionsToStore(a.store, options)
	return &cloned
}

type sessionResumeRunner struct {
	runner   SessionTurnRunner
	pushes   sessionPushAdapter
	finalize func(response string, sessionID string) (finalizedAgentTurn, error)
}

func (s *bridgeService) sessionResumeRunner() sessionResumeRunner {
	return sessionResumeRunner{
		runner:   s.agentRunner,
		pushes:   s.sessionPushAdapter(),
		finalize: s.finalizeAgentTurn,
	}
}

func (r sessionResumeRunner) Resume(ctx context.Context, sessionID string, traceID string) (ServiceResult, error) {
	response, resumedSessionID, err := r.runner.RunTurn(ctx, "", sessionID, traceID)
	if err != nil {
		awaitingErr, kind, _, normalizedErr := classifyAgentTurnError(err)
		if awaitingErr != nil {
			r.pushes.publishAwaitingHuman(traceID, resumedSessionID, awaitingErr)
			return serviceResultAccepted(newAwaitingHumanResponse(resumedSessionID, awaitingErr)), nil
		}
		return ServiceResult{}, wrapServiceError(kind, normalizedErr)
	}

	result, err := r.finalize(response, resumedSessionID)
	if err != nil {
		return ServiceResult{}, wrapServiceError(ServiceErrorKindOf(err), err)
	}
	payload, err := newAgentResponsePayload(result.message, result.sessionID, result.sessionEnd, agentResponseMeta{})
	if err != nil {
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}
	r.pushes.publishAssistant(traceID, result)
	return serviceResultSuccess(payload), nil
}

func (r sessionResumeRunner) ResumeStream(
	ctx context.Context,
	sessionID string,
	traceID string,
	sink streaming.Sink,
) (string, string, error) {
	trackedSink := newEventTurnTracker(newSessionStreamBroadcastSink(sink, r.pushes.hub))
	response, resumedSessionID, err := r.runner.RunTurnStream(ctx, "", sessionID, traceID, trackedSink)
	if err != nil {
		awaitingErr, _, cancelled, normalizedErr := classifyAgentTurnError(err)
		if awaitingErr != nil {
			r.pushes.publishAwaitingHuman(traceID, resumedSessionID, awaitingErr)
			return "", resumedSessionID, err
		}
		if cancelled {
			return "", resumedSessionID, normalizedErr
		}
		return "", resumedSessionID, normalizedErr
	}

	result, err := r.finalize(response, resumedSessionID)
	if err != nil {
		stepID, stepErr := streaming.AssistantStepID(trackedSink.finalAssistantTurn())
		if stepErr != nil {
			return "", "", stepErr
		}
		if emitErr := emitStreamErrorEvent(
			ctx,
			trackedSink,
			traceID,
			trackedSink.finalAssistantTurn(),
			stepID,
			resumedSessionID,
			legacyStatusFromServiceError(err),
			err,
		); emitErr != nil {
			return "", "", emitErr
		}
		return "", "", err
	}

	r.pushes.publishAssistant(traceID, result)
	return result.message, result.sessionID, nil
}
