package agentturn

import (
	"context"
	"errors"

	"ghost-os/bridge/orchestration/internal/contracts/bus"
	"ghost-os/bridge/streaming"
)

func (s Service) Resume(ctx context.Context, sessionID string, traceID string) (bus.ServiceResult, error) {
	if s.Runner == nil || s.Finalizer == nil {
		err := errors.New("agent turn runner is not configured")
		return bus.ServiceResult{}, bus.WrapError(bus.ServiceErrorInternal, err)
	}
	response, resumedSessionID, err := s.Runner.RunTurn(ctx, PreparedRequest{SessionID: sessionID}, traceID)
	if err != nil {
		return s.handleResumeError(traceID, resumedSessionID, err)
	}
	return s.completeResumeTurn(traceID, response, resumedSessionID)
}

func (s Service) handleResumeError(
	traceID string,
	sessionID string,
	err error,
) (bus.ServiceResult, error) {
	awaitingErr, kind, _, normalizedErr := s.classify(err)
	if awaitingErr != nil {
		s.publishAwaiting(traceID, sessionID, awaitingErr)
		return bus.ResultAccepted(NewAwaitingHumanResponse(sessionID, awaitingErr)), nil
	}
	return bus.ServiceResult{}, bus.WrapError(kind, normalizedErr)
}

func (s Service) completeResumeTurn(
	traceID string,
	response string,
	sessionID string,
) (bus.ServiceResult, error) {
	result, err := s.Finalizer.Finalize(response, sessionID)
	if err != nil {
		return bus.ServiceResult{}, bus.WrapError(bus.ErrorKindOf(err), err)
	}
	payload, err := s.Finalizer.NewResponsePayload(result)
	if err != nil {
		return bus.ServiceResult{}, bus.WrapError(bus.ServiceErrorInternal, err)
	}
	s.publishAssistant(traceID, result)
	return bus.ResultSuccess(payload), nil
}

func (s Service) ResumeStream(
	ctx context.Context,
	sessionID string,
	traceID string,
	sink streaming.Sink,
) (string, string, error) {
	if s.Runner == nil || s.Finalizer == nil {
		return "", sessionID, errors.New("agent turn runner is not configured")
	}
	trackedSink := newEventTurnTracker(sink)
	response, resumedSessionID, err := s.Runner.RunTurnStream(
		ctx,
		PreparedRequest{SessionID: sessionID},
		traceID,
		trackedSink,
	)
	if err != nil {
		return s.handleResumeStreamError(traceID, resumedSessionID, err)
	}
	return s.completeResumeStream(ctx, traceID, response, resumedSessionID, trackedSink)
}

func (s Service) handleResumeStreamError(
	traceID string,
	sessionID string,
	err error,
) (string, string, error) {
	awaitingErr, _, cancelled, normalizedErr := s.classify(err)
	if awaitingErr != nil {
		s.publishAwaiting(traceID, sessionID, awaitingErr)
		return "", sessionID, err
	}
	if cancelled {
		return "", sessionID, normalizedErr
	}
	return "", sessionID, normalizedErr
}

func (s Service) completeResumeStream(
	ctx context.Context,
	traceID string,
	response string,
	sessionID string,
	sink *eventTurnTracker,
) (string, string, error) {
	result, err := s.Finalizer.Finalize(response, sessionID)
	if err != nil {
		return s.emitFinalizeStreamError(ctx, traceID, sessionID, sink, err)
	}
	s.publishAssistant(traceID, result)
	return result.Message, result.SessionID, nil
}
