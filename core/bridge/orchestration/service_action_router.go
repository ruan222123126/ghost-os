package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type serviceActionRouter struct {
	handlers map[string]actionHandler
}

func newServiceActionRouter(initialCapacity int) *serviceActionRouter {
	if initialCapacity < 0 {
		initialCapacity = 0
	}
	return &serviceActionRouter{
		handlers: make(map[string]actionHandler, initialCapacity),
	}
}

func (r *serviceActionRouter) register(action string, handler actionHandler) {
	if r == nil {
		return
	}
	r.handlers[action] = handler
}

func (r *serviceActionRouter) handler(action string) (actionHandler, bool) {
	if r == nil {
		return nil, false
	}
	handler, ok := r.handlers[action]
	return handler, ok
}

func (r *serviceActionRouter) dispatch(ctx context.Context, action string, params json.RawMessage, traceID string) (ServiceResult, error) {
	handler, ok := r.handler(action)
	if !ok {
		return ServiceResult{}, wrapServiceError(ServiceErrorInvalidInput, r.unsupportedActionError(action))
	}
	return handler(ctx, params, traceID)
}

func (r *serviceActionRouter) unsupportedActionError(action string) error {
	registered := r.actionNames()
	return fmt.Errorf(
		"unsupported action %q, expected one of: %s",
		action,
		strings.Join(registered, "|"),
	)
}

func (r *serviceActionRouter) actionNames() []string {
	if r == nil {
		return nil
	}
	registered := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		registered = append(registered, name)
	}
	sort.Strings(registered)
	return registered
}
