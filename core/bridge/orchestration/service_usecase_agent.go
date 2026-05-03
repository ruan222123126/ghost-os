// Agent turn use cases exposed to the transport layer.

package orchestration

import (
	"context"
	"errors"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
)

// executeAgentAction 执行一次 Agent 回合，并处理“等待人工回答”的中断状态。
func (s *bridgeService) executeAgentAction(ctx context.Context, params agentParams, traceID string) (ServiceResult, error) {
	return s.executeAgentActionWithRuntimeOverrides(ctx, params, nil, traceID)
}

func (s *bridgeService) executeAgentActionWithRuntimeOverrides(
	ctx context.Context,
	params agentParams,
	runtimeOverrides *TaskRuntimeOverrides,
	traceID string,
) (ServiceResult, error) {
	prepared, err := s.prepareAgentTurnWithRuntimeOverrides(params, runtimeOverrides, traceID)
	if err != nil {
		return ServiceResult{}, err
	}
	if result, handled, err := s.executeSpecialAgentMode(ctx, prepared, traceID); handled {
		return result, err
	}
	return s.executeStandardAgentTurnWithPrepared(ctx, prepared, traceID)
}

func (s *bridgeService) prepareAgentTurnWithRuntimeOverrides(
	params agentParams,
	runtimeOverrides *TaskRuntimeOverrides,
	traceID string,
) (preparedAgentTurnRequest, error) {
	prepared, err := s.validateAgentTurnRequest(params)
	if err != nil {
		if !errors.Is(err, errAgentMessageRequired) {
			logAction(traceID, busActionAgentSend, "error", err)
		}
		return preparedAgentTurnRequest{}, err
	}
	prepared.runtimeOverrides = cloneTaskRuntimeOverrides(runtimeOverrides)
	return prepared, nil
}

func (s *bridgeService) executeSpecialAgentMode(
	ctx context.Context,
	prepared preparedAgentTurnRequest,
	traceID string,
) (ServiceResult, bool, error) {
	if prepared.mode == agentModePlan {
		result, err := s.executePlanModeWithPrepared(ctx, prepared, traceID)
		return result, true, err
	}

	_, matched, parseErr := parseProModeRequest(prepared.message, bridgeconfig.DefaultProMaxIterations)
	if !matched && parseErr == nil {
		return ServiceResult{}, false, nil
	}
	if parseErr != nil {
		logAction(traceID, busActionAgentSend, "error", parseErr)
		return ServiceResult{}, true, wrapServiceError(ServiceErrorInvalidInput, parseErr)
	}

	result, err := s.executeProModeWithPrepared(ctx, prepared, traceID)
	return result, true, err
}

func (s *bridgeService) executePlanModeWithPrepared(
	ctx context.Context,
	prepared preparedAgentTurnRequest,
	traceID string,
) (ServiceResult, error) {
	if prepared.runtimeOverrides != nil {
		err := errors.New("runtime_overrides are not supported in plan mode")
		logAction(traceID, busActionAgentSend, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInvalidInput, err)
	}
	logAction(traceID, busActionAgentSend, "running", nil)

	payload, code, err := s.executePlanModeAction(ctx, prepared, traceID)
	if err != nil {
		logAction(traceID, busActionAgentSend, "error", err)
		return ServiceResult{}, wrapServiceError(serviceErrorKindFromLegacyStatus(code), err)
	}

	s.publishSpecialModeAgentTurn(traceID, payload.Message, payload.SessionID)
	return serviceResultFromLegacySuccess(payload, code), nil
}

func (s *bridgeService) executeProModeWithPrepared(
	ctx context.Context,
	prepared preparedAgentTurnRequest,
	traceID string,
) (ServiceResult, error) {
	if prepared.runtimeOverrides != nil {
		err := errors.New("runtime_overrides are not supported in pro mode")
		logAction(traceID, busActionAgentSend, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInvalidInput, err)
	}
	logAction(traceID, busActionAgentSend, "running", nil)

	payload, code, err := s.executeProModeAction(ctx, prepared, traceID)
	if err != nil {
		logAction(traceID, busActionAgentSend, "error", err)
		return ServiceResult{}, wrapServiceError(serviceErrorKindFromLegacyStatus(code), err)
	}

	s.publishSpecialModeAgentTurn(traceID, payload.Message, payload.SessionID)
	return serviceResultFromLegacySuccess(payload, code), nil
}

func (s *bridgeService) publishSpecialModeAgentTurn(traceID string, message string, sessionID string) {
	s.publishAssistantSessionPush(traceID, finalizedAgentTurn{
		message:   message,
		sessionID: sessionID,
	})
	logAction(traceID, busActionAgentSend, "success", nil)
}

func (s *bridgeService) executeStandardAgentTurnWithPrepared(
	ctx context.Context,
	prepared preparedAgentTurnRequest,
	traceID string,
) (ServiceResult, error) {
	logAction(traceID, busActionAgentSend, "running", nil)
	response, sessionID, err := s.runPreparedAgentTurn(ctx, prepared, traceID)
	if err != nil {
		awaitingErr, kind, _, normalizedErr := classifyAgentTurnError(err)
		if awaitingErr != nil {
			logAction(traceID, busActionAgentSend, "awaiting_human", nil)
			s.publishAwaitingHumanSessionPush(traceID, sessionID, awaitingErr)
			return serviceResultAccepted(newAwaitingHumanResponse(sessionID, awaitingErr)), nil
		}
		logAction(traceID, busActionAgentSend, "error", normalizedErr)
		return ServiceResult{}, wrapServiceError(kind, normalizedErr)
	}

	result, err := s.finalizeAgentTurn(response, sessionID)
	if err != nil {
		logAction(traceID, busActionAgentSend, "error", err)
		return ServiceResult{}, err
	}

	payload, payloadErr := newAgentResponsePayload(result.message, result.sessionID, result.sessionEnd, agentResponseMeta{})
	if payloadErr != nil {
		logAction(traceID, busActionAgentSend, "error", payloadErr)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, payloadErr)
	}
	s.publishAssistantSessionPush(traceID, result)
	logAction(traceID, busActionAgentSend, "success", nil)
	return serviceResultSuccess(payload), nil
}

func (s *bridgeService) executeAgentStopAction(ctx context.Context, params agentStopParams, traceID string) (ServiceResult, error) {
	sessionID := strings.TrimSpace(params.SessionID)
	stopTraceID := strings.TrimSpace(params.TraceID)
	if sessionID == "" && stopTraceID == "" {
		return ServiceResult{}, wrapServiceError(ServiceErrorInvalidInput, errors.New("session_id or trace_id is required"))
	}
	if s.runRegistry == nil {
		return ServiceResult{}, wrapServiceError(ServiceErrorUnavailable, errors.New("run registry is not available"))
	}

	var (
		err    error
		handle *RunHandle
	)
	if sessionID != "" {
		handle, err = s.runRegistry.CancelAndWaitBySessionID(ctx, sessionID)
	} else {
		handle, err = s.runRegistry.CancelAndWaitByTraceID(ctx, stopTraceID)
	}
	if err != nil {
		if errors.Is(err, ErrRunNotFound) {
			logAction(traceID, busActionAgentStop, "not_running", nil)
			return serviceResultSuccess(agentStopResponse{
				Status:  "not_running",
				Message: "no active run found",
			}), nil
		}
		logAction(traceID, busActionAgentStop, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}

	logAction(traceID, busActionAgentStop, "success", nil)
	return serviceResultSuccess(agentStopResponse{
		Status:    "stopped",
		Message:   "agent run cancelled successfully",
		SessionID: resolveStoppedSessionID(sessionID, handle),
	}), nil
}

func resolveStoppedSessionID(sessionID string, handle *RunHandle) string {
	if trimmedSessionID := strings.TrimSpace(sessionID); trimmedSessionID != "" {
		return trimmedSessionID
	}
	if handle == nil {
		return ""
	}
	return strings.TrimSpace(handle.SessionID)
}
