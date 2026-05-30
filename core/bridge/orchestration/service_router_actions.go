package orchestration

import (
	"context"

	"ghost-os/bridge/orchestration/internal/dispatch"
)

func registerDefaultActions(service *bridgeService) {
	if service == nil || service.actionRouter == nil {
		return
	}
	dispatch.RegisterDefaultActions(service.actionRouter.inner, defaultActionHandlers(service))
}

func defaultActionHandlers(service *bridgeService) dispatch.DefaultHandlers {
	return dispatch.DefaultHandlers{
		AgentSend: service.executeAgentAction,
		AgentStop: service.executeAgentStopAction,
		ConfigGet: func(_ context.Context, traceID string) (ServiceResult, error) {
			return service.executeConfigGetAction(traceID)
		},
		ConfigUpdate: func(_ context.Context, params configUpdateRequest, traceID string) (ServiceResult, error) {
			return service.executeConfigUpdateAction(params, traceID)
		},
		HumanResponse: service.executeHumanResponseAction,
		TaskCreate: func(_ context.Context, params taskCreateParams, traceID string) (ServiceResult, error) {
			return service.executeTaskCreateActionResult(params, traceID)
		},
		TaskList: service.executeTaskListDispatchAction,
		TaskGet: func(_ context.Context, params taskIDParams, traceID string) (ServiceResult, error) {
			return service.executeTaskGetActionResult(params, traceID)
		},
		TaskUpdate: func(_ context.Context, params taskUpdateParams, traceID string) (ServiceResult, error) {
			return service.executeTaskUpdateActionResult(params, traceID)
		},
		TaskRunNow: func(_ context.Context, params taskIDParams, traceID string) (ServiceResult, error) {
			return service.executeTaskRunNowActionResult(params, traceID)
		},
		TaskLogs: func(_ context.Context, params taskLogsParams, traceID string) (ServiceResult, error) {
			return service.executeTaskLogsActionResult(params, traceID)
		},
		TaskDelete: func(_ context.Context, params taskIDParams, traceID string) (ServiceResult, error) {
			return service.executeTaskDeleteActionResult(params, traceID)
		},
	}
}

func (s *bridgeService) executeTaskListDispatchAction(
	_ context.Context,
	params dispatch.TaskListParams,
	traceID string,
) (ServiceResult, error) {
	scope, err := dispatch.NormalizeTaskListScope(params.Scope)
	if err != nil {
		return ServiceResult{}, wrapServiceError(ServiceErrorInvalidInput, err)
	}
	return s.executeTaskListActionResult(scope, traceID)
}

func normalizeTaskListScope(scope string) (string, error) {
	return dispatch.NormalizeTaskListScope(scope)
}
