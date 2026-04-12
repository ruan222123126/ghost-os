package memory

import (
	"context"

	"ghost-os/bridge/session"
	toolcontracts "ghost-os/bridge/tools/contracts"
)

type Tool = toolcontracts.Tool

func SessionFromContext(ctx context.Context) *session.Session {
	sess, _ := toolcontracts.SessionFromContext(ctx).(*session.Session)
	return sess
}

func WithSession(ctx context.Context, sess *session.Session) context.Context {
	return toolcontracts.WithSession(ctx, sess)
}
