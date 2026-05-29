package dispatch

import (
	"context"
	"encoding/json"
	"log"

	"ghost-os/bridge/orchestration/internal/contracts/bus"
)

type TypedHandler[T any] func(context.Context, T, string) (bus.ServiceResult, error)

func RegisterTrace(router *Router, action string, handler TraceHandler) {
	RegisterTyped[map[string]any](router, action, func(ctx context.Context, _ map[string]any, traceID string) (bus.ServiceResult, error) {
		return handler(ctx, traceID)
	})
}

func RegisterTyped[T any](router *Router, action string, handler TypedHandler[T]) {
	if router == nil {
		return
	}
	router.Register(action, func(ctx context.Context, rawParams json.RawMessage, traceID string) (bus.ServiceResult, error) {
		params, err := DecodeActionParams[T](rawParams)
		if err != nil {
			return bus.ServiceResult{}, bus.WrapError(bus.ServiceErrorInvalidInput, err)
		}
		return handler(ctx, params, traceID)
	})
}

func LogAction(traceID string, action string, status string, err error) {
	if err != nil {
		log.Printf("trace_id=%s action=%s status=%s error=%v", traceID, action, status, err)
		return
	}
	log.Printf("trace_id=%s action=%s status=%s", traceID, action, status)
}
