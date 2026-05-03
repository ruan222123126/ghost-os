package orchestration

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

var errAgentMessageRequired = errors.New("message or images is required")

const (
	agentModeDefault = ""
	agentModePlan    = "plan"
)

type preparedAgentTurnRequest struct {
	userInput        llm.Message
	message          string
	mode             string
	sessionID        string
	requestRuntime   *requestRuntimeOptions
	runtimeOverrides *TaskRuntimeOverrides
}

type finalizedAgentTurn struct {
	message    string
	sessionID  string
	sessionEnd *assistantSessionEndSignalPayload
}

func prepareAgentTurnRequest(params agentParams) (preparedAgentTurnRequest, error) {
	userInput, message, err := buildAgentUserInput(params.Message, params.Images)
	if err != nil {
		return preparedAgentTurnRequest{}, wrapServiceError(ServiceErrorInvalidInput, err)
	}
	mode, err := normalizeAgentMode(params.Mode)
	if err != nil {
		return preparedAgentTurnRequest{}, wrapServiceError(ServiceErrorInvalidInput, err)
	}
	requestRuntime, err := normalizeRequestRuntimeOptions(params.ProjectRoot)
	if err != nil {
		return preparedAgentTurnRequest{}, wrapServiceError(ServiceErrorInvalidInput, err)
	}
	return preparedAgentTurnRequest{
		userInput:      userInput,
		message:        message,
		mode:           mode,
		sessionID:      strings.TrimSpace(params.SessionID),
		requestRuntime: requestRuntime,
	}, nil
}

func normalizeAgentMode(raw string) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(raw))
	if mode == "" {
		return agentModeDefault, nil
	}
	if mode == agentModePlan {
		return mode, nil
	}
	return "", fmt.Errorf("unsupported agent mode: %q", mode)
}

func (s *bridgeService) validateAgentTurnRequest(params agentParams) (preparedAgentTurnRequest, error) {
	prepared, err := prepareAgentTurnRequest(params)
	if err != nil {
		return preparedAgentTurnRequest{}, err
	}
	if inflightErr := s.ensureSessionNotInflight(prepared.sessionID); inflightErr != nil {
		return preparedAgentTurnRequest{}, inflightErr
	}
	if activeErr := s.ensureSessionActive(prepared.sessionID); activeErr != nil {
		return preparedAgentTurnRequest{}, activeErr
	}
	return prepared, nil
}

func classifyAgentTurnError(err error) (*agent.ErrAwaitingHuman, ServiceErrorKind, bool, error) {
	var awaitingErr *agent.ErrAwaitingHuman
	if errors.As(err, &awaitingErr) {
		return awaitingErr, "", false, nil
	}
	kind, normalizedErr := normalizeAgentExecutionError(err)
	return nil, kind, errors.Is(normalizedErr, ErrRunCancelled), normalizedErr
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

func (s *bridgeService) finalizeAgentTurn(response string, sessionID string) (finalizedAgentTurn, error) {
	normalizedMessage, sessionEndSignal, err := parseSessionEndSignal(response)
	if err != nil {
		return finalizedAgentTurn{}, wrapServiceError(ServiceErrorInternal, err)
	}
	if sessionEndSignal != nil {
		if markErr := s.markSessionEnded(sessionID); markErr != nil {
			return finalizedAgentTurn{}, markErr
		}
	}
	return finalizedAgentTurn{
		message:    normalizedMessage,
		sessionID:  strings.TrimSpace(sessionID),
		sessionEnd: sessionEndSignal,
	}, nil
}

// emitDirectAgentStreamResult 只用于未经过 agent.RunMessageStreamWithTraceID() 的流式完成路径，例如人工取消后直接结束会话。
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

func normalizeAgentExecutionError(err error) (ServiceErrorKind, error) {
	switch {
	case errors.Is(err, session.ErrInvalidSessionID), errors.Is(err, errSessionEnded):
		return ServiceErrorInvalidInput, err
	case errors.Is(err, session.ErrSessionNotFound):
		return ServiceErrorNotFound, err
	case errors.Is(err, ErrSessionInflight):
		return ServiceErrorConflict, err
	case errors.Is(err, context.Canceled):
		return ServiceErrorConflict, ErrRunCancelled
	default:
		return ServiceErrorInternal, err
	}
}
