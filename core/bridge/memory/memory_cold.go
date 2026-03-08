package memory

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ghost-os/bridge/llm"
)

// ColdArchive 暴露冷存归档快照，供 graph backfill 等后台任务复用。
type ColdArchive struct {
	SessionID  string        `json:"session_id"`
	ArchivedAt time.Time     `json:"archived_at"`
	Messages   []llm.Message `json:"messages"`
}

type ColdReadMode string

const (
	coldReadModeLegacy ColdReadMode = "legacy"
	coldReadModeLedger ColdReadMode = "ledger"
)

type ColdMemoryConfig struct {
	LedgerBaseDir       string
	LedgerNamespace     string
	LedgerWorkspaceID   string
	LedgerDualWrite     bool
	LedgerReadEnabled   bool
	LedgerShadowCompare bool
	Metrics             *memoryCounters
}

// ColdMemory 是 L3 冷数据层 façade，内部兼容 legacy snapshot 与 raw ledger 双后端。
type ColdMemory struct {
	legacy          *LegacyColdStore
	ledger          *LedgerStore
	markdown        *MarkdownStore
	truth           *TruthWriter
	truthMapper     *TruthMapper
	ledgerDualWrite bool
	readMode        ColdReadMode
	shadowCompare   bool
	metrics         *memoryCounters
	mu              sync.RWMutex
}

func NewColdMemory(baseDir string) *ColdMemory {
	return NewColdMemoryWithConfig(baseDir, ColdMemoryConfig{})
}

func NewColdMemoryWithConfig(baseDir string, cfg ColdMemoryConfig) *ColdMemory {
	resolved := resolveMemoryPath(baseDir)
	markdownDir := ""
	if resolved != "" {
		markdownDir = filepath.Join(resolved, "markdown", "nodes")
	}
	ledgerBaseDir := strings.TrimSpace(cfg.LedgerBaseDir)
	if ledgerBaseDir == "" && resolved != "" {
		ledgerBaseDir = filepath.Join(resolved, "ledger")
	}
	readMode := coldReadModeLegacy
	if cfg.LedgerReadEnabled {
		readMode = coldReadModeLedger
	}
	return &ColdMemory{
		legacy:          NewLegacyColdStore(resolved),
		ledger:          NewLedgerStore(ledgerBaseDir, cfg.LedgerNamespace, cfg.LedgerWorkspaceID, cfg.Metrics),
		markdown:        NewMarkdownStoreWithConfig(markdownDir, cfg.LedgerNamespace, cfg.LedgerWorkspaceID),
		ledgerDualWrite: cfg.LedgerDualWrite,
		readMode:        readMode,
		shadowCompare:   cfg.LedgerShadowCompare,
		metrics:         cfg.Metrics,
	}
}

func (c *ColdMemory) SetTruthShadow(writer *TruthWriter, mapper *TruthMapper) {
	if c == nil {
		return
	}
	c.truth = writer
	c.truthMapper = mapper
	if c.legacy != nil {
		c.legacy.SetTruthShadow(writer, mapper)
	}
}

func (c *ColdMemory) LedgerRuntimeStats() LedgerRuntimeStats {
	if c == nil {
		return LedgerRuntimeStats{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	stats := LedgerRuntimeStats{
		DualWrite:     c.ledgerDualWrite,
		ReadEnabled:   c.readMode == coldReadModeLedger,
		ShadowCompare: c.shadowCompare,
	}
	if c.ledger != nil {
		stats.BaseDir = c.ledger.BaseDir()
		stats.Namespace = c.ledger.Namespace()
		stats.WorkspaceID = c.ledger.WorkspaceID()
	}
	return stats
}

func (c *ColdMemory) ReplayLedgerSession(sessionID string) (LedgerReplaySession, error) {
	if c == nil || c.ledger == nil {
		return LedgerReplaySession{SessionID: strings.TrimSpace(sessionID)}, nil
	}
	return c.ledger.ReplaySession(sessionID)
}

func (c *ColdMemory) AppendTurn(sessionID string, turnID string, traceID string, startIndex int, messages []llm.Message, occurredAt time.Time) (LedgerAppendResult, error) {
	if c == nil || c.ledger == nil {
		return LedgerAppendResult{}, nil
	}
	c.mu.RLock()
	enabled := c.ledgerDualWrite
	c.mu.RUnlock()
	if !enabled {
		return LedgerAppendResult{}, nil
	}
	return c.ledger.AppendTurn(sessionID, turnID, traceID, startIndex, messages, occurredAt)
}

func (c *ColdMemory) BackfillLedger(opts LedgerBackfillOptions) (LedgerBackfillStats, error) {
	if c == nil || c.ledger == nil {
		return LedgerBackfillStats{}, nil
	}
	return c.ledger.BackfillLegacy(c.legacy, opts)
}

// Archive 继续写 legacy cold snapshot，供兼容与回滚使用。
func (c *ColdMemory) Archive(sessionID string, messages []llm.Message) error {
	if c == nil || c.legacy == nil {
		return nil
	}
	return c.legacy.Archive(sessionID, messages)
}

// Retrieve 默认保持 legacy 语义，读切换后由 ledger replay 输出兼容视图。
func (c *ColdMemory) Retrieve(query MemoryQuery) ([]MemoryEntry, error) {
	if c == nil {
		return nil, nil
	}
	start := time.Now()
	primaryMode, shadowEnabled := c.readSettings()
	primary, err := c.retrieveWithMode(primaryMode, query)
	if err != nil {
		return nil, err
	}
	if primaryMode == coldReadModeLedger {
		c.recordLedgerReplayLatency(time.Since(start))
	}
	if shadowEnabled {
		shadowMode := coldReadModeLegacy
		if primaryMode == coldReadModeLegacy {
			shadowMode = coldReadModeLedger
		}
		shadow, shadowErr := c.retrieveWithMode(shadowMode, query)
		if shadowErr == nil {
			c.compareEntries(primary, shadow)
			if shadowMode == coldReadModeLedger {
				c.recordLedgerReplayLatency(time.Since(start))
			}
		}
	}
	return primary, nil
}

func (c *ColdMemory) RetrieveWithBucketPlan(query MemoryQuery, plan *BucketPlan) ([]MemoryEntry, error) {
	if c == nil {
		return nil, nil
	}
	if plan == nil || len(plan.SelectedBuckets) == 0 {
		return c.Retrieve(query)
	}
	start := time.Now()
	primaryMode, shadowEnabled := c.readSettings()
	primary, err := c.retrieveWithModeAndPlan(primaryMode, query, plan)
	if err != nil {
		return nil, err
	}
	if primaryMode == coldReadModeLedger {
		c.recordLedgerReplayLatency(time.Since(start))
	}
	if shadowEnabled {
		shadowMode := coldReadModeLegacy
		if primaryMode == coldReadModeLegacy {
			shadowMode = coldReadModeLedger
		}
		shadow, shadowErr := c.retrieveWithMode(shadowMode, query)
		if shadowErr == nil {
			c.compareEntries(primary, shadow)
		}
	}
	return primary, nil
}

// SaveMarkdownNode 将演化后的记忆节点写入 markdown 冷存目录。
func (c *ColdMemory) SaveMarkdownNode(node MarkdownNode) error {
	if c == nil || c.markdown == nil {
		return fmt.Errorf("markdown store is not configured")
	}
	normalized := normalizeMarkdownNode(node)
	if err := c.markdown.Save(normalized); err != nil {
		return err
	}
	if c.truth != nil && c.truth.DualWriteEnabled() && c.truthMapper != nil {
		object := c.truthMapper.MapMarkdownNode(normalized)
		if err := writeTruthObjectShadow(c.truth, truthEventTypeMarkdownNode, object, ""); err != nil {
			return handleTruthShadowWriteError(c.truth, "", object.ObjectID, err)
		}
	}
	return nil
}

// LoadMarkdownNode 从 markdown 冷存目录读取节点。
func (c *ColdMemory) LoadMarkdownNode(id string) (MarkdownNode, error) {
	if c == nil || c.markdown == nil {
		return MarkdownNode{}, fmt.Errorf("markdown store is not configured")
	}
	return c.markdown.Load(id)
}

// ListMarkdownNodes 列出所有 markdown 节点 ID。
func (c *ColdMemory) ListMarkdownNodes() ([]string, error) {
	if c == nil || c.markdown == nil {
		return nil, nil
	}
	return c.markdown.List()
}

// ListSessions 返回时间范围内存在归档的会话 ID。
func (c *ColdMemory) ListSessions(timeRange TimeRange) ([]string, error) {
	if c == nil {
		return nil, nil
	}
	start := time.Now()
	primaryMode, shadowEnabled := c.readSettings()
	primary, err := c.listSessionsWithMode(primaryMode, timeRange)
	if err != nil {
		return nil, err
	}
	if primaryMode == coldReadModeLedger {
		c.recordLedgerReplayLatency(time.Since(start))
	}
	if shadowEnabled {
		shadowMode := coldReadModeLegacy
		if primaryMode == coldReadModeLegacy {
			shadowMode = coldReadModeLedger
		}
		shadow, shadowErr := c.listSessionsWithMode(shadowMode, timeRange)
		if shadowErr == nil {
			c.compareSessionIDs(primary, shadow)
			if shadowMode == coldReadModeLedger {
				c.recordLedgerReplayLatency(time.Since(start))
			}
		}
	}
	return primary, nil
}

// ListArchives 返回时间窗内的归档会话快照，便于 graph 等 sidecar 做重建。
func (c *ColdMemory) ListArchives(timeRange *TimeRange) ([]ColdArchive, error) {
	if c == nil {
		return nil, nil
	}
	start := time.Now()
	primaryMode, shadowEnabled := c.readSettings()
	primary, err := c.listArchivesWithMode(primaryMode, timeRange)
	if err != nil {
		return nil, err
	}
	if primaryMode == coldReadModeLedger {
		c.recordLedgerReplayLatency(time.Since(start))
	}
	if shadowEnabled {
		shadowMode := coldReadModeLegacy
		if primaryMode == coldReadModeLegacy {
			shadowMode = coldReadModeLedger
		}
		shadow, shadowErr := c.listArchivesWithMode(shadowMode, timeRange)
		if shadowErr == nil {
			c.compareArchives(primary, shadow)
			if shadowMode == coldReadModeLedger {
				c.recordLedgerReplayLatency(time.Since(start))
			}
		}
	}
	return primary, nil
}

func (c *ColdMemory) ListArchivesWithBucketPlan(timeRange *TimeRange, plan *BucketPlan) ([]ColdArchive, error) {
	if c == nil {
		return nil, nil
	}
	if plan == nil || len(plan.SelectedBuckets) == 0 {
		return c.ListArchives(timeRange)
	}
	primaryMode, _ := c.readSettings()
	return c.listArchivesWithModeAndPlan(primaryMode, timeRange, plan)
}

func (c *ColdMemory) readSettings() (ColdReadMode, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.readMode, c.shadowCompare
}

func (c *ColdMemory) retrieveWithMode(mode ColdReadMode, query MemoryQuery) ([]MemoryEntry, error) {
	if mode == coldReadModeLedger && c.ledger != nil {
		return c.ledger.RetrieveCompat(query)
	}
	if c.legacy == nil {
		return nil, nil
	}
	return c.legacy.Retrieve(query)
}

func (c *ColdMemory) retrieveWithModeAndPlan(mode ColdReadMode, query MemoryQuery, plan *BucketPlan) ([]MemoryEntry, error) {
	if mode == coldReadModeLedger && c.ledger != nil {
		return c.ledger.RetrieveCompatWithPlan(query, plan)
	}
	if c.legacy == nil {
		return nil, nil
	}
	return c.legacy.Retrieve(query)
}

func (c *ColdMemory) listSessionsWithMode(mode ColdReadMode, timeRange TimeRange) ([]string, error) {
	if mode == coldReadModeLedger && c.ledger != nil {
		return c.ledger.ListSessionsCompat(timeRange)
	}
	if c.legacy == nil {
		return nil, nil
	}
	return c.legacy.ListSessions(timeRange)
}

func (c *ColdMemory) listArchivesWithMode(mode ColdReadMode, timeRange *TimeRange) ([]ColdArchive, error) {
	if mode == coldReadModeLedger && c.ledger != nil {
		return c.ledger.ReplayArchivesCompat(timeRange)
	}
	if c.legacy == nil {
		return nil, nil
	}
	return c.legacy.ListArchives(timeRange)
}

func (c *ColdMemory) listArchivesWithModeAndPlan(mode ColdReadMode, timeRange *TimeRange, plan *BucketPlan) ([]ColdArchive, error) {
	if mode == coldReadModeLedger && c.ledger != nil {
		return c.ledger.ReplayArchivesCompatWithPlan(timeRange, plan)
	}
	if c.legacy == nil {
		return nil, nil
	}
	return c.legacy.ListArchives(timeRange)
}

func (c *ColdMemory) compareEntries(primary []MemoryEntry, shadow []MemoryEntry) {
	if len(primary) != len(shadow) {
		c.recordLedgerMismatch()
		return
	}
	for index := range primary {
		left := primary[index]
		right := shadow[index]
		if left.ID != right.ID || strings.TrimSpace(left.Content) != strings.TrimSpace(right.Content) {
			c.recordLedgerMismatch()
			return
		}
		if left.Timestamp.UTC() != right.Timestamp.UTC() {
			c.recordLedgerMismatch()
			return
		}
		if fingerprintLedgerEntry(left) != fingerprintLedgerEntry(right) {
			c.recordLedgerMismatch()
			return
		}
	}
}

func (c *ColdMemory) compareSessionIDs(primary []string, shadow []string) {
	if len(primary) != len(shadow) {
		c.recordLedgerMismatch()
		return
	}
	for index := range primary {
		if strings.TrimSpace(primary[index]) != strings.TrimSpace(shadow[index]) {
			c.recordLedgerMismatch()
			return
		}
	}
}

func (c *ColdMemory) compareArchives(primary []ColdArchive, shadow []ColdArchive) {
	if len(primary) != len(shadow) {
		c.recordLedgerMismatch()
		return
	}
	for index := range primary {
		left := primary[index]
		right := shadow[index]
		if left.SessionID != right.SessionID || !left.ArchivedAt.UTC().Equal(right.ArchivedAt.UTC()) || len(left.Messages) != len(right.Messages) {
			c.recordLedgerMismatch()
			return
		}
		for messageIndex := range left.Messages {
			if fingerprintLedgerMessage(left.Messages[messageIndex]) != fingerprintLedgerMessage(right.Messages[messageIndex]) {
				c.recordLedgerMismatch()
				return
			}
		}
	}
}

func (c *ColdMemory) recordLedgerReplayLatency(duration time.Duration) {
	if c == nil || c.metrics == nil {
		return
	}
	c.metrics.ledgerReplayQueries.Add(1)
	c.metrics.ledgerReplayLatencyMs.Add(uint64(duration.Milliseconds()))
}

func (c *ColdMemory) recordLedgerMismatch() {
	if c == nil || c.metrics == nil {
		return
	}
	c.metrics.ledgerShadowMismatchTotal.Add(1)
}

func fingerprintLedgerEntry(entry MemoryEntry) string {
	parts := []string{
		strings.TrimSpace(entry.ID),
		strings.TrimSpace(entry.Content),
		strings.TrimSpace(entry.Summary),
		fmt.Sprint(entry.Metadata["session_id"]),
		fmt.Sprint(entry.Metadata["role"]),
		entry.Timestamp.UTC().Format(time.RFC3339Nano),
	}
	return hashLedgerFingerprint(parts...)
}

func fingerprintLedgerMessage(message llm.Message) string {
	payload, _ := json.Marshal(llm.CloneMessages([]llm.Message{message}))
	return hashLedgerFingerprint(string(payload))
}

func hashLedgerFingerprint(parts ...string) string {
	hash := sha1.Sum([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(hash[:])
}

func messageToContent(msg llm.Message) string {
	if text := strings.TrimSpace(msg.Text); text != "" {
		return text
	}
	if len(msg.ToolCalls) == 0 {
		return ""
	}
	encoded, err := json.Marshal(msg.ToolCalls)
	if err != nil {
		return ""
	}
	return string(encoded)
}
