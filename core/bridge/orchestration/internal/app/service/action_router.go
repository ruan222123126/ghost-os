package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"ghost-os/bridge/orchestration/internal/contracts/bus"
)

type ActionHandler func(ctx context.Context, params json.RawMessage, traceID string) (bus.ServiceResult, error)

type ActionRouter struct {
	handlers map[string]ActionHandler
}

func NewActionRouter(initialCapacity int) *ActionRouter {
	if initialCapacity < 0 {
		initialCapacity = 0
	}
	return &ActionRouter{handlers: make(map[string]ActionHandler, initialCapacity)}
}

func (r *ActionRouter) Register(action string, handler ActionHandler) {
	if r == nil {
		return
	}
	r.handlers[action] = handler
}

func (r *ActionRouter) Handler(action string) (ActionHandler, bool) {
	if r == nil {
		return nil, false
	}
	handler, ok := r.handlers[action]
	return handler, ok
}

func (r *ActionRouter) Dispatch(
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

func (r *ActionRouter) UnsupportedActionError(action string) error {
	return fmt.Errorf(
		"unsupported action %q, expected one of: %s",
		action,
		strings.Join(r.ActionNames(), "|"),
	)
}

func (r *ActionRouter) ActionNames() []string {
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
