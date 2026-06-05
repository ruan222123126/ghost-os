package orchestration

import (
	"context"
	"encoding/json"

	"ghost-os/bridge/orchestration/internal/dispatch"
)

type serviceActionRouter struct {
	inner *dispatch.Router
}

func newServiceActionRouter(initialCapacity int) *serviceActionRouter {
	return &serviceActionRouter{
		inner: dispatch.NewRouter(initialCapacity),
	}
}

func (r *serviceActionRouter) register(action string, handler actionHandler) {
	if r == nil || r.inner == nil {
		return
	}
	r.inner.Register(action, dispatch.Handler(handler))
}

func (r *serviceActionRouter) handler(action string) (actionHandler, bool) {
	if r == nil || r.inner == nil {
		return nil, false
	}
	handler, ok := r.inner.Handler(action)
	return actionHandler(handler), ok
}

func (r *serviceActionRouter) dispatch(ctx context.Context, action string, params json.RawMessage, traceID string) (ServiceResult, error) {
	return r.inner.Dispatch(ctx, action, params, traceID)
}

func (r *serviceActionRouter) unsupportedActionError(action string) error {
	return r.inner.UnsupportedActionError(action)
}

func (r *serviceActionRouter) actionNames() []string {
	if r == nil || r.inner == nil {
		return nil
	}
	return r.inner.ActionNames()
}

func decodeActionParams[T any](raw json.RawMessage) (T, error) {
	return dispatch.DecodeActionParams[T](raw)
}

func validateBusRequest(req apiRequest) error {
	return dispatch.ValidateBusRequest(req)
}

func decodeParams(raw json.RawMessage, target any) error {
	return dispatch.DecodeParams(raw, target)
}

func logAction(traceID string, action string, status string, err error) {
	dispatch.LogAction(traceID, action, status, err)
}

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
		TaskStop: service.executeTaskStopActionResult,
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
