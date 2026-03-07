package memory

import (
	"path/filepath"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
)

const (
	defaultAutoRecallLimit   = 5
	defaultEvolutionInterval = time.Hour
)

// Summarizer 为自动摘要能力提供统一接口。
type Summarizer interface {
	Summarize(messages []llm.Message) (string, error)
}

// AnchorExtractor 为可选的结构化锚点抽取能力提供统一接口。
type AnchorExtractor interface {
	ExtractAnchors(messages []llm.Message) ([]MemoryAnchor, error)
}

// MemoryConfig 定义三层记忆管理器初始化参数。
type MemoryConfig struct {
	WarmCapacity int
	WarmPath     string
	ColdBaseDir  string

	AutoRecallEnabled     bool
	AutoRecallLimit       int
	WarmTTL               time.Duration
	TemporalDecayEnabled  bool
	TemporalDecayHalfLife time.Duration
	AnchorEnabled         bool
	AnchorMinWeight       float64
	EvolutionInterval     time.Duration
	EvolutionEnabled      bool
	EvolutionUseWorker    bool
	EvolutionBatchSize    int

	// 运行态绑定依赖。
	SessionStore SessionStorePort
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

// MemoryManager 保留对外 façade，内部通过 query/lifecycle/evolver 组合职责。
type MemoryManager struct {
	warm *WarmMemory
	cold *ColdMemory

	query     *QueryService
	lifecycle *MemoryLifecycle
	evolver   *Evolver
	metrics   *memoryCounters
}

func NewMemoryManager(config MemoryConfig) *MemoryManager {
	normalized := normalizeMemoryConfig(config)

	warm := NewWarmMemoryWithTTL(normalized.WarmCapacity, normalized.WarmPath, normalized.WarmTTL)
	warm.SetScoringConfig(newMemoryScoringConfig(normalized))
	_ = warm.Load()
	cold := NewColdMemory(normalized.ColdBaseDir)
	metrics := &memoryCounters{}

	manager := &MemoryManager{
		warm:    warm,
		cold:    cold,
		metrics: metrics,
	}
	manager.query = NewQueryService(normalized, warm, cold, metrics)
	manager.lifecycle = NewMemoryLifecycle(normalized, warm, cold, normalized.SessionStore)
	manager.evolver = NewEvolver(normalized, warm, cold, normalized.Summarizer, metrics)

	if normalized.EvolutionEnabled {
		manager.StartDreaming()
	}
	return manager
}

func normalizeMemoryConfig(config MemoryConfig) MemoryConfig {
	out := config
	explicitTemporalDisable := !config.TemporalDecayEnabled && config.TemporalDecayHalfLife < 0
	explicitAnchorDisable := !config.AnchorEnabled && config.AnchorMinWeight < 0
	if out.WarmCapacity <= 0 {
		out.WarmCapacity = defaultWarmCapacity
	}
	if out.AutoRecallLimit <= 0 {
		out.AutoRecallLimit = defaultAutoRecallLimit
	}
	if out.WarmTTL <= 0 {
		out.WarmTTL = defaultWarmTTL
	}
	if out.TemporalDecayHalfLife <= 0 {
		out.TemporalDecayHalfLife = defaultTemporalDecayHalfLife
	}
	if out.AnchorMinWeight <= 0 {
		out.AnchorMinWeight = defaultAnchorMinWeight
	}
	if out.EvolutionInterval <= 0 {
		out.EvolutionInterval = defaultEvolutionInterval
	}
	if out.EvolutionBatchSize <= 0 {
		out.EvolutionBatchSize = defaultEvolutionBatchSize
	}
	if explicitTemporalDisable {
		out.TemporalDecayEnabled = false
	} else {
		out.TemporalDecayEnabled = true
	}
	if explicitAnchorDisable {
		out.AnchorEnabled = false
	} else {
		out.AnchorEnabled = true
	}
	if config.EvolutionUseWorker {
		out.EvolutionUseWorker = true
	} else {
		out.EvolutionUseWorker = hasSummarizer(config.Summarizer)
	}
	if out.ColdBaseDir != "" {
		out.ColdBaseDir = filepath.Clean(out.ColdBaseDir)
	}
	return out
}

func hasSummarizer(summarizer Summarizer) bool {
	return summarizer != nil
}

// BuildContextWindow 从 warm 层构建自动召回上下文，供 Agent 在当前轮次注入。
func (m *MemoryManager) BuildContextWindow(sessionID string, userInput string) ([]llm.Message, error) {
	return m.BuildContextWindowWithScope(SessionScope{SessionID: sessionID}, userInput)
}

// BuildContextWindowWithScope 允许调用方提供当前可见历史，用于避免重复注入已在上下文中的内容。
func (m *MemoryManager) BuildContextWindowWithScope(scope SessionScope, userInput string) ([]llm.Message, error) {
	return m.query.BuildContextWindow(scope, userInput)
}

// Query 按 L1 -> L2 -> L3 -> Markdown 顺序级联检索。
func (m *MemoryManager) Query(query MemoryQuery) ([]MemoryEntry, error) {
	return m.query.Query(query)
}

// QueryWithScope 在统一级联检索中显式接收请求级热态上下文，避免共享状态串味。
func (m *MemoryManager) QueryWithScope(query MemoryQuery, scope SessionScope) ([]MemoryEntry, error) {
	return m.query.QueryWithScope(query, scope)
}

// PromoteToWarm 手动把冷数据会话提升到温数据层。
func (m *MemoryManager) PromoteToWarm(sessionID string) error {
	return m.lifecycle.PromoteToWarm(sessionID)
}

// ArchiveToCold 把当前会话写入冷数据层。
func (m *MemoryManager) ArchiveToCold(sessionID string) error {
	return m.lifecycle.ArchiveToCold(sessionID)
}

// StoreWarmMessages 将新增会话消息写入 warm 层，供后续自动召回使用。
func (m *MemoryManager) StoreWarmMessages(sessionID string, startIndex int, messages []llm.Message) error {
	return m.lifecycle.StoreWarmMessages(sessionID, startIndex, messages)
}

// Evolve 扫描 warm 层低重要度旧条目，沉淀为 markdown 记忆节点。
func (m *MemoryManager) Evolve() (EvolutionStats, error) {
	return m.evolver.Evolve()
}

// StartDreaming 启动后台演化协程。
func (m *MemoryManager) StartDreaming() {
	m.evolver.StartDreaming()
}

// StopDreaming 停止后台演化协程，并等待其优雅退出。
func (m *MemoryManager) StopDreaming() {
	if m == nil {
		return
	}
	m.evolver.StopDreaming()
}

// Metrics 返回当前内存管理器的指标快照。
func (m *MemoryManager) Metrics() MemoryMetrics {
	if m == nil || m.metrics == nil {
		return MemoryMetrics{}
	}
	return m.metrics.snapshot()
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
