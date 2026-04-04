// Agent turn use cases exposed to the transport layer.

package orchestration

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"ghost-os/bridge/streaming"
)

// executeAgentAction 执行一次 Agent 回合，并处理“等待人工回答”的中断状态。
func (s *bridgeService) executeAgentAction(ctx context.Context, params agentParams, traceID string) (any, int, error) {
	prepared, code, err := s.validateAgentTurnRequest(params)
	if err != nil {
		if !errors.Is(err, errAgentMessageRequired) {
			logAction(traceID, busActionAgentSend, "error", err)
		}
		return nil, code, err
	}
	if _, matched, parseErr := parseProModeRequest(prepared.message, defaultProMaxIterations); matched || parseErr != nil {
		if parseErr != nil {
			logAction(traceID, busActionAgentSend, "error", parseErr)
			return nil, http.StatusBadRequest, parseErr
		}
		logAction(traceID, busActionAgentSend, "running", nil)
		payload, code, err := s.executeProModeAction(ctx, prepared, traceID)
		if err != nil {
			logAction(traceID, busActionAgentSend, "error", err)
			return nil, code, err
		}
		s.publishAssistantSessionPush(traceID, finalizedAgentTurn{
			message:   payload.Message,
			sessionID: payload.SessionID,
		})
		logAction(traceID, busActionAgentSend, "success", nil)
		return payload, code, nil
	}

	logAction(traceID, busActionAgentSend, "running", nil)
	response, sessionID, err := s.runPreparedAgentTurn(ctx, prepared, traceID)
	if err != nil {
		awaitingErr, statusCode, _, normalizedErr := classifyAgentTurnError(err)
		if awaitingErr != nil {
			logAction(traceID, busActionAgentSend, "awaiting_human", nil)
			s.publishAwaitingHumanSessionPush(traceID, sessionID, awaitingErr)
			return newAwaitingHumanResponse(sessionID, awaitingErr), http.StatusAccepted, nil
		}
		logAction(traceID, busActionAgentSend, "error", normalizedErr)
		return nil, statusCode, normalizedErr
	}
	result, code, err := s.finalizeAgentTurn(response, sessionID)
	if err != nil {
		logAction(traceID, busActionAgentSend, "error", err)
		return nil, code, err
	}

	payload, payloadErr := newAgentResponsePayload(result.message, result.sessionID, result.sessionEnd, agentResponseMeta{})
	if payloadErr != nil {
		logAction(traceID, busActionAgentSend, "error", payloadErr)
		return nil, http.StatusInternalServerError, payloadErr
	}
	s.publishAssistantSessionPush(traceID, result)
	logAction(traceID, busActionAgentSend, "success", nil)
	return payload, http.StatusOK, nil
}

func (s *bridgeService) executeAgentStreamAction(ctx context.Context, params agentParams, traceID string, sink streaming.Sink) (string, string, error) {
	trackedSink := newEventTurnTracker(newSessionStreamBroadcastSink(sink, s.sessionPush))
	sessionID := strings.TrimSpace(params.SessionID)

	prepared, err := s.validateAgentStreamRequest(ctx, params, traceID, sessionID, trackedSink)
	if err != nil {
		return "", "", err
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
	prepared, code, err := s.validateAgentTurnRequest(params)
	if err == nil {
		return prepared, nil
	}
	if !errors.Is(err, errAgentMessageRequired) {
		logAction(traceID, busActionAgentSend, "error", err)
	}
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
	_, matched, parseErr := parseProModeRequest(prepared.message, defaultProMaxIterations)
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
		statusCode, normalizedErr := normalizeAgentExecutionError(err)
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

	result, code, err := s.finalizeAgentTurn(response, sessionID)
	if err != nil {
		logAction(traceID, busActionAgentSend, "error", err)
		stepID, stepErr := streaming.AssistantStepID(trackedSink.finalAssistantTurn())
		if stepErr != nil {
			return "", "", stepErr
		}
		if emitErr := emitStreamErrorEvent(ctx, trackedSink, traceID, trackedSink.finalAssistantTurn(), stepID, sessionID, code, err); emitErr != nil {
			return "", "", emitErr
		}
		return "", "", err
	}

	s.publishAssistantSessionPush(traceID, result)
	logAction(traceID, busActionAgentSend, "success", nil)
	return result.message, result.sessionID, nil
}

func (s *bridgeService) executeAgentStopAction(_ context.Context, params agentStopParams, traceID string) (any, int, error) {
	sessionID := strings.TrimSpace(params.SessionID)
	stopTraceID := strings.TrimSpace(params.TraceID)
	if sessionID == "" && stopTraceID == "" {
		return nil, http.StatusBadRequest, errors.New("session_id or trace_id is required")
	}
	if s.runRegistry == nil {
		return nil, http.StatusServiceUnavailable, errors.New("run registry is not available")
	}

	var err error
	if sessionID != "" {
		err = s.runRegistry.CancelBySessionID(sessionID)
	} else {
		err = s.runRegistry.CancelByTraceID(stopTraceID)
	}
	if err != nil {
		if errors.Is(err, ErrRunNotFound) {
			logAction(traceID, busActionAgentStop, "not_running", nil)
			return agentStopResponse{
				Status:  "not_running",
				Message: "no active run found",
			}, http.StatusOK, nil
		}
		logAction(traceID, busActionAgentStop, "error", err)
		return nil, http.StatusInternalServerError, err
	}

	logAction(traceID, busActionAgentStop, "success", nil)
	return agentStopResponse{
		Status:  "stopped",
		Message: "agent run cancelled successfully",
	}, http.StatusOK, nil
}
