package tools

import (
	"context"
	"encoding/json"
	"strings"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
)

// AskHumanOption 描述 ask_human 向用户展示的单个可选项。
type AskHumanOption struct {
	Label       string `json:"label"`
	AllowCustom bool   `json:"allow_custom,omitempty"`
}

// ToolCatalog 定义 Agent 与上下文层共享的最小工具目录契约。
type ToolCatalog interface {
	Get(name string) Tool
	ToolDefs() []llm.ToolDef
}

// Tool 定义 Agent 可调用的最小工具契约。
type Tool interface {
	Name() string
	Description() string
	Parameters() json.RawMessage
	Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error)
}

// ExecutionClient 是 bridge tool 调用 execution layer 的最小契约。
type ExecutionClient interface {
	Call(ctx context.Context, action string, params map[string]any, traceID string) (map[string]any, error)
}

// WorkingDirSetter 允许工具更新 execution 层的工作目录。
type WorkingDirSetter interface {
	SetWorkingDir(dir string) error
}

// AwaitingHumanSignal 表示工具要求 Agent 暂停并等待用户输入。
type AwaitingHumanSignal struct {
	QuestionID    string
	Prompt        string
	SelectionMode string
	Options       []AskHumanOption
}

// IterationHandoffSignal tells the orchestrator to end the current fresh-memory agent
// iteration and either hand off to the next agent or finish the pro run.
type IterationHandoffSignal struct {
	Did            string
	Remaining      string
	Completed      bool
	FinalMessage   string
	FinalChangeLog string
}

// ExecuteMeta 描述工具执行后的附加语义，不影响原始 output envelope。
type ExecuteMeta struct {
	Content       []llm.ContentPart
	AwaitingHuman *AwaitingHumanSignal
	Iteration     *IterationHandoffSignal
}

// ResultPostProcessor 允许工具在编排层统一规范化输出，避免把后处理塞进 Execute。
type ResultPostProcessor interface {
	PostProcessResult(output string, traceID string) (string, error)
}

// ResultInterpreter 允许工具把自身 output 解释为编排层可消费的元信息。
type ResultInterpreter interface {
	InterpretResult(output string) ExecuteMeta
}

type sessionContextKey struct{}
type toolCallIDContextKey struct{}

// WithSession 把当前会话注入 tool 执行上下文。
func WithSession(ctx context.Context, sess *session.Session) context.Context {
	if sess == nil {
		return ctx
	}
	return context.WithValue(ctx, sessionContextKey{}, sess)
}

// SessionFromContext 读取 tool 执行时绑定的会话。
func SessionFromContext(ctx context.Context) *session.Session {
	if ctx == nil {
		return nil
	}
	sess, _ := ctx.Value(sessionContextKey{}).(*session.Session)
	return sess
}

// WithToolCallID 注入当前工具调用 ID，便于工具写回可追踪状态。
func WithToolCallID(ctx context.Context, toolCallID string) context.Context {
	trimmed := strings.TrimSpace(toolCallID)
	if trimmed == "" {
		return ctx
	}
	return context.WithValue(ctx, toolCallIDContextKey{}, trimmed)
}

// ToolCallIDFromContext 返回当前工具调用 ID。
func ToolCallIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(toolCallIDContextKey{}).(string)
	return strings.TrimSpace(value)
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

// PostProcessExecuteResult 先执行可选输出后处理，再解释 tool meta。
func PostProcessExecuteResult(tool Tool, output string, traceID string) (string, ExecuteMeta, error) {
	processed := output
	if tool != nil {
		if processor, ok := tool.(ResultPostProcessor); ok {
			var err error
			processed, err = processor.PostProcessResult(output, traceID)
			if err != nil {
				return "", ExecuteMeta{}, err
			}
		}
	}
	return processed, InterpretExecuteResult(tool, processed), nil
}
