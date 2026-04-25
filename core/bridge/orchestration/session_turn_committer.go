package orchestration

import (
	"context"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
)

// SessionTurnCommitter 负责回合完成后的持久化与归档。
type SessionTurnCommitter struct {
	sessionStore *session.Store
}

func newSessionTurnCommitter(sessionStore *session.Store) *SessionTurnCommitter {
	return &SessionTurnCommitter{
		sessionStore: sessionStore,
	}
}

// CommitTurn 将本轮新增消息写回会话存储。
func (c *SessionTurnCommitter) CommitTurn(
	ctx context.Context,
	sess *session.Session,
	messages []llm.Message,
	traceID string,
	completed bool,
) error {
	_ = ctx
	_ = traceID
	_ = completed
	if c == nil || c.sessionStore == nil {
		return nil
	}
	for _, msg := range messages {
		sess.AddMessage(msg)
	}
	if err := c.sessionStore.Save(sess); err != nil {
		return err
	}
	return nil
}
