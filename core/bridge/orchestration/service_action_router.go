package orchestration

import (
	"context"
	"encoding/json"

	appservice "ghost-os/bridge/orchestration/internal/app/service"
)

type serviceActionRouter struct {
	inner *appservice.ActionRouter
}

func newServiceActionRouter(initialCapacity int) *serviceActionRouter {
	return &serviceActionRouter{
		inner: appservice.NewActionRouter(initialCapacity),
	}
}

func (r *serviceActionRouter) register(action string, handler actionHandler) {
	if r == nil || r.inner == nil {
		return
	}
	r.inner.Register(action, appservice.ActionHandler(handler))
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
