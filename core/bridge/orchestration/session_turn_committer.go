package orchestration

import (
	"context"
	"log"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/memoryaug"
	"ghost-os/bridge/session"
)

// SessionTurnCommitter 负责回合完成后的持久化与归档。
type SessionTurnCommitter struct {
	sessionStore   *session.Store
	memoryLearning memoryaug.LearningService
}

func newSessionTurnCommitter(sessionStore *session.Store, memoryLearning memoryaug.LearningService) *SessionTurnCommitter {
	return &SessionTurnCommitter{
		sessionStore:   sessionStore,
		memoryLearning: memoryLearning,
	}
}

// CommitTurn 将本轮新增消息写回会话存储，并在完整回合结束后触发自动学习。
func (c *SessionTurnCommitter) CommitTurn(
	ctx context.Context,
	sess *session.Session,
	messages []llm.Message,
	traceID string,
	completed bool,
) error {
	if c == nil || c.sessionStore == nil {
		return nil
	}
	for _, msg := range messages {
		sess.AddMessage(msg)
	}
	if err := c.sessionStore.Save(sess); err != nil {
		return err
	}
	if !completed || c.memoryLearning == nil {
		return nil
	}
	if err := c.memoryLearning.LearnFromTurn(ctx, memoryaug.LearnFromTurnInput{
		SessionID: sess.ID,
		TraceID:   traceID,
		Messages:  projectLearningMessages(messages),
	}); err != nil {
		log.Printf("trace_id=%s action=MEMORY_LEARN status=error error=%v", traceID, err)
	}
	return nil
}

func projectLearningMessages(messages []llm.Message) []memoryaug.TurnMessage {
	out := make([]memoryaug.TurnMessage, 0, len(messages))
	for _, message := range messages {
		out = append(out, memoryaug.TurnMessage{
			Role: string(message.Role),
			Text: message.Text,
		})
	}
	return out
}
