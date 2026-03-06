// Agent/config use cases: query and mutation entrypoints exposed to transport layer.

package app

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/session"
)

// executeAgentAction 执行一次 Agent 回合，并处理“等待人工回答”的中断状态。
func (s *bridgeService) executeAgentAction(ctx context.Context, params agentParams, traceID string) (any, int, error) {
	trimmed := strings.TrimSpace(params.Message)
	trimmedSessionID := strings.TrimSpace(params.SessionID)
	if trimmed == "" && trimmedSessionID == "" {
		return nil, http.StatusBadRequest, errors.New("message is required")
	}
	if code, activeErr := s.ensureSessionActive(trimmedSessionID); activeErr != nil {
		logAction(traceID, actionAgentSend, "error", activeErr)
		return nil, code, activeErr
	}

	logAction(traceID, actionAgentSend, "running", nil)
	response, sessionID, err := s.agentExecutor(ctx, trimmed, trimmedSessionID, traceID, s.configStore, s.sessionStore)
	if err != nil {
		var awaitingErr *agent.ErrAwaitingHuman
		if errors.As(err, &awaitingErr) {
			logAction(traceID, actionAgentSend, "awaiting_human", nil)
			return askHumanAwaitingResponse{
				Status:     "awaiting_human",
				SessionID:  sessionID,
				QuestionID: awaitingErr.QuestionID,
				Prompt:     awaitingErr.Prompt,
			}, http.StatusAccepted, nil
		}

		normalizedErr, statusCode := normalizeAgentExecutionError(err)
		logAction(traceID, actionAgentSend, "error", normalizedErr)
		return nil, statusCode, normalizedErr
	}
	normalizedMessage, sessionEndSignal, parseErr := parseSessionEndSignal(response)
	if parseErr != nil {
		logAction(traceID, actionAgentSend, "error", parseErr)
		return nil, http.StatusInternalServerError, parseErr
	}

	if sessionEndSignal != nil {
		if code, markErr := s.markSessionEnded(sessionID); markErr != nil {
			logAction(traceID, actionAgentSend, "error", markErr)
			return nil, code, markErr
		}
	}

	payload, payloadErr := newAgentResponsePayload(normalizedMessage, sessionID, sessionEndSignal)
	if payloadErr != nil {
		logAction(traceID, actionAgentSend, "error", payloadErr)
		return nil, http.StatusInternalServerError, payloadErr
	}
	logAction(traceID, actionAgentSend, "success", nil)
	return payload, http.StatusOK, nil
}

func (s *bridgeService) executeAgentStreamAction(ctx context.Context, params agentParams, traceID string, sink agent.EventSink) (string, string, error) {
	trimmed := strings.TrimSpace(params.Message)
	trimmedSessionID := strings.TrimSpace(params.SessionID)
	trackedSink := newEventTurnTracker(sink)

	if trimmed == "" && trimmedSessionID == "" {
		err := errors.New("message is required")
		if emitErr := emitStreamErrorEvent(ctx, trackedSink, traceID, 0, "", trimmedSessionID, http.StatusBadRequest, err); emitErr != nil {
			return "", "", emitErr
		}
		return "", "", err
	}
	if code, activeErr := s.ensureSessionActive(trimmedSessionID); activeErr != nil {
		logAction(traceID, actionAgentSend, "error", activeErr)
		if emitErr := emitStreamErrorEvent(ctx, trackedSink, traceID, 0, "", trimmedSessionID, code, activeErr); emitErr != nil {
			return "", "", emitErr
		}
		return "", "", activeErr
	}

	logAction(traceID, actionAgentSend, "running", nil)
	response, sessionID, err := s.agentExecutorStream(ctx, trimmed, trimmedSessionID, traceID, s.configStore, s.sessionStore, trackedSink)
	if err != nil {
		var awaitingErr *agent.ErrAwaitingHuman
		if errors.As(err, &awaitingErr) {
			logAction(traceID, actionAgentSend, "awaiting_human", nil)
			return "", sessionID, err
		}

		normalizedErr, statusCode := normalizeAgentExecutionError(err)
		if errors.Is(normalizedErr, ErrRunCancelled) {
			logAction(traceID, actionAgentSend, "cancelled", normalizedErr)
			return "", sessionID, normalizedErr
		}
		logAction(traceID, actionAgentSend, "error", normalizedErr)
		if emitErr := emitStreamErrorEvent(ctx, trackedSink, traceID, trackedSink.finalAssistantTurn(), "", sessionID, statusCode, normalizedErr); emitErr != nil {
			return "", "", emitErr
		}
		return "", sessionID, normalizedErr
	}

	normalizedMessage, sessionEndSignal, parseErr := parseSessionEndSignal(response)
	finalTurn := trackedSink.finalAssistantTurn()
	if parseErr != nil {
		logAction(traceID, actionAgentSend, "error", parseErr)
		if emitErr := emitStreamErrorEvent(ctx, trackedSink, traceID, finalTurn, agent.AssistantStepID(finalTurn), sessionID, http.StatusInternalServerError, parseErr); emitErr != nil {
			return "", "", emitErr
		}
		return "", "", parseErr
	}

	if sessionEndSignal != nil {
		if code, markErr := s.markSessionEnded(sessionID); markErr != nil {
			logAction(traceID, actionAgentSend, "error", markErr)
			if emitErr := emitStreamErrorEvent(ctx, trackedSink, traceID, finalTurn, agent.AssistantStepID(finalTurn), sessionID, code, markErr); emitErr != nil {
				return "", "", emitErr
			}
			return "", "", markErr
		}
	}

	if emitErr := emitStreamEvent(ctx, trackedSink, agent.NewEvent(traceID, finalTurn, agent.AssistantStepID(finalTurn), agent.EventMessage, map[string]any{
		"text":       normalizedMessage,
		"session_id": strings.TrimSpace(sessionID),
	})); emitErr != nil {
		logAction(traceID, actionAgentSend, "error", emitErr)
		return "", "", emitErr
	}
	if emitErr := emitStreamEvent(ctx, trackedSink, agent.NewEvent(traceID, finalTurn, "", agent.EventDone, map[string]any{
		"session_id":    strings.TrimSpace(sessionID),
		"session_ended": sessionEndSignal != nil,
	})); emitErr != nil {
		logAction(traceID, actionAgentSend, "error", emitErr)
		return "", "", emitErr
	}

	logAction(traceID, actionAgentSend, "success", nil)
	return normalizedMessage, sessionID, nil
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
			logAction(traceID, actionAgentStop, "not_running", nil)
			return agentStopResponse{
				Status:  "not_running",
				Message: "no active run found",
			}, http.StatusOK, nil
		}
		logAction(traceID, actionAgentStop, "error", err)
		return nil, http.StatusInternalServerError, err
	}

	logAction(traceID, actionAgentStop, "success", nil)
	return agentStopResponse{
		Status:  "stopped",
		Message: "agent run cancelled successfully",
	}, http.StatusOK, nil
}

func normalizeAgentExecutionError(err error) (error, int) {
	switch {
	case errors.Is(err, session.ErrInvalidSessionID):
		return err, http.StatusBadRequest
	case errors.Is(err, ErrSessionInflight):
		return err, http.StatusConflict
	case errors.Is(err, context.Canceled):
		return ErrRunCancelled, http.StatusConflict
	default:
		return err, http.StatusInternalServerError
	}
}

// executeConfigGetAction 返回当前运行态配置快照，不暴露敏感明文字段。
func (s *bridgeService) executeConfigGetAction(traceID string) (any, int, error) {
	logAction(traceID, actionConfigGet, "success", nil)
	return s.configStore.Snapshot(), http.StatusOK, nil
}

// executeConfigUpdateAction 按请求局部更新运行态配置，并返回更新后快照。
func (s *bridgeService) executeConfigUpdateAction(req configUpdateRequest, traceID string) (any, int, error) {
	logAction(traceID, actionConfigUpdate, "running", nil)
	if err := s.configStore.Update(req); err != nil {
		logAction(traceID, actionConfigUpdate, "error", err)
		return nil, http.StatusBadRequest, err
	}
	logAction(traceID, actionConfigUpdate, "success", nil)
	return s.configStore.Snapshot(), http.StatusOK, nil
}

// executeHumanResponseAction 接收 HUMAN_RESPONSE，将答案写回会话并解除 pending 状态。
func (s *bridgeService) executeHumanResponseAction(_ context.Context, params humanResponseParams, traceID string) (any, int, error) {
	store, code, err := s.requireSessionStore()
	if err != nil {
		return nil, code, err
	}

	sessionID, code, err := requireSessionID(params.SessionID)
	if err != nil {
		return nil, code, err
	}

	questionID := strings.TrimSpace(params.QuestionID)
	if questionID == "" {
		return nil, http.StatusBadRequest, errors.New("question_id is required")
	}

	answer := strings.TrimSpace(params.Answer)
	if answer == "" {
		return nil, http.StatusBadRequest, errors.New("answer is required")
	}

	logAction(traceID, actionHumanResponse, "running", nil)
	sess, err := store.Load(sessionID)
	if err != nil {
		statusCode := mapSessionStorageError(err)
		logAction(traceID, actionHumanResponse, "error", err)
		return nil, statusCode, err
	}

	if !sess.SetHumanAnswer(questionID, answer) {
		err = errors.New("question not found in pending questions")
		logAction(traceID, actionHumanResponse, "error", err)
		return nil, http.StatusNotFound, err
	}

	if err := store.Save(sess); err != nil {
		statusCode := mapSessionStorageError(err)
		logAction(traceID, actionHumanResponse, "error", err)
		return nil, statusCode, err
	}

	logAction(traceID, actionHumanResponse, "success", nil)
	return humanResponseAck{
		SessionID:  sessionID,
		QuestionID: questionID,
		Accepted:   true,
	}, http.StatusOK, nil
}
