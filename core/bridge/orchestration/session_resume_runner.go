package orchestration

import (
	"context"
	"net/http"

	"ghost-os/bridge/streaming"
)

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

func (r sessionResumeRunner) Resume(ctx context.Context, sessionID string, traceID string) (any, int, error) {
	response, resumedSessionID, err := r.runner.RunTurn(ctx, "", sessionID, traceID)
	if err != nil {
		awaitingErr, kind, _, normalizedErr := classifyAgentTurnError(err)
		if awaitingErr != nil {
			r.pushes.publishAwaitingHuman(traceID, resumedSessionID, awaitingErr)
			return newAwaitingHumanResponse(resumedSessionID, awaitingErr), http.StatusAccepted, nil
		}
		return nil, legacyStatusFromServiceErrorKind(kind), normalizedErr
	}

	result, err := r.finalize(response, resumedSessionID)
	if err != nil {
		return nil, legacyStatusFromServiceError(err), err
	}
	payload, err := newAgentResponsePayload(result.message, result.sessionID, result.sessionEnd, agentResponseMeta{})
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	r.pushes.publishAssistant(traceID, result)
	return payload, http.StatusOK, nil
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
