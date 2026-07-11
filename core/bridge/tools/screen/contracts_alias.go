package screen

import (
	"context"

	"ghost-os/bridge/session"
	toolcontracts "ghost-os/bridge/tools/contracts"
)

type AskHumanOption = toolcontracts.AskHumanOption

type AwaitingHumanSignal = toolcontracts.AwaitingHumanSignal

type ExecutionClient = toolcontracts.ExecutionClient

type ExecuteMeta = toolcontracts.ExecuteMeta

type ResultInterpreter = toolcontracts.ResultInterpreter

type Tool = toolcontracts.Tool

func InterpretExecuteResult(tool Tool, output string) ExecuteMeta {
	return toolcontracts.InterpretExecuteResult(tool, output)
}

func SessionFromContext(ctx context.Context) *session.Session {
	sess, _ := toolcontracts.SessionFromContext(ctx).(*session.Session)
	return sess
}

func ToolCallIDFromContext(ctx context.Context) string {
	return toolcontracts.ToolCallIDFromContext(ctx)
}

func WithSession(ctx context.Context, sess *session.Session) context.Context {
	return toolcontracts.WithSession(ctx, sess)
}

func WithToolCallID(ctx context.Context, toolCallID string) context.Context {
	return toolcontracts.WithToolCallID(ctx, toolCallID)
}
