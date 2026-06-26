package orchestration

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	agentturnadapter "ghost-os/bridge/orchestration/internal/adapters/agentturnservice"
	"ghost-os/bridge/orchestration/internal/app/agentturn"
	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
	internaltrace "ghost-os/bridge/orchestration/internal/trace"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
	bridgeTasks "ghost-os/bridge/tasks"
)

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

func (s *bridgeService) publishAssistantSessionPush(traceID string, result finalizedAgentTurn) {
	if s == nil {
		return
	}
	internaltrace.PublishAssistantMessage(
		s.sessionPushHub(),
		traceID,
		result.sessionID,
		api.AssistantMessagePushPayload{
			Message:      result.message,
			SessionEnded: result.sessionEnd != nil,
			SessionEnd:   result.sessionEnd,
		},
	)
}

func (s *bridgeService) publishAwaitingHumanSessionPush(
	traceID string,
	sessionID string,
	awaitingErr *agent.ErrAwaitingHuman,
) {
	if s == nil {
		return
	}
	internaltrace.PublishAwaitingHuman(s.sessionPushHub(), traceID, sessionID, awaitingErr)
}

func (s *bridgeService) pendingQuestionSnapshot(sessionID string) (sessionPushEvent, bool) {
	if s == nil {
		return sessionPushEvent{}, false
	}
	return internaltrace.PendingQuestionSnapshot(s.sessionStore, sessionID)
}

func (s *bridgeService) ensureSessionNotInflight(sessionID string) error {
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return nil
	}
	if s == nil || s.runRegistry == nil {
		return bus.WrapError(ServiceErrorInternal, errors.New("run registry is not configured"))
	}
	if !s.runRegistry.IsInflight(id) {
		return nil
	}
	return bus.WrapError(ServiceErrorConflict, fmt.Errorf("%w: session_id=%s", ErrSessionInflight, id))
}

func prepareAgentTurnRequest(params agentParams) (preparedAgentTurnRequest, error) {
	return agentturn.PrepareRequest(params)
}

func classifyAgentTurnError(err error) (*agent.ErrAwaitingHuman, ServiceErrorKind, bool, error) {
	var awaitingErr *agent.ErrAwaitingHuman
	if errors.As(err, &awaitingErr) {
		return awaitingErr, "", false, nil
	}
	kind, normalizedErr := normalizeAgentExecutionError(err)
	return nil, kind, errors.Is(normalizedErr, ErrRunCancelled), normalizedErr
}

func (s *bridgeService) agentTurnService() agentturn.Service {
	return agentturnadapter.New(agentturnadapter.Config{
		EnsureSessionNotInflight: s.ensureSessionNotInflight,
		EnsureSessionActive:      s.ensureSessionActive,
		RunTurn:                  s.runPreparedAgentTurn,
		RunTurnStream:            s.runPreparedAgentTurnStream,
		Plan: agentturnadapter.PlanConfig{
			RuntimeBuilder: s.agentTurnRuntimeBuilder(),
			ConfigStore:    s.configStore,
			SessionStore:   s.sessionStore,
			RunRegistry:    s.runRegistry,
		},
		Finalize: func(response string, sessionID string) (agentturn.FinalizedTurn, error) {
			result, err := s.finalizeAgentTurn(response, sessionID)
			if err != nil {
				return agentturn.FinalizedTurn{}, err
			}
			return finalizedTurnToApp(result), nil
		},
		NewResponsePayload: func(
			turn agentturn.FinalizedTurn,
			meta agentturn.ResponseMeta,
		) (api.AgentResponse, error) {
			return newAgentResponsePayload(
				turn.Message,
				turn.SessionID,
				turn.SessionEnd,
				agentResponseMeta{Mode: meta.Mode},
			)
		},
		PublishAssistant: func(traceID string, turn agentturn.FinalizedTurn) {
			s.publishAssistantSessionPush(traceID, finalizedTurnFromApp(turn))
		},
		PublishAwaitingHuman: s.publishAwaitingHumanSessionPush,
		Classify:             classifyAgentTurnError,
		Log:                  logAction,
		Stop:                 s.agentTurnStopConfig(),
	})
}

func (s *bridgeService) agentTurnRuntimeBuilder() agentturnadapter.RuntimeBuilder {
	factory := AgentRuntimeFactory(nil)
	if s != nil {
		factory = s.runtimeFactory
	}
	if factory == nil {
		factory = newAgentRuntimeFactory()
	}
	return func(store bridgeconfig.Store) (agentturnadapter.RuntimeDependencies, error) {
		deps, err := factory.Build(store)
		if err != nil {
			return nil, err
		}
		return deps, nil
	}
}

func (s *bridgeService) agentTurnStopConfig() agentturnadapter.StopConfig {
	if s == nil || s.runRegistry == nil {
		return agentturnadapter.StopConfig{}
	}
	return agentturnadapter.StopConfig{
		CancelAndWaitBySessionID: func(ctx context.Context, sessionID string) (agentturn.StopHandle, error) {
			handle, err := s.runRegistry.CancelAndWaitBySessionID(ctx, sessionID)
			return stopHandleFromRun(handle), mapAgentTurnStopError(err)
		},
		CancelAndWaitByTraceID: func(ctx context.Context, traceID string) (agentturn.StopHandle, error) {
			handle, err := s.runRegistry.CancelAndWaitByTraceID(ctx, traceID)
			return stopHandleFromRun(handle), mapAgentTurnStopError(err)
		},
	}
}

func mapAgentTurnStopError(err error) error {
	if errors.Is(err, ErrRunNotFound) {
		return agentturn.ErrRunNotFound
	}
	return err
}

func stopHandleFromRun(handle *RunHandle) agentturn.StopHandle {
	if handle == nil {
		return agentturn.StopHandle{}
	}
	return agentturn.StopHandle{SessionID: handle.SessionID}
}

func (s *bridgeService) finalizeAgentTurn(response string, sessionID string) (finalizedAgentTurn, error) {
	normalizedMessage, sessionEndSignal, err := parseSessionEndSignal(response)
	if err != nil {
		return finalizedAgentTurn{}, bus.WrapError(ServiceErrorInternal, err)
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

func finalizedTurnToApp(result finalizedAgentTurn) agentturn.FinalizedTurn {
	return agentturn.FinalizedTurn{
		Message:    result.message,
		SessionID:  result.sessionID,
		SessionEnd: result.sessionEnd,
	}
}

func finalizedTurnFromApp(result agentturn.FinalizedTurn) finalizedAgentTurn {
	return finalizedAgentTurn{
		message:    result.Message,
		sessionID:  result.SessionID,
		sessionEnd: result.SessionEnd,
	}
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
	if err := internaltrace.EmitStreamEvent(ctx, sink, messageEvent); err != nil {
		return err
	}
	doneEvent, err := streaming.NewEvent(traceID, result.sessionID, turn, "", streaming.EventDone, map[string]any{
		"session_id":    result.sessionID,
		"session_ended": result.sessionEnd != nil,
	})
	if err != nil {
		return err
	}
	return internaltrace.EmitStreamEvent(ctx, sink, doneEvent)
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
	return s.executeAgentActionWithRuntimeOverrides(ctx, params, params.RuntimeOverrides, traceID)
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
	broadcastSink := internaltrace.NewSessionStreamBroadcastSink(sink, s.sessionPushHub())
	return s.agentTurnService().ExecuteStream(ctx, params, traceID, broadcastSink)
}
