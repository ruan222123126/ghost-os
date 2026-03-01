package memory

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
)

// Summarizer 为后续自动摘要能力预留。
type Summarizer interface {
	Summarize(messages []llm.Message) (string, error)
}

// MemoryConfig 定义三层记忆管理器初始化参数。
type MemoryConfig struct {
	WarmCapacity int
	WarmPath     string
	ColdBaseDir  string

	// L1/L2/L3 运行态绑定项。
	HotHistory   *agent.History
	HotSessionID string
	SessionStore *session.Store
	Summarizer   Summarizer
}

// MemoryManager 统一协调 L1/L2/L3 记忆层。
type MemoryManager struct {
	hot          *agent.History
	hotSessionID string
	warm         *WarmMemory
	cold         *ColdMemory
	markdown     *MarkdownStore
	sessionStore *session.Store
	summarizer   Summarizer
	config       MemoryConfig
	mu           sync.RWMutex
}

func NewMemoryManager(config MemoryConfig) *MemoryManager {
	warm := NewWarmMemory(config.WarmCapacity, config.WarmPath)
	_ = warm.Load()

	markdownDir := ""
	if config.ColdBaseDir != "" {
		markdownDir = config.ColdBaseDir + "/markdown"
	}

	manager := &MemoryManager{
		hot:          config.HotHistory,
		hotSessionID: strings.TrimSpace(config.HotSessionID),
		warm:         warm,
		cold:         NewColdMemory(config.ColdBaseDir),
		markdown:     NewMarkdownStore(markdownDir),
		sessionStore: config.SessionStore,
		summarizer:   config.Summarizer,
		config:       config,
	}
	return manager
}

func (m *MemoryManager) SetHotContext(sessionID string, history *agent.History) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hot = history
	m.hotSessionID = strings.TrimSpace(sessionID)
}

func (m *MemoryManager) GetHotContext() []llm.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.hot == nil {
		return nil
	}
	return m.hot.Messages()
}

// Query 按 L1 -> L2 -> L3 顺序级联检索。
func (m *MemoryManager) Query(query MemoryQuery) ([]MemoryEntry, error) {
	results := make([]MemoryEntry, 0, 32)
	seen := make(map[string]struct{}, 32)

	collect := func(entries []MemoryEntry) {
		for _, entry := range entries {
			if _, ok := seen[entry.ID]; ok {
				continue
			}
			seen[entry.ID] = struct{}{}
			results = append(results, entry)
		}
	}

	hotEntries := m.queryHot(query)
	collect(hotEntries)
	if query.Limit > 0 && len(results) >= query.Limit {
		return results[:query.Limit], nil
	}

	warmQuery := query
	if query.Limit > 0 {
		warmQuery.Limit = query.Limit - len(results)
	}
	warmEntries, err := m.warm.Retrieve(warmQuery)
	if err != nil {
		return nil, err
	}
	collect(warmEntries)
	if query.Limit > 0 && len(results) >= query.Limit {
		out := results[:query.Limit]
		m.markSessionsAccessed(out)
		return out, nil
	}

	coldQuery := query
	if query.Limit > 0 {
		coldQuery.Limit = query.Limit - len(results)
	}
	coldEntries, err := m.cold.Retrieve(coldQuery)
	if err != nil {
		return nil, err
	}
	collect(coldEntries)

	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}
	m.markSessionsAccessed(results)
	return results, nil
}

// PromoteToWarm 手动把冷数据会话提升到温数据层。
func (m *MemoryManager) PromoteToWarm(sessionID string) error {
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		return errors.New("session id is required")
	}

	limit := m.config.WarmCapacity
	if limit <= 0 {
		limit = defaultWarmCapacity
	}
	query := MemoryQuery{
		Limit:    limit,
		Metadata: map[string]any{"session_id": sid},
	}
	entries, err := m.cold.Retrieve(query)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := m.warm.Store(entry); err != nil {
			return err
		}
	}
	return nil
}

// ArchiveToCold 把当前会话写入冷数据层。
func (m *MemoryManager) ArchiveToCold(sessionID string) error {
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		m.mu.RLock()
		sid = strings.TrimSpace(m.hotSessionID)
		m.mu.RUnlock()
	}
	if sid == "" {
		return errors.New("session id is required")
	}

	messages, err := m.resolveMessagesForArchive(sid)
	if err != nil {
		return err
	}
	if len(messages) == 0 {
		return nil
	}

	if err := m.cold.Archive(sid, messages); err != nil {
		return err
	}
	return m.markSessionArchived(sid)
}

func (m *MemoryManager) queryHot(query MemoryQuery) []MemoryEntry {
	m.mu.RLock()
	hot := m.hot
	hotSessionID := m.hotSessionID
	m.mu.RUnlock()
	if hot == nil {
		return nil
	}

	messages := hot.Messages()
	if len(messages) == 0 {
		return nil
	}

	base := time.Now().UTC()
	out := make([]MemoryEntry, 0, len(messages))
	for i, msg := range messages {
		entry := MemoryEntry{
			ID:        fmt.Sprintf("hot:%s:%06d", hotSessionID, i),
			Content:   messageToContent(msg),
			Type:      MemoryTypeMessage,
			Timestamp: base.Add(time.Duration(i) * time.Millisecond),
			Metadata: map[string]any{
				"layer":        "hot",
				"session_id":   hotSessionID,
				"role":         string(msg.Role),
				"tool_call_id": strings.TrimSpace(msg.ToolCallID),
			},
		}
		entry = normalizeEntry(entry)
		if !entryMatchesQuery(entry, query) {
			continue
		}
		out = append(out, entry)
	}
	if query.Limit > 0 && len(out) > query.Limit {
		return out[len(out)-query.Limit:]
	}
	return out
}

func (m *MemoryManager) resolveMessagesForArchive(sessionID string) ([]llm.Message, error) {
	m.mu.RLock()
	hot := m.hot
	hotSessionID := m.hotSessionID
	store := m.sessionStore
	m.mu.RUnlock()

	if hot != nil && strings.TrimSpace(sessionID) == strings.TrimSpace(hotSessionID) {
		return hot.Messages(), nil
	}
	if store == nil {
		return nil, nil
	}

	sess, err := store.Load(sessionID)
	if err != nil {
		return nil, err
	}
	return llm.CloneMessages(sess.Messages), nil
}

func (m *MemoryManager) markSessionArchived(sessionID string) error {
	m.mu.RLock()
	store := m.sessionStore
	m.mu.RUnlock()
	if store == nil {
		return nil
	}

	sess, err := store.Load(sessionID)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			return nil
		}
		return err
	}
	sess.MarkMemoryArchived(time.Now().UTC())
	return store.Save(sess)
}

// SaveMarkdownNode 保存记忆节点为 Markdown 文件（人类可读）。
func (m *MemoryManager) SaveMarkdownNode(node MarkdownNode) error {
	m.mu.RLock()
	markdown := m.markdown
	m.mu.RUnlock()
	if markdown == nil {
		return fmt.Errorf("markdown store is not configured")
	}
	return markdown.Save(node)
}

// LoadMarkdownNode 加载 Markdown 记忆节点。
func (m *MemoryManager) LoadMarkdownNode(id string) (MarkdownNode, error) {
	m.mu.RLock()
	markdown := m.markdown
	m.mu.RUnlock()
	if markdown == nil {
		return MarkdownNode{}, fmt.Errorf("markdown store is not configured")
	}
	return markdown.Load(id)
}

// ListMarkdownNodes 列出所有 Markdown 记忆节点。
func (m *MemoryManager) ListMarkdownNodes() ([]string, error) {
	m.mu.RLock()
	markdown := m.markdown
	m.mu.RUnlock()
	if markdown == nil {
		return nil, nil
	}
	return markdown.List()
}

func (m *MemoryManager) markSessionsAccessed(entries []MemoryEntry) {
	m.mu.RLock()
	store := m.sessionStore
	m.mu.RUnlock()
	if store == nil || len(entries) == 0 {
		return
	}

	sessionIDs := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		raw, ok := entry.Metadata["session_id"]
		if !ok {
			continue
		}
		sessionID, ok := raw.(string)
		if !ok || strings.TrimSpace(sessionID) == "" {
			continue
		}
		sessionIDs[strings.TrimSpace(sessionID)] = struct{}{}
	}

	now := time.Now().UTC()
	for sessionID := range sessionIDs {
		sess, err := store.Load(sessionID)
		if err != nil {
			continue
		}
		sess.MarkMemoryAccess(now)
		_ = store.Save(sess)
	}
}
