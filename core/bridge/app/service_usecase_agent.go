// Agent turn use cases exposed to the transport layer.

package app

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/session"
)

var errAgentMessageRequired = errors.New("message is required")

type preparedAgentTurnRequest struct {
	message   string
	sessionID string
}

type finalizedAgentTurn struct {
	message    string
	sessionID  string
	sessionEnd *assistantSessionEndSignalPayload
}

func prepareAgentTurnRequest(params agentParams) (preparedAgentTurnRequest, int, error) {
	prepared := preparedAgentTurnRequest{
		message:   strings.TrimSpace(params.Message),
		sessionID: strings.TrimSpace(params.SessionID),
	}
	if prepared.message == "" {
		return preparedAgentTurnRequest{}, http.StatusBadRequest, errAgentMessageRequired
	}
	return prepared, http.StatusOK, nil
}

func (s *bridgeService) validateAgentTurnRequest(params agentParams) (preparedAgentTurnRequest, int, error) {
	prepared, code, err := prepareAgentTurnRequest(params)
	if err != nil {
		return preparedAgentTurnRequest{}, code, err
	}
	if code, activeErr := s.ensureSessionActive(prepared.sessionID); activeErr != nil {
		return preparedAgentTurnRequest{}, code, activeErr
	}
	return prepared, http.StatusOK, nil
}

func classifyAgentTurnError(err error) (*agent.ErrAwaitingHuman, error, int, bool) {
	var awaitingErr *agent.ErrAwaitingHuman
	if errors.As(err, &awaitingErr) {
		return awaitingErr, nil, http.StatusAccepted, false
	}

	normalizedErr, statusCode := normalizeAgentExecutionError(err)
	return nil, normalizedErr, statusCode, errors.Is(normalizedErr, ErrRunCancelled)
}

func newAwaitingHumanResponse(sessionID string, awaitingErr *agent.ErrAwaitingHuman) askHumanAwaitingResponse {
	response := askHumanAwaitingResponse{
		Status:        "awaiting_human",
		SessionID:     strings.TrimSpace(sessionID),
		QuestionID:    awaitingErr.QuestionID,
		Prompt:        awaitingErr.Prompt,
		SelectionMode: strings.TrimSpace(awaitingErr.SelectionMode),
	}
	if len(awaitingErr.Options) > 0 {
		response.Options = make([]askHumanOption, 0, len(awaitingErr.Options))
		for _, option := range awaitingErr.Options {
			label := strings.TrimSpace(option.Label)
			if label == "" {
				continue
			}
			response.Options = append(response.Options, askHumanOption{
				Label:       label,
				AllowCustom: option.AllowCustom,
			})
		}
	}
	return response
}

func (s *bridgeService) finalizeAgentTurn(response string, sessionID string) (finalizedAgentTurn, int, error) {
	normalizedMessage, sessionEndSignal, err := parseSessionEndSignal(response)
	if err != nil {
		return finalizedAgentTurn{}, http.StatusInternalServerError, err
	}
	if sessionEndSignal != nil {
		if code, markErr := s.markSessionEnded(sessionID); markErr != nil {
			return finalizedAgentTurn{}, code, markErr
		}
	}
	return finalizedAgentTurn{
		message:    normalizedMessage,
		sessionID:  strings.TrimSpace(sessionID),
		sessionEnd: sessionEndSignal,
	}, http.StatusOK, nil
}

func emitFinalAgentStreamEvents(ctx context.Context, sink agent.EventSink, traceID string, turn int, result finalizedAgentTurn) error {
	if emitErr := emitStreamEvent(ctx, sink, agent.NewEvent(traceID, turn, agent.AssistantStepID(turn), agent.EventMessage, map[string]any{
		"text":       result.message,
		"session_id": result.sessionID,
	})); emitErr != nil {
		return emitErr
	}
	return emitStreamEvent(ctx, sink, agent.NewEvent(traceID, turn, "", agent.EventDone, map[string]any{
		"session_id":    result.sessionID,
		"session_ended": result.sessionEnd != nil,
	}))
}

// executeAgentAction 执行一次 Agent 回合，并处理“等待人工回答”的中断状态。
func (s *bridgeService) executeAgentAction(ctx context.Context, params agentParams, traceID string) (any, int, error) {
	prepared, code, err := s.validateAgentTurnRequest(params)
	if err != nil {
		if !errors.Is(err, errAgentMessageRequired) {
			logAction(traceID, busActionAgentSend, "error", err)
		}
		return nil, code, err
	}

	logAction(traceID, busActionAgentSend, "running", nil)
	response, sessionID, err := s.agentRunner.RunTurn(ctx, prepared.message, prepared.sessionID, traceID)
	if err != nil {
		awaitingErr, normalizedErr, statusCode, _ := classifyAgentTurnError(err)
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

	payload, payloadErr := newAgentResponsePayload(result.message, result.sessionID, result.sessionEnd)
	if payloadErr != nil {
		logAction(traceID, busActionAgentSend, "error", payloadErr)
		return nil, http.StatusInternalServerError, payloadErr
	}
	s.publishAssistantSessionPush(traceID, result)
	logAction(traceID, busActionAgentSend, "success", nil)
	return payload, http.StatusOK, nil
}

func (s *bridgeService) executeAgentStreamAction(ctx context.Context, params agentParams, traceID string, sink agent.EventSink) (string, string, error) {
	trackedSink := newEventTurnTracker(newSessionStreamBroadcastSink(sink, s.sessionPush, params.SessionID))
	prepared, code, err := s.validateAgentTurnRequest(params)
	if err != nil {
		if !errors.Is(err, errAgentMessageRequired) {
			logAction(traceID, busActionAgentSend, "error", err)
		}
		if emitErr := emitStreamErrorEvent(ctx, trackedSink, traceID, 0, "", strings.TrimSpace(params.SessionID), code, err); emitErr != nil {
			return "", "", emitErr
		}
		return "", "", err
	}

	logAction(traceID, busActionAgentSend, "running", nil)
	response, sessionID, err := s.agentRunner.RunTurnStream(ctx, prepared.message, prepared.sessionID, traceID, trackedSink)
	if err != nil {
		awaitingErr, normalizedErr, statusCode, cancelled := classifyAgentTurnError(err)
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
		if emitErr := emitStreamErrorEvent(ctx, trackedSink, traceID, trackedSink.finalAssistantTurn(), "", sessionID, statusCode, normalizedErr); emitErr != nil {
			return "", "", emitErr
		}
		return "", sessionID, normalizedErr
	}

	finalTurn := trackedSink.finalAssistantTurn()
	result, code, err := s.finalizeAgentTurn(response, sessionID)
	if err != nil {
		logAction(traceID, busActionAgentSend, "error", err)
		if emitErr := emitStreamErrorEvent(ctx, trackedSink, traceID, finalTurn, agent.AssistantStepID(finalTurn), sessionID, code, err); emitErr != nil {
			return "", "", emitErr
		}
		return "", "", err
	}

	if emitErr := emitFinalAgentStreamEvents(ctx, trackedSink, traceID, finalTurn, result); emitErr != nil {
		logAction(traceID, busActionAgentSend, "error", emitErr)
		return "", "", emitErr
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
