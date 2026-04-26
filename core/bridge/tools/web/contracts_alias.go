package web

import (
	"context"

	"ghost-os/bridge/session"
	toolcontracts "ghost-os/bridge/tools/contracts"
)

type Tool = toolcontracts.Tool

type ExecuteMeta = toolcontracts.ExecuteMeta

func SessionFromContext(ctx context.Context) *session.Session {
	sess, _ := toolcontracts.SessionFromContext(ctx).(*session.Session)
	return sess
}

func ToolCallIDFromContext(ctx context.Context) string {
	return toolcontracts.ToolCallIDFromContext(ctx)
}
