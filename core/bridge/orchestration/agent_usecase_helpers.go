package orchestration

import (
	"context"
	"errors"
	"strings"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/orchestration/internal/app/agentturn"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
	bridgeTasks "ghost-os/bridge/tasks"
)

var errAgentMessageRequired = agentturn.ErrMessageRequired

const (
	agentModeDefault = agentturn.ModeDefault
	agentModePlan    = agentturn.ModePlan
)

type preparedAgentTurnRequest = agentturn.PreparedRequest

type finalizedAgentTurn struct {
	message    string
	sessionID  string
	sessionEnd *assistantSessionEndSignalPayload
}

func prepareAgentTurnRequest(params agentParams) (preparedAgentTurnRequest, error) {
	return agentturn.PrepareRequest(params)
}

func normalizeAgentMode(raw string) (string, error) {
	return agentturn.NormalizeMode(raw)
}

func (s *bridgeService) validateAgentTurnRequest(params agentParams) (preparedAgentTurnRequest, error) {
	return agentturn.PrepareWithRuntimeOverrides(s.agentTurnGuards(), params, nil)
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
	return s.agentTurnService().Execute(
		ctx,
		params,
		bridgeTasks.CloneTaskRuntimeOverrides(runtimeOverrides),
		traceID,
	)
}

func (s *bridgeService) executeAgentStopAction(
	ctx context.Context,
	params agentStopParams,
	traceID string,
) (ServiceResult, error) {
	return s.agentTurnService().Stop(ctx, params, traceID)
}

func (s *bridgeService) executeAgentStreamAction(
	ctx context.Context,
	params agentParams,
	traceID string,
	sink streaming.Sink,
) (string, string, error) {
	broadcastSink := newSessionStreamBroadcastSink(sink, s.sessionPushHub())
	return s.agentTurnService().ExecuteStream(ctx, params, traceID, broadcastSink)
}
