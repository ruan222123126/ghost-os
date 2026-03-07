package app

import (
	"log"
	"strings"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/memory"
	"ghost-os/bridge/session"
)

// SessionTurnCommitter 负责回合完成后的持久化与归档。
type SessionTurnCommitter struct {
	sessionStore  *session.Store
	memoryManager *memory.MemoryManager
	traceID       string
}

func newSessionTurnCommitter(sessionStore *session.Store, memoryManager *memory.MemoryManager, traceID string) *SessionTurnCommitter {
	return &SessionTurnCommitter{
		sessionStore:  sessionStore,
		memoryManager: memoryManager,
		traceID:       strings.TrimSpace(traceID),
	}
}

// PersistSessionMessages 将本轮新增消息写回会话存储。
func (c *SessionTurnCommitter) PersistSessionMessages(sess *session.Session, messages []llm.Message) error {
	if c == nil || c.sessionStore == nil {
		return nil
	}

	startIndex := len(sess.Messages)
	for _, msg := range messages {
		sess.AddMessage(msg)
	}
	if err := c.sessionStore.Save(sess); err != nil {
		return err
	}

	if c.memoryManager != nil {
		if err := c.memoryManager.AppendLedgerTurn(sess.ID, c.traceID, startIndex, messages); err != nil {
			log.Printf(
				"trace_id=%s action=MEMORY_LEDGER_APPEND status=error session_id=%s error=%v",
				c.traceID,
				strings.TrimSpace(sess.ID),
				err,
			)
		}
		if err := c.memoryManager.StoreWarmMessages(sess.ID, startIndex, messages); err != nil {
			log.Printf(
				"trace_id=%s action=MEMORY_WARM_STORE status=error session_id=%s error=%v",
				c.traceID,
				strings.TrimSpace(sess.ID),
				err,
			)
		}
	}
	return nil
}

// ArchiveSession 在回合结束后触发冷存归档；失败仅记录日志不影响主流程。
func (c *SessionTurnCommitter) ArchiveSession(sessionID string) {
	if c == nil || c.memoryManager == nil {
		return
	}
	if err := c.memoryManager.ArchiveToCold(sessionID); err != nil {
		log.Printf(
			"trace_id=%s action=MEMORY_ARCHIVE status=error session_id=%s error=%v",
			c.traceID,
			strings.TrimSpace(sessionID),
			err,
		)
		return
	}
	if err := markSessionMemoryArchived(c.sessionStore, sessionID); err != nil {
		log.Printf(
			"trace_id=%s action=SESSION_MEMORY_ARCHIVE_MARK status=error session_id=%s error=%v",
			c.traceID,
			strings.TrimSpace(sessionID),
			err,
		)
	}
}
