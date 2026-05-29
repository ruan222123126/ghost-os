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
