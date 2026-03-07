package memory

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

// MemoryLifecycle 负责 warm/cold 之间的写入、归档与回填。
type MemoryLifecycle struct {
	warm  *WarmMemory
	cold  *ColdMemory
	graph *GraphService

	sessionStore SessionStorePort
	warmCapacity int
}

func NewMemoryLifecycle(config MemoryConfig, warm *WarmMemory, cold *ColdMemory, graph *GraphService, sessionStore SessionStorePort) *MemoryLifecycle {
	return &MemoryLifecycle{
		warm:         warm,
		cold:         cold,
		graph:        graph,
		sessionStore: sessionStore,
		warmCapacity: config.WarmCapacity,
	}
}

func (l *MemoryLifecycle) PromoteToWarm(sessionID string) error {
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		return errors.New("session id is required")
	}

	limit := l.warmCapacity
	if limit <= 0 {
		limit = defaultWarmCapacity
	}

	entries, err := l.cold.Retrieve(MemoryQuery{
		Limit:    limit,
		Metadata: map[string]any{"session_id": sid},
	})
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := l.warm.Store(entry); err != nil {
			return err
		}
	}
	return nil
}

func (l *MemoryLifecycle) ArchiveToCold(sessionID string) error {
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		return errors.New("session id is required")
	}

	messages, err := l.resolveMessagesForArchive(sid)
	if err != nil {
		return err
	}
	if len(messages) == 0 {
		return nil
	}

	if err := l.cold.Archive(sid, messages); err != nil {
		return err
	}
	if l.graph != nil {
		if err := l.graph.IngestArchiveMessages(sid, messages); err != nil {
			log.Printf("[MEMORY] graph archive ingest failed, continuing without graph update: session=%s err=%v", sid, err)
		}
	}
	return nil
}

func (l *MemoryLifecycle) StoreWarmMessages(sessionID string, startIndex int, messages []llm.Message) error {
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		return errors.New("session id is required")
	}
	if len(messages) == 0 {
		return nil
	}

	baseTime := time.Now().UTC()
	indexBase := startIndex
	if indexBase < 0 {
		indexBase = 0
	}

	for i, msg := range messages {
		content := strings.TrimSpace(messageToContent(msg))
		if content == "" {
			continue
		}

		entry := MemoryEntry{
			ID:             fmt.Sprintf("%s:%06d", sid, indexBase+i),
			Content:        content,
			Type:           MemoryTypeMessage,
			Timestamp:      baseTime.Add(time.Duration(i) * time.Millisecond),
			LastAccessedAt: baseTime.Add(time.Duration(i) * time.Millisecond),
			Source:         "conversation",
			Summary:        summarizeLine(content, 220),
			Confidence:     1,
			Metadata: map[string]any{
				"layer":        "warm",
				"source":       "conversation",
				"session_id":   sid,
				"role":         string(msg.Role),
				"tool_call_id": strings.TrimSpace(msg.ToolCallID),
			},
		}
		if err := l.warm.Store(entry); err != nil {
			return err
		}
	}
	return nil
}

func (l *MemoryLifecycle) resolveMessagesForArchive(sessionID string) ([]llm.Message, error) {
	if !hasSessionStore(l.sessionStore) {
		return nil, nil
	}

	sess, err := l.sessionStore.Load(sessionID)
	if err != nil {
		return nil, err
	}
	return llm.CloneMessages(sess.Messages), nil
}
