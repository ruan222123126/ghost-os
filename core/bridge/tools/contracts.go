package tools

import (
	"context"
	"fmt"

	"ghost-os/bridge/session"
	toolcontracts "ghost-os/bridge/tools/contracts"
)

// AskHumanOption 描述 ask_human 向用户展示的单个可选项。
type AskHumanOption = toolcontracts.AskHumanOption

// ToolCatalog 定义 Agent 与上下文层共享的最小工具目录契约。
type ToolCatalog = toolcontracts.ToolCatalog

// Tool 定义 Agent 可调用的最小工具契约。
type Tool = toolcontracts.Tool

// ExecutionClient 是 bridge tool 调用 execution layer 的最小契约。
type ExecutionClient = toolcontracts.ExecutionClient

// WorkingDirSetter 允许工具更新 execution 层的工作目录。
type WorkingDirSetter = toolcontracts.WorkingDirSetter

// SessionCheckpoint 暴露工具可用的最小即时持久化能力。
type SessionCheckpoint interface {
	Save(*session.Session) error
}

type sessionCheckpointAdapter struct {
	checkpoint SessionCheckpoint
}

func (a sessionCheckpointAdapter) Save(state toolcontracts.SessionState) error {
	sess, ok := state.(*session.Session)
	if !ok {
		return fmt.Errorf("session checkpoint expects *session.Session, got %T", state)
	}
	return a.checkpoint.Save(sess)
}

// AwaitingHumanSignal 表示工具要求 Agent 暂停并等待用户输入。
type AwaitingHumanSignal = toolcontracts.AwaitingHumanSignal

// IterationHandoffSignal tells the orchestrator to end the current fresh-memory agent
// iteration and either hand off to the next agent or finish the pro run.
type IterationHandoffSignal = toolcontracts.IterationHandoffSignal

// ExecuteMeta 描述工具执行后的附加语义，不影响原始 output envelope。
type ExecuteMeta = toolcontracts.ExecuteMeta

// ResultPostProcessor 允许工具在编排层统一规范化输出，避免把后处理塞进 Execute。
type ResultPostProcessor = toolcontracts.ResultPostProcessor

// ResultInterpreter 允许工具把自身 output 解释为编排层可消费的元信息。
type ResultInterpreter = toolcontracts.ResultInterpreter

// WithSession 把当前会话注入 tool 执行上下文。
func WithSession(ctx context.Context, sess *session.Session) context.Context {
	return toolcontracts.WithSession(ctx, sess)
}

// SessionFromContext 读取 tool 执行时绑定的会话。
func SessionFromContext(ctx context.Context) *session.Session {
	sess, _ := toolcontracts.SessionFromContext(ctx).(*session.Session)
	return sess
}

// WithSessionCheckpoint 注入工具执行中允许使用的最小即时持久化能力。
func WithSessionCheckpoint(
	ctx context.Context,
	checkpoint SessionCheckpoint,
) context.Context {
	if checkpoint == nil {
		return ctx
	}
	return toolcontracts.WithSessionCheckpoint(ctx, sessionCheckpointAdapter{checkpoint: checkpoint})
}

// SessionCheckpointFromContext 读取工具执行中绑定的即时持久化能力。
func SessionCheckpointFromContext(ctx context.Context) SessionCheckpoint {
	checkpoint := toolcontracts.SessionCheckpointFromContext(ctx)
	if checkpoint == nil {
		return nil
	}
	adapter, ok := checkpoint.(sessionCheckpointAdapter)
	if !ok {
		return nil
	}
	return adapter.checkpoint
}

// WithToolCallID 注入当前工具调用 ID，便于工具写回可追踪状态。
func WithToolCallID(ctx context.Context, toolCallID string) context.Context {
	return toolcontracts.WithToolCallID(ctx, toolCallID)
}

// ToolCallIDFromContext 返回当前工具调用 ID。
func ToolCallIDFromContext(ctx context.Context) string {
	return toolcontracts.ToolCallIDFromContext(ctx)
}

// InterpretExecuteResult 统一读取工具执行元信息，不支持时返回零值。
func InterpretExecuteResult(tool Tool, output string) ExecuteMeta {
	return toolcontracts.InterpretExecuteResult(tool, output)
}

// PostProcessExecuteResult 先执行可选输出后处理，再解释 tool meta。
func PostProcessExecuteResult(tool Tool, output string, traceID string) (string, ExecuteMeta, error) {
	return toolcontracts.PostProcessExecuteResult(tool, output, traceID)
}
