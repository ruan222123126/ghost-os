package memory

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
)

const (
	defaultAutoRecallLimit   = 5
	defaultEvolutionInterval = time.Hour
)

// Summarizer 为自动摘要能力提供统一接口。
type Summarizer interface {
	Summarize(messages []llm.Message) (string, error)
}

// MemoryConfig 定义三层记忆管理器初始化参数。
type MemoryConfig struct {
	WarmCapacity int
	WarmPath     string
	ColdBaseDir  string

	AutoRecallEnabled bool
	AutoRecallLimit   int
	WarmTTL           time.Duration
	EvolutionInterval time.Duration
	EvolutionEnabled  bool

	// 运行态绑定依赖。
	SessionStore *session.Store
	Summarizer   Summarizer
}

// SessionScope 显式描述一次请求可见的会话热态上下文，避免共享管理器持有可变请求态。
type SessionScope struct {
	SessionID string
	History   *agent.History
}

// EvolutionStats 描述一次后台演化流程的处理结果。
type EvolutionStats struct {
	ScannedEntries   int
	ProcessedEntries int
	NodesCreated     int
}

// MemoryMetrics 是管理器运行指标快照。
type MemoryMetrics struct {
	L1Hits         uint64 `json:"l1_hits"`
	L2Hits         uint64 `json:"l2_hits"`
	L3Hits         uint64 `json:"l3_hits"`
	MarkdownHits   uint64 `json:"markdown_hits"`
	EvolutionRuns  uint64 `json:"evolution_runs"`
	NodesCreated   uint64 `json:"nodes_created"`
	EntriesEvolved uint64 `json:"entries_evolved"`
}

// MemoryManager 统一协调 L1/L2/L3 记忆层。
type MemoryManager struct {
	warm         *WarmMemory
	cold         *ColdMemory
	sessionStore *session.Store
	summarizer   Summarizer
	config       MemoryConfig

	dreamStopOnce sync.Once
	dreamStop     chan struct{}
	dreamOnce     sync.Once
	dreamWG       sync.WaitGroup

	l1Hits         atomic.Uint64
	l2Hits         atomic.Uint64
	l3Hits         atomic.Uint64
	markdownHits   atomic.Uint64
	evolutionRuns  atomic.Uint64
	nodesCreated   atomic.Uint64
	entriesEvolved atomic.Uint64
}

func NewMemoryManager(config MemoryConfig) *MemoryManager {
	normalized := normalizeMemoryConfig(config)

	warm := NewWarmMemoryWithTTL(normalized.WarmCapacity, normalized.WarmPath, normalized.WarmTTL)
	_ = warm.Load()
	cold := NewColdMemory(normalized.ColdBaseDir)

	manager := &MemoryManager{
		warm:         warm,
		cold:         cold,
		sessionStore: normalized.SessionStore,
		summarizer:   normalized.Summarizer,
		config:       normalized,
		dreamStop:    make(chan struct{}),
	}
	if normalized.EvolutionEnabled {
		manager.StartDreaming()
	}
	return manager
}

func normalizeMemoryConfig(config MemoryConfig) MemoryConfig {
	out := config
	if out.WarmCapacity <= 0 {
		out.WarmCapacity = defaultWarmCapacity
	}
	if out.AutoRecallLimit <= 0 {
		out.AutoRecallLimit = defaultAutoRecallLimit
	}
	if out.WarmTTL <= 0 {
		out.WarmTTL = defaultWarmTTL
	}
	if out.EvolutionInterval <= 0 {
		out.EvolutionInterval = defaultEvolutionInterval
	}
	if out.ColdBaseDir != "" {
		out.ColdBaseDir = filepath.Clean(out.ColdBaseDir)
	}
	return out
}

// BuildContextWindow 从 warm 层构建自动召回上下文，供 Agent 在当前轮次注入。
func (m *MemoryManager) BuildContextWindow(sessionID string, userInput string) ([]llm.Message, error) {
	if !m.config.AutoRecallEnabled {
		return nil, nil
	}

	query := MemoryQuery{
		Limit:         m.config.AutoRecallLimit,
		Keywords:      extractKeywords(userInput),
		SemanticQuery: strings.TrimSpace(userInput),
		UseTimeDecay:  true,
	}

	if sid := strings.TrimSpace(sessionID); sid != "" {
		query.Metadata = map[string]any{"session_id": sid}
	}

	entries, err := m.warm.Retrieve(query)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}
	m.l2Hits.Add(uint64(len(entries)))

	lines := make([]string, 0, len(entries)+1)
	lines = append(lines, "Auto-recalled memory context:")
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("- [importance=%.2f] %s", entry.Importance, summarizeLine(entry.Content, 220)))
	}

	return []llm.Message{{
		Role: llm.RoleSystem,
		Text: strings.Join(lines, "\n"),
	}}, nil
}

// Query 按 L1 -> L2 -> L3 -> Markdown 顺序级联检索。
func (m *MemoryManager) Query(query MemoryQuery) ([]MemoryEntry, error) {
	return m.QueryWithScope(query, SessionScope{})
}

// QueryWithScope 在统一级联检索中显式接收请求级热态上下文，避免共享状态串味。
func (m *MemoryManager) QueryWithScope(query MemoryQuery, scope SessionScope) ([]MemoryEntry, error) {
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

	hotEntries := queryHot(scope, query)
	if len(hotEntries) > 0 {
		m.l1Hits.Add(uint64(len(hotEntries)))
	}
	collect(hotEntries)
	if query.Limit > 0 && len(results) >= query.Limit {
		out := results[:query.Limit]
		m.markSessionsAccessed(out)
		return out, nil
	}

	warmQuery := query
	if query.Limit > 0 {
		warmQuery.Limit = query.Limit - len(results)
	}
	warmEntries, err := m.warm.Retrieve(warmQuery)
	if err != nil {
		return nil, err
	}
	if len(warmEntries) > 0 {
		m.l2Hits.Add(uint64(len(warmEntries)))
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
	if len(coldEntries) > 0 {
		m.l3Hits.Add(uint64(len(coldEntries)))
	}
	collect(coldEntries)

	if query.IncludeMarkdown {
		markdownQuery := query
		if query.Limit > 0 {
			markdownQuery.Limit = query.Limit - len(results)
		}
		markdownEntries, err := m.queryMarkdown(markdownQuery)
		if err != nil {
			return nil, err
		}
		if len(markdownEntries) > 0 {
			m.markdownHits.Add(uint64(len(markdownEntries)))
		}
		collect(markdownEntries)
	}

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

// StoreWarmMessages 将新增会话消息写入 warm 层，供后续自动召回使用。
func (m *MemoryManager) StoreWarmMessages(sessionID string, startIndex int, messages []llm.Message) error {
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
			ID:        fmt.Sprintf("%s:%06d", sid, indexBase+i),
			Content:   content,
			Type:      MemoryTypeMessage,
			Timestamp: baseTime.Add(time.Duration(i) * time.Millisecond),
			Metadata: map[string]any{
				"layer":        "warm",
				"session_id":   sid,
				"role":         string(msg.Role),
				"tool_call_id": strings.TrimSpace(msg.ToolCallID),
			},
		}
		if err := m.warm.Store(entry); err != nil {
			return err
		}
	}
	return nil
}

// Evolve 扫描 warm 层低重要度旧条目，沉淀为 markdown 记忆节点。
func (m *MemoryManager) Evolve() (EvolutionStats, error) {
	stats := EvolutionStats{}
	m.evolutionRuns.Add(1)

	entries := m.warm.Snapshot()
	stats.ScannedEntries = len(entries)
	log.Printf("[MEMORY] Evolution started: scanning L2 (%d entries)", stats.ScannedEntries)
	if len(entries) == 0 {
		log.Printf("[MEMORY] Evolution complete: processed %d entries, created %d nodes", stats.ProcessedEntries, stats.NodesCreated)
		return stats, nil
	}

	now := time.Now().UTC()
	minAge := m.config.WarmTTL / 4
	if minAge < 30*time.Minute {
		minAge = 30 * time.Minute
	}

	groups := make(map[string][]MemoryEntry, 8)
	for _, entry := range entries {
		age := now.Sub(entry.Timestamp)
		if age < minAge {
			continue
		}
		if entry.Importance >= 0.7 {
			continue
		}
		groupKey := evolutionGroupKey(entry)
		groups[groupKey] = append(groups[groupKey], entry)
	}

	for _, group := range groups {
		if len(group) == 0 {
			continue
		}

		relatedIDs := make([]string, 0, len(group))
		tags := make(map[string]struct{}, 8)
		sessionID := ""
		for _, entry := range group {
			relatedIDs = append(relatedIDs, entry.ID)
			for _, tag := range extractKeywords(entry.Content) {
				tags[tag] = struct{}{}
			}
			if sid, ok := entry.Metadata["session_id"].(string); ok && strings.TrimSpace(sid) != "" {
				sessionID = strings.TrimSpace(sid)
			}
		}

		summary := m.summarizeGroup(group)
		node := MarkdownNode{
			ID:         fmt.Sprintf("node_%d", time.Now().UTC().UnixNano()),
			Importance: averageImportance(group),
			CreatedAt:  now,
			RelatedTo:  relatedIDs,
			Tags:       sortedTagList(tags),
			SessionID:  sessionID,
			Content:    summary,
		}

		if err := m.cold.SaveMarkdownNode(node); err != nil {
			return stats, err
		}

		for _, entry := range group {
			_ = m.warm.Delete(entry.ID)
		}

		stats.ProcessedEntries += len(group)
		stats.NodesCreated++
		m.nodesCreated.Add(1)
		m.entriesEvolved.Add(uint64(len(group)))
		log.Printf("[MEMORY] Created markdown node: %s (importance=%.2f, related=%d)", node.ID, node.Importance, len(node.RelatedTo))
	}

	log.Printf("[MEMORY] Evolution complete: processed %d entries, created %d nodes", stats.ProcessedEntries, stats.NodesCreated)
	return stats, nil
}

// StartDreaming 启动后台演化协程。
func (m *MemoryManager) StartDreaming() {
	if !m.config.EvolutionEnabled {
		return
	}
	m.dreamOnce.Do(func() {
		interval := m.config.EvolutionInterval
		if interval <= 0 {
			interval = defaultEvolutionInterval
		}

		m.dreamWG.Add(1)
		go func() {
			defer m.dreamWG.Done()
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-m.dreamStop:
					return
				case <-ticker.C:
					if _, err := m.Evolve(); err != nil {
						log.Printf("[MEMORY] Evolution error: %v", err)
					}
				}
			}
		}()
	})
}

// StopDreaming 停止后台演化协程，并等待其优雅退出。
func (m *MemoryManager) StopDreaming() {
	if m == nil {
		return
	}
	m.dreamStopOnce.Do(func() {
		close(m.dreamStop)
	})
	m.dreamWG.Wait()
}

func queryHot(scope SessionScope, query MemoryQuery) []MemoryEntry {
	if scope.History == nil {
		return nil
	}

	sessionID := strings.TrimSpace(scope.SessionID)
	messages := scope.History.Messages()
	if len(messages) == 0 {
		return nil
	}

	base := time.Now().UTC()
	out := make([]MemoryEntry, 0, len(messages))
	for i, msg := range messages {
		entry := MemoryEntry{
			ID:        fmt.Sprintf("hot:%s:%06d", sessionID, i),
			Content:   messageToContent(msg),
			Type:      MemoryTypeMessage,
			Timestamp: base.Add(time.Duration(i) * time.Millisecond),
			Metadata: map[string]any{
				"layer":        "hot",
				"session_id":   sessionID,
				"role":         string(msg.Role),
				"tool_call_id": strings.TrimSpace(msg.ToolCallID),
			},
		}
		entry = normalizeEntry(entry)
		entry.Importance = calculateImportance(entry)
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

func (m *MemoryManager) queryMarkdown(query MemoryQuery) ([]MemoryEntry, error) {
	nodes, err := m.cold.ListMarkdownNodes()
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, nil
	}

	entries := make([]MemoryEntry, 0, len(nodes))
	for _, id := range nodes {
		node, err := m.cold.LoadMarkdownNode(id)
		if err != nil {
			continue
		}

		entry := normalizeEntry(MemoryEntry{
			ID:          "markdown:" + node.ID,
			Content:     strings.TrimSpace(node.Content),
			Type:        MemoryTypeKnowledge,
			Timestamp:   node.CreatedAt.UTC(),
			Importance:  node.Importance,
			RelatedTo:   append([]string(nil), node.RelatedTo...),
			EmbeddingID: node.EmbeddingID,
			Metadata: map[string]any{
				"layer":      "markdown",
				"session_id": strings.TrimSpace(node.SessionID),
				"tags":       append([]string(nil), node.Tags...),
				"node_id":    node.ID,
			},
		})
		if !entryMatchesQuery(entry, query) {
			continue
		}
		entries = append(entries, entry)
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Importance != entries[j].Importance {
			return entries[i].Importance > entries[j].Importance
		}
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})
	if query.Limit > 0 && len(entries) > query.Limit {
		entries = entries[:query.Limit]
	}
	return entries, nil
}

func (m *MemoryManager) resolveMessagesForArchive(sessionID string) ([]llm.Message, error) {
	if m.sessionStore == nil {
		return nil, nil
	}

	sess, err := m.sessionStore.Load(sessionID)
	if err != nil {
		return nil, err
	}
	return llm.CloneMessages(sess.Messages), nil
}

func (m *MemoryManager) markSessionArchived(sessionID string) error {
	if m.sessionStore == nil {
		return nil
	}

	sess, err := m.sessionStore.Load(sessionID)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			return nil
		}
		return err
	}
	sess.MarkMemoryArchived(time.Now().UTC())
	return m.sessionStore.Save(sess)
}

// SaveMarkdownNode 保存记忆节点为 Markdown 文件（人类可读）。
func (m *MemoryManager) SaveMarkdownNode(node MarkdownNode) error {
	return m.cold.SaveMarkdownNode(node)
}

// LoadMarkdownNode 加载 Markdown 记忆节点。
func (m *MemoryManager) LoadMarkdownNode(id string) (MarkdownNode, error) {
	return m.cold.LoadMarkdownNode(id)
}

// ListMarkdownNodes 列出所有 Markdown 记忆节点。
func (m *MemoryManager) ListMarkdownNodes() ([]string, error) {
	return m.cold.ListMarkdownNodes()
}

func (m *MemoryManager) markSessionsAccessed(entries []MemoryEntry) {
	if m.sessionStore == nil || len(entries) == 0 {
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
		sess, err := m.sessionStore.Load(sessionID)
		if err != nil {
			continue
		}
		sess.MarkMemoryAccess(now)
		_ = m.sessionStore.Save(sess)
	}
}

func summarizeLine(content string, maxLen int) string {
	trimmed := strings.TrimSpace(content)
	if len(trimmed) <= maxLen {
		return trimmed
	}
	if maxLen <= 3 {
		return trimmed[:maxLen]
	}
	return trimmed[:maxLen-3] + "..."
}

func extractKeywords(input string) []string {
	words := strings.Fields(strings.ToLower(strings.TrimSpace(input)))
	if len(words) == 0 {
		return nil
	}
	stopWords := map[string]struct{}{
		"the": {}, "and": {}, "for": {}, "with": {}, "that": {}, "this": {}, "from": {}, "into": {}, "what": {}, "how": {}, "why": {}, "are": {}, "you": {}, "is": {}, "was": {}, "were": {}, "have": {}, "has": {},
	}

	seen := make(map[string]struct{}, len(words))
	out := make([]string, 0, 8)
	for _, word := range words {
		cleaned := strings.Trim(word, ",.!?:;()[]{}\"'")
		if len(cleaned) < 3 {
			continue
		}
		if _, isStop := stopWords[cleaned]; isStop {
			continue
		}
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		out = append(out, cleaned)
		if len(out) >= 8 {
			break
		}
	}
	return out
}

func evolutionGroupKey(entry MemoryEntry) string {
	if sid, ok := entry.Metadata["session_id"].(string); ok && strings.TrimSpace(sid) != "" {
		return "session:" + strings.TrimSpace(sid)
	}
	keywords := extractKeywords(entry.Content)
	if len(keywords) > 0 {
		return "topic:" + keywords[0]
	}
	return "fallback"
}

func (m *MemoryManager) summarizeGroup(entries []MemoryEntry) string {
	messages := make([]llm.Message, 0, len(entries))
	for _, entry := range entries {
		messages = append(messages, llm.Message{Role: llm.RoleUser, Text: summarizeLine(entry.Content, 500)})
	}

	if m.summarizer != nil {
		summary, err := m.summarizer.Summarize(messages)
		if err == nil && strings.TrimSpace(summary) != "" {
			return strings.TrimSpace(summary)
		}
	}

	lines := make([]string, 0, minInt(len(entries), 5)+1)
	lines = append(lines, "# Memory Evolution Summary")
	for i, entry := range entries {
		if i >= 5 {
			break
		}
		lines = append(lines, "- "+summarizeLine(entry.Content, 180))
	}
	return strings.Join(lines, "\n")
}

func averageImportance(entries []MemoryEntry) float64 {
	if len(entries) == 0 {
		return 0
	}
	sum := 0.0
	for _, entry := range entries {
		importance := entry.Importance
		if importance <= 0 {
			importance = calculateImportance(entry)
		}
		sum += importance
	}
	return clamp01(sum / float64(len(entries)))
}

func sortedTagList(tagSet map[string]struct{}) []string {
	if len(tagSet) == 0 {
		return nil
	}
	out := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		out = append(out, tag)
	}
	sort.Strings(out)
	if len(out) > 12 {
		out = out[:12]
	}
	return out
}

func minInt(a, b int) int {
	if a <= b {
		return a
	}
	return b
}
