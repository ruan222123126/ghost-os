// Agent turn use cases exposed to the transport layer.

package orchestration

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

// executeAgentAction 执行一次 Agent 回合，并处理“等待人工回答”的中断状态。
func (s *bridgeService) executeAgentAction(ctx context.Context, params agentParams, traceID string) (any, int, error) {
	return s.executeAgentActionWithRuntimeOverrides(ctx, params, nil, traceID)
}

func (s *bridgeService) executeAgentActionWithRuntimeOverrides(
	ctx context.Context,
	params agentParams,
	runtimeOverrides *TaskRuntimeOverrides,
	traceID string,
) (any, int, error) {
	prepared, code, err := s.validateAgentTurnRequest(params)
	if err != nil {
		if !errors.Is(err, errAgentMessageRequired) {
			logAction(traceID, busActionAgentSend, "error", err)
		}
		return nil, code, err
	}
	prepared.runtimeOverrides = cloneTaskRuntimeOverrides(runtimeOverrides)
	if prepared.mode == agentModePlan {
		if prepared.runtimeOverrides != nil {
			err := errors.New("runtime_overrides are not supported in plan mode")
			logAction(traceID, busActionAgentSend, "error", err)
			return nil, http.StatusBadRequest, err
		}
		logAction(traceID, busActionAgentSend, "running", nil)
		payload, code, err := s.executePlanModeAction(ctx, prepared, traceID)
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
	if _, matched, parseErr := parseProModeRequest(prepared.message, defaultProMaxIterations); matched || parseErr != nil {
		if parseErr != nil {
			logAction(traceID, busActionAgentSend, "error", parseErr)
			return nil, http.StatusBadRequest, parseErr
		}
		if prepared.runtimeOverrides != nil {
			err := errors.New("runtime_overrides are not supported in pro mode")
			logAction(traceID, busActionAgentSend, "error", err)
			return nil, http.StatusBadRequest, err
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
