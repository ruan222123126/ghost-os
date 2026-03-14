package orchestration

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
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
	if code, inflightErr := s.ensureSessionNotInflight(prepared.sessionID); inflightErr != nil {
		return preparedAgentTurnRequest{}, code, inflightErr
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
	if len(awaitingErr.Options) == 0 {
		return response
	}

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

// emitDirectAgentStreamResult 只用于未经过 agent.RunStream() 的流式完成路径，例如人工取消后直接结束会话。
func emitDirectAgentStreamResult(ctx context.Context, sink streaming.Sink, traceID string, turn int, result finalizedAgentTurn) error {
	stepID, err := streaming.AssistantStepID(turn)
	if err != nil {
		return err
	}
	messageEvent, err := streaming.NewEvent(traceID, result.sessionID, turn, stepID, streaming.EventMessage, map[string]any{
		"text":       result.message,
		"session_id": result.sessionID,
	})
	if err != nil {
		return err
	}
	if err := emitStreamEvent(ctx, sink, messageEvent); err != nil {
		return err
	}
	doneEvent, err := streaming.NewEvent(traceID, result.sessionID, turn, "", streaming.EventDone, map[string]any{
		"session_id":    result.sessionID,
		"session_ended": result.sessionEnd != nil,
	})
	if err != nil {
		return err
	}
	return emitStreamEvent(ctx, sink, doneEvent)
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
