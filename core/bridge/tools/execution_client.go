package tools

import "context"

// ExecutionClient 是 bridge tool 调用 execution layer 的最小契约。
type ExecutionClient interface {
	Call(ctx context.Context, action string, params map[string]any, traceID string) (map[string]any, error)
}
