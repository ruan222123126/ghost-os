package tools

import (
	"context"
	"encoding/json"
)

// Tool 定义 Agent 可调用的最小工具契约。
type Tool interface {
	Name() string
	Description() string
	Parameters() json.RawMessage
	Execute(ctx context.Context, argsJSON json.RawMessage) (string, error)
}
