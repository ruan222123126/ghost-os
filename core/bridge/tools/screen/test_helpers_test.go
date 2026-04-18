package screen

import (
	"context"
)

type mockExecutionClient struct {
	callFunc func(ctx context.Context, action string, params map[string]any, traceID string) (map[string]any, error)
}

func (m mockExecutionClient) Call(
	ctx context.Context,
	action string,
	params map[string]any,
	traceID string,
) (map[string]any, error) {
	if m.callFunc == nil {
		return map[string]any{"ok": true}, nil
	}
	return m.callFunc(ctx, action, params, traceID)
}
