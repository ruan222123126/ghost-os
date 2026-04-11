package orchestration

import (
	"context"
	"errors"
	"net/http"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/streaming"
)

func (s *bridgeService) executeAgentStreamAction(ctx context.Context, params agentParams, traceID string, sink streaming.Sink) (string, string, error) {
	trackedSink := newEventTurnTracker(newSessionStreamBroadcastSink(sink, s.sessionPushHub()))
	sessionID := strings.TrimSpace(params.SessionID)

	prepared, err := s.validateAgentStreamRequest(ctx, params, traceID, sessionID, trackedSink)
	if err != nil {
		return "", "", err
	}
	if prepared.mode == agentModePlan {
		return s.executePlanModeStreamTurn(ctx, prepared, traceID, trackedSink)
	}
	handled, message, resolvedSessionID, err := s.tryExecuteProModeStream(ctx, prepared, traceID, sessionID, trackedSink)
	if handled {
		return message, resolvedSessionID, err
	}
	return s.executeStandardAgentStreamTurn(ctx, prepared, traceID, trackedSink)
}

func (s *bridgeService) validateAgentStreamRequest(
	ctx context.Context,
	params agentParams,
	traceID string,
	sessionID string,
	trackedSink *eventTurnTracker,
) (preparedAgentTurnRequest, error) {
	prepared, err := s.validateAgentTurnRequest(params)
	if err == nil {
		return prepared, nil
	}
	if !errors.Is(err, errAgentMessageRequired) {
		logAction(traceID, busActionAgentSend, "error", err)
	}
	code := legacyStatusFromServiceError(err)
	if emitErr := emitStreamErrorEvent(ctx, trackedSink, traceID, 0, "", sessionID, code, err); emitErr != nil {
		return preparedAgentTurnRequest{}, emitErr
	}
	return preparedAgentTurnRequest{}, err
}

func (s *bridgeService) tryExecuteProModeStream(
	ctx context.Context,
	prepared preparedAgentTurnRequest,
	traceID string,
	sessionID string,
	trackedSink *eventTurnTracker,
) (bool, string, string, error) {
	if prepared.mode != agentModeDefault {
		return false, "", "", nil
	}
	_, matched, parseErr := parseProModeRequest(prepared.message, bridgeconfig.DefaultProMaxIterations)
	if parseErr != nil {
		message, resolvedSessionID, err := s.emitProModeParseError(ctx, traceID, sessionID, trackedSink, parseErr)
		return true, message, resolvedSessionID, err
	}
	if !matched {
		return false, "", "", nil
	}
	message, resolvedSessionID, err := s.executeProModeStreamTurn(ctx, prepared, traceID, trackedSink)
	return true, message, resolvedSessionID, err
}

func (s *bridgeService) executePlanModeStreamTurn(
	ctx context.Context,
	prepared preparedAgentTurnRequest,
	traceID string,
	trackedSink *eventTurnTracker,
) (string, string, error) {
	logAction(traceID, busActionAgentSend, "running", nil)
	payload, code, err := s.executePlanModeAction(ctx, prepared, traceID)
	if err != nil {
		statusCode := code
		if statusCode <= 0 {
			statusCode = http.StatusInternalServerError
		}
		if errors.Is(err, context.Canceled) {
			logAction(traceID, busActionAgentSend, "cancelled", err)
			return "", prepared.sessionID, ErrRunCancelled
		}
		logAction(traceID, busActionAgentSend, "error", err)
		if emitErr := emitStreamErrorEvent(ctx, trackedSink, traceID, 0, "", prepared.sessionID, statusCode, err); emitErr != nil {
			return "", "", emitErr
		}
		return "", prepared.sessionID, err
	}
	result := finalizedAgentTurn{
		message:   payload.Message,
		sessionID: payload.SessionID,
	}
	if emitErr := emitDirectAgentStreamResult(ctx, trackedSink, traceID, 0, result); emitErr != nil {
		return "", "", emitErr
	}
	s.publishAssistantSessionPush(traceID, result)
	logAction(traceID, busActionAgentSend, "success", nil)
	return result.message, result.sessionID, nil
}

func (s *bridgeService) emitProModeParseError(
	ctx context.Context,
	traceID string,
	sessionID string,
	trackedSink *eventTurnTracker,
	parseErr error,
) (string, string, error) {
	logAction(traceID, busActionAgentSend, "error", parseErr)
	if emitErr := emitStreamErrorEvent(ctx, trackedSink, traceID, 0, "", sessionID, http.StatusBadRequest, parseErr); emitErr != nil {
		return "", "", emitErr
	}
	return "", "", parseErr
}

func (s *bridgeService) executeProModeStreamTurn(
	ctx context.Context,
	prepared preparedAgentTurnRequest,
	traceID string,
	trackedSink *eventTurnTracker,
) (string, string, error) {
	logAction(traceID, busActionAgentSend, "running", nil)
	payload, code, err := s.executeProModeAction(ctx, prepared, traceID)
	if err != nil {
		kind, normalizedErr := normalizeAgentExecutionError(err)
		statusCode := legacyStatusFromServiceErrorKind(kind)
		if code > 0 {
			statusCode = code
		}
		if errors.Is(normalizedErr, ErrRunCancelled) {
			logAction(traceID, busActionAgentSend, "cancelled", normalizedErr)
			return "", prepared.sessionID, normalizedErr
		}
		logAction(traceID, busActionAgentSend, "error", normalizedErr)
		if emitErr := emitStreamErrorEvent(ctx, trackedSink, traceID, 0, "", prepared.sessionID, statusCode, normalizedErr); emitErr != nil {
			return "", "", emitErr
		}
		return "", prepared.sessionID, normalizedErr
	}

	result := finalizedAgentTurn{
		message:   payload.Message,
		sessionID: payload.SessionID,
	}
	if emitErr := emitDirectAgentStreamResult(ctx, trackedSink, traceID, 0, result); emitErr != nil {
		return "", "", emitErr
	}
	s.publishAssistantSessionPush(traceID, result)
	logAction(traceID, busActionAgentSend, "success", nil)
	return result.message, result.sessionID, nil
}

func (s *bridgeService) executeStandardAgentStreamTurn(
	ctx context.Context,
	prepared preparedAgentTurnRequest,
	traceID string,
	trackedSink *eventTurnTracker,
) (string, string, error) {
	logAction(traceID, busActionAgentSend, "running", nil)
	response, sessionID, err := s.runPreparedAgentTurnStream(ctx, prepared, traceID, trackedSink)
	if err != nil {
		awaitingErr, _, cancelled, normalizedErr := classifyAgentTurnError(err)
		if awaitingErr != nil {
			logAction(traceID, busActionAgentSend, "awaiting_human", nil)
			s.publishAwaitingHumanSessionPush(traceID, sessionID, awaitingErr)
			return "", sessionID, err
		}
		if cancelled {
			logAction(traceID, busActionAgentSend, "cancelled", normalizedErr)
			return "", sessionID, normalizedErr
		}
		logAction(traceID, busActionAgentSend, "error", normalizedErr)
		return "", sessionID, normalizedErr
	}

	result, err := s.finalizeAgentTurn(response, sessionID)
	if err != nil {
		logAction(traceID, busActionAgentSend, "error", err)
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
			sessionID,
			legacyStatusFromServiceError(err),
			err,
		); emitErr != nil {
			return "", "", emitErr
		}
		return "", "", err
	}

	s.publishAssistantSessionPush(traceID, result)
	logAction(traceID, busActionAgentSend, "success", nil)
	return result.message, result.sessionID, nil
}
