// Human-response use cases exposed to the transport layer.

package orchestration

import (
	"context"
	"errors"

	taskservice "ghost-os/bridge/orchestration/internal/adapters/taskservice"
	appexternal "ghost-os/bridge/orchestration/internal/app/externalagent"
	appsessions "ghost-os/bridge/orchestration/internal/app/sessions"
	apptasks "ghost-os/bridge/orchestration/internal/app/tasks"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
	internaltrace "ghost-os/bridge/orchestration/internal/trace"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

const cancelledHumanDialogueMessage = "Conversation cancelled by user."

func (s *bridgeService) requireTaskStore() (*TaskStore, int, error) {
	return s.taskActions().RequireStore()
}

func (s *bridgeService) requireTaskMutationRunner() (apptasks.Mutation, int, error) {
	return s.taskActions().RequireMutationRunner()
}

func legacyStatusFromServiceError(err error) int {
	return bus.StatusFromError(err)
}

func (s *bridgeService) taskActions() taskservice.Actions {
	return taskservice.New(taskservice.Config{
		Store:             s.taskStore(),
		Scheduler:         s.taskScheduler(),
		InitErr:           s.taskInitErr(),
		ConfigStore:       s.configStore,
		SessionStore:      s.sessionStore,
		SessionEndedError: errSessionEnded,
		Logger:            serviceActionLogger{},
	})
}

func (s *bridgeService) executeTaskCreateAction(params taskCreateParams, traceID string) (any, int, error) {
	return s.taskActions().Create(params, traceID)
}

func (s *bridgeService) executeTaskUpdateAction(params taskUpdateParams, traceID string) (any, int, error) {
	return s.taskActions().Update(params, traceID)
}

func (s *bridgeService) executeTaskListAction(scope string, traceID string) (any, int, error) {
	return s.taskActions().List(scope, traceID)
}

func (s *bridgeService) executeTaskGetAction(params taskIDParams, traceID string) (any, int, error) {
	return s.taskActions().Get(params, traceID)
}

func (s *bridgeService) executeTaskLogsAction(params taskLogsParams, traceID string) (any, int, error) {
	return s.taskActions().Logs(params, traceID)
}

func (s *bridgeService) executeTaskRunNowAction(params taskRunNowParams, traceID string) (any, int, error) {
	return s.taskActions().RunNow(params, traceID)
}

func (s *bridgeService) executeTaskStopAction(
	ctx context.Context,
	params taskStopParams,
	traceID string,
) (any, int, error) {
	return s.taskActions().Stop(ctx, params, traceID)
}

func (s *bridgeService) executeTaskDeleteAction(params taskIDParams, traceID string) (any, int, error) {
	return s.taskActions().Delete(params, traceID)
}

func (s *bridgeService) executeTaskCreateActionResult(params taskCreateParams, traceID string) (ServiceResult, error) {
	return bus.ResultFromStatus(s.executeTaskCreateAction(params, traceID))
}

func (s *bridgeService) executeTaskListActionResult(scope string, traceID string) (ServiceResult, error) {
	return bus.ResultFromStatus(s.executeTaskListAction(scope, traceID))
}

func (s *bridgeService) executeTaskGetActionResult(params taskIDParams, traceID string) (ServiceResult, error) {
	return bus.ResultFromStatus(s.executeTaskGetAction(params, traceID))
}

func (s *bridgeService) executeTaskUpdateActionResult(params taskUpdateParams, traceID string) (ServiceResult, error) {
	return bus.ResultFromStatus(s.executeTaskUpdateAction(params, traceID))
}

func (s *bridgeService) executeTaskRunNowActionResult(params taskRunNowParams, traceID string) (ServiceResult, error) {
	return bus.ResultFromStatus(s.executeTaskRunNowAction(params, traceID))
}

func (s *bridgeService) executeTaskStopActionResult(
	ctx context.Context,
	params taskStopParams,
	traceID string,
) (ServiceResult, error) {
	return bus.ResultFromStatus(s.executeTaskStopAction(ctx, params, traceID))
}

func (s *bridgeService) executeTaskLogsActionResult(params taskLogsParams, traceID string) (ServiceResult, error) {
	return bus.ResultFromStatus(s.executeTaskLogsAction(params, traceID))
}

func (s *bridgeService) executeTaskDeleteActionResult(params taskIDParams, traceID string) (ServiceResult, error) {
	return bus.ResultFromStatus(s.executeTaskDeleteAction(params, traceID))
}

// executeHumanResponseAction 接收 HUMAN_RESPONSE，将答案写回会话并解除 pending 状态。
func (s *bridgeService) executeHumanResponseAction(_ context.Context, params humanResponseParams, traceID string) (ServiceResult, error) {
	usecase, err := s.sessionUsecase()
	if err != nil {
		return ServiceResult{}, err
	}

	ack, err := usecase.AnswerHuman(params, traceID)
	if err != nil {
		return ServiceResult{}, bus.WrapError(mapHumanResponseErrorKind(err), err)
	}
	return bus.ResultSuccess(ack), nil
}

// executeHumanAnswerAndResumeAction 先写入人类答案，再继续执行被 ask_human 暂停的回合。
func (s *bridgeService) executeHumanAnswerAndResumeAction(ctx context.Context, params humanResponseParams, traceID string) (ServiceResult, error) {
	sessionID, err := requireSessionID(params.SessionID)
	if err != nil {
		return ServiceResult{}, err
	}

	if inflightErr := s.ensureSessionNotInflight(sessionID); inflightErr != nil {
		return ServiceResult{}, inflightErr
	}
	if activeErr := s.ensureSessionActive(sessionID); activeErr != nil {
		return ServiceResult{}, activeErr
	}

	params.SessionID = sessionID
	if _, err := s.executeHumanResponseAction(ctx, params, traceID); err != nil {
		return ServiceResult{}, err
	}
	if params.Cancelled {
		result := cancelledHumanTurn(sessionID)
		payload, err := newAgentResponsePayload(result.message, result.sessionID, result.sessionEnd, agentResponseMeta{})
		if err != nil {
			return ServiceResult{}, bus.WrapError(ServiceErrorInternal, err)
		}
		s.publishAssistantSessionPush(traceID, result)
		return bus.ResultSuccess(payload), nil
	}

	return s.agentTurnService().Resume(ctx, sessionID, traceID)
}

// executeHumanAnswerAndResumeStreamAction 先写入人类答案，再以 SSE 方式续跑被 ask_human 暂停的回合。
func (s *bridgeService) executeHumanAnswerAndResumeStreamAction(ctx context.Context, params humanResponseParams, traceID string, sink streaming.Sink) (string, string, error) {
	sessionID, err := s.resolveHumanResumeStreamSession(ctx, params.SessionID, traceID, sink)
	if err != nil {
		return "", sessionID, err
	}

	params.SessionID = sessionID
	if err := s.applyHumanResponseForResumeStream(ctx, params, traceID, sink); err != nil {
		return "", sessionID, err
	}
	if params.Cancelled {
		result := cancelledHumanTurn(sessionID)
		if emitErr := emitDirectAgentStreamResult(ctx, sink, traceID, 0, result); emitErr != nil {
			return "", "", emitErr
		}
		s.publishAssistantSessionPush(traceID, result)
		return result.message, result.sessionID, nil
	}

	broadcastSink := internaltrace.NewSessionStreamBroadcastSink(sink, s.sessionPushHub())
	return s.agentTurnService().ResumeStream(ctx, sessionID, traceID, broadcastSink)
}

func cancelledHumanTurn(sessionID string) finalizedAgentTurn {
	return finalizedAgentTurn{
		message:   cancelledHumanDialogueMessage,
		sessionID: sessionID,
		sessionEnd: &assistantSessionEndSignalPayload{
			Signal:  busAssistantSessionEndSignal,
			Message: cancelledHumanDialogueMessage,
		},
	}
}

func emitHumanResumeStreamError(
	ctx context.Context,
	sink streaming.Sink,
	traceID string,
	sessionID string,
	statusCode int,
	cause error,
) error {
	if emitErr := internaltrace.EmitStreamErrorEvent(ctx, sink, traceID, 0, "", sessionID, statusCode, cause); emitErr != nil {
		return emitErr
	}
	return cause
}

func (s *bridgeService) resolveHumanResumeStreamSession(
	ctx context.Context,
	rawSessionID string,
	traceID string,
	sink streaming.Sink,
) (string, error) {
	sessionID, err := requireSessionID(rawSessionID)
	if err != nil {
		return "", emitHumanResumeStreamError(
			ctx,
			sink,
			traceID,
			rawSessionID,
			bus.StatusFromError(err),
			err,
		)
	}
	if inflightErr := s.ensureSessionNotInflight(sessionID); inflightErr != nil {
		return sessionID, emitHumanResumeStreamError(
			ctx,
			sink,
			traceID,
			sessionID,
			bus.StatusFromError(inflightErr),
			inflightErr,
		)
	}
	if activeErr := s.ensureSessionActive(sessionID); activeErr != nil {
		return sessionID, emitHumanResumeStreamError(
			ctx,
			sink,
			traceID,
			sessionID,
			bus.StatusFromError(activeErr),
			activeErr,
		)
	}
	return sessionID, nil
}

func (s *bridgeService) applyHumanResponseForResumeStream(
	ctx context.Context,
	params humanResponseParams,
	traceID string,
	sink streaming.Sink,
) error {
	_, err := s.executeHumanResponseAction(ctx, params, traceID)
	if err == nil {
		return nil
	}
	return emitHumanResumeStreamError(
		ctx,
		sink,
		traceID,
		params.SessionID,
		bus.StatusFromError(err),
		err,
	)
}

func mapHumanResponseErrorKind(err error) ServiceErrorKind {
	switch {
	case errors.Is(err, appsessions.ErrSessionIDRequired),
		errors.Is(err, appsessions.ErrHumanQuestionIDRequired),
		errors.Is(err, appsessions.ErrHumanAnswerRequired):
		return ServiceErrorInvalidInput
	case errors.Is(err, appsessions.ErrHumanQuestionNotFound):
		return ServiceErrorNotFound
	default:
		return mapSessionAppErrorKind(err)
	}
}

func mapExternalAgentError(err error) ServiceErrorKind {
	switch {
	case errors.Is(err, session.ErrInvalidSessionID),
		errors.Is(err, appexternal.ErrSessionRequired),
		errors.Is(err, appexternal.ErrNotImplemented):
		return ServiceErrorInvalidInput
	case errors.Is(err, session.ErrSessionNotFound),
		errors.Is(err, appexternal.ErrApprovalNotFound):
		return ServiceErrorNotFound
	case errors.Is(err, appexternal.ErrExternalRunActive):
		return ServiceErrorConflict
	default:
		return ServiceErrorInternal
	}
}
