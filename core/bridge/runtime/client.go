package runtime

import "context"

// Response 是 runtime execution client 返回给 central layer 的原子执行结果。
type Response = map[string]any

// Client 定义 runtime 调用 execution layer 的最小契约。
type Client interface {
	Call(ctx context.Context, action string, params map[string]any, traceID string) (Response, error)
}
