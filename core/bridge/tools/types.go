package tools

import (
	"context"
	"encoding/json"

	"ghost-os/bridge/llm"
)

// Tool 定义 Agent 可调用的最小工具契约。
type Tool interface {
	Name() string
	Description() string
	Parameters() json.RawMessage
	Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error)
}

// AwaitingHumanSignal 表示工具要求 Agent 暂停并等待用户输入。
type AwaitingHumanSignal struct {
	QuestionID string
	Prompt     string
}

// ExecuteMeta 描述工具执行后的附加语义，不影响原始 output envelope。
type ExecuteMeta struct {
	Content       []llm.ContentPart
	AwaitingHuman *AwaitingHumanSignal
}

// ResultInterpreter 允许工具把自身 output 解释为编排层可消费的元信息。
type ResultInterpreter interface {
	InterpretResult(output string) ExecuteMeta
}

// InterpretExecuteResult 统一读取工具执行元信息，不支持时返回零值。
func InterpretExecuteResult(tool Tool, output string) ExecuteMeta {
	if tool == nil {
		return ExecuteMeta{}
	}
	interpreter, ok := tool.(ResultInterpreter)
	if !ok {
		return ExecuteMeta{}
	}
	return interpreter.InterpretResult(output)
}
