package dispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"ghost-os/bridge/orchestration/internal/contracts/bus"
)

type Handler func(ctx context.Context, params json.RawMessage, traceID string) (bus.ServiceResult, error)

type Router struct {
	handlers map[string]Handler
}

func NewRouter(initialCapacity int) *Router {
	if initialCapacity < 0 {
		initialCapacity = 0
	}
	return &Router{handlers: make(map[string]Handler, initialCapacity)}
}

func (r *Router) Register(action string, handler Handler) {
	if r == nil {
		return
	}
	r.handlers[action] = handler
}

func (r *Router) Handler(action string) (Handler, bool) {
	if r == nil {
		return nil, false
	}
	handler, ok := r.handlers[action]
	return handler, ok
}

func (r *Router) Dispatch(
	ctx context.Context,
	action string,
	params json.RawMessage,
	traceID string,
) (bus.ServiceResult, error) {
	handler, ok := r.Handler(action)
	if !ok {
		return bus.ServiceResult{}, bus.WrapError(bus.ServiceErrorInvalidInput, r.UnsupportedActionError(action))
	}
	return handler(ctx, params, traceID)
}

func (r *Router) UnsupportedActionError(action string) error {
	return fmt.Errorf(
		"unsupported action %q, expected one of: %s",
		action,
		strings.Join(r.ActionNames(), "|"),
	)
}

func (r *Router) ActionNames() []string {
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
