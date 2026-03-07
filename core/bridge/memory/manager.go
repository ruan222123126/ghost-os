package memory

import (
	"log"
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

// GraphFactExtractor 为可选的 graph 结构化抽取提供统一接口。
type GraphFactExtractor interface {
	ExtractGraphFacts(messages []llm.Message) ([]GraphFact, error)
}

// DecisionMemoExtractor 为可选的 decision 摘要补全能力提供统一接口。
type DecisionMemoExtractor interface {
	ExtractDecisionMemo(input DecisionCaptureInput) (DecisionMemo, error)
}

// MemoryConfig 定义三层记忆管理器初始化参数。
type MemoryConfig struct {
	WarmCapacity int
	WarmPath     string
	ColdBaseDir  string

	AutoRecallEnabled        bool
	AutoRecallLimit          int
	WarmTTL                  time.Duration
	TemporalDecayEnabled     bool
	TemporalDecayHalfLife    time.Duration
	AnchorEnabled            bool
	AnchorMinWeight          float64
	EvolutionInterval        time.Duration
	EvolutionEnabled         bool
	EvolutionUseWorker       bool
	EvolutionBatchSize       int
	GraphEnabled             bool
	GraphPath                string
	GraphExtractOnArchive    bool
	GraphExtractOnEvolve     bool
	GraphMaxHops             int
	GraphMaxHits             int
	GraphMinConfidence       float64
	GraphNamespace           string
	GraphDebugEnabled        bool
	DecisionEnabled          bool
	DecisionCaptureOnTurn    bool
	DecisionCaptureOnTurnSet bool
	DecisionPath             string
	DecisionMaxHits          int
	DecisionMinConfidence    float64
	DecisionMinReuseScore    float64
	DecisionRecipeEnabled    bool
	DecisionRecipeInterval   time.Duration
	DecisionRecipeMinSupport int
	DecisionDebugEnabled     bool

	// 运行态绑定依赖。
	SessionStore SessionStorePort
	Summarizer   Summarizer
}

// SessionScope 显式描述一次请求可见的会话热态上下文，避免共享管理器持有可变请求态。
type SessionScope struct {
	SessionID   string
	History     *agent.History
	Environment *DecisionEnvFingerprint
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
	GraphHits      uint64 `json:"graph_hits"`
	EvolutionRuns  uint64 `json:"evolution_runs"`
	NodesCreated   uint64 `json:"nodes_created"`
	EntriesEvolved uint64 `json:"entries_evolved"`
}

// MemoryManager 保留对外 façade，内部通过 query/lifecycle/evolver 组合职责。
type MemoryManager struct {
	warm     *WarmMemory
	cold     *ColdMemory
	graph    *GraphService
	decision *DecisionService

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
	graph := NewGraphService(normalized, cold, normalized.Summarizer)
	decision := NewDecisionService(normalized)
	if graph.Enabled() {
		log.Printf("[MEMORY] graph sidecar enabled: path=%s namespace=%s", normalized.GraphPath, normalized.GraphNamespace)
	}
	if decision.Enabled() {
		log.Printf("[MEMORY] decision sidecar enabled: path=%s", normalized.DecisionPath)
	}
	metrics := &memoryCounters{}

	manager := &MemoryManager{
		warm:     warm,
		cold:     cold,
		graph:    graph,
		decision: decision,
		metrics:  metrics,
	}
	manager.query = NewQueryService(normalized, warm, cold, graph, decision, metrics)
	manager.lifecycle = NewMemoryLifecycle(normalized, warm, cold, graph, normalized.SessionStore)
	manager.evolver = NewEvolver(normalized, warm, cold, graph, normalized.Summarizer, metrics)

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
	if out.GraphMaxHops <= 0 {
		out.GraphMaxHops = defaultGraphMaxHops
	}
	if out.GraphMaxHits <= 0 {
		out.GraphMaxHits = defaultGraphMaxHits
	}
	if out.GraphMinConfidence <= 0 {
		out.GraphMinConfidence = defaultGraphMinConfidence
	}
	if out.DecisionMaxHits <= 0 {
		out.DecisionMaxHits = defaultDecisionMaxHits
	}
	if !out.DecisionCaptureOnTurnSet {
		out.DecisionCaptureOnTurn = true
	}
	if out.DecisionMinConfidence <= 0 {
		out.DecisionMinConfidence = defaultDecisionMinConfidence
	}
	if out.DecisionMinReuseScore <= 0 {
		out.DecisionMinReuseScore = defaultDecisionMinReuseScore
	}
	if out.DecisionRecipeInterval <= 0 {
		out.DecisionRecipeInterval = defaultDecisionRecipeInterval
	}
	if out.DecisionRecipeMinSupport <= 0 {
		out.DecisionRecipeMinSupport = defaultDecisionRecipeMinSupport
	}
	out.GraphNamespace = normalizeGraphNamespace(out.GraphNamespace)
	if out.GraphPath != "" {
		out.GraphPath = filepath.Clean(out.GraphPath)
	}
	if out.DecisionPath != "" {
		out.DecisionPath = filepath.Clean(out.DecisionPath)
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

// CaptureDecisionTurn 在单轮完成后提取并持久化 decision memo。
func (m *MemoryManager) CaptureDecisionTurn(input DecisionCaptureInput) error {
	if m == nil || m.decision == nil || !m.decision.Enabled() {
		return nil
	}
	if !m.decision.captureOnTurn {
		return nil
	}
	return m.decision.CaptureTurn(input)
}

// BuildContextWindow 从 warm 层构建自动召回上下文，供 Agent 在当前轮次注入。
func (m *MemoryManager) BuildContextWindow(sessionID string, userInput string) ([]llm.Message, error) {
	return m.BuildContextWindowWithScope(SessionScope{SessionID: sessionID}, userInput)
}

// BuildContextWindowWithScope 允许调用方提供当前可见历史，用于避免重复注入已在上下文中的内容。
func (m *MemoryManager) BuildContextWindowWithScope(scope SessionScope, userInput string) ([]llm.Message, error) {
	return m.query.BuildContextWindow(scope, userInput)
}

// QueryResult 返回 entries 与可选 graph debug hits。
func (m *MemoryManager) QueryResult(query MemoryQuery) (MemoryQueryResult, error) {
	return m.query.QueryResult(query)
}

// Query 按 L1 -> L2 -> L3 -> Markdown 顺序级联检索。
func (m *MemoryManager) Query(query MemoryQuery) ([]MemoryEntry, error) {
	return m.query.Query(query)
}

// QueryWithScope 在统一级联检索中显式接收请求级热态上下文，避免共享状态串味。
func (m *MemoryManager) QueryWithScope(query MemoryQuery, scope SessionScope) ([]MemoryEntry, error) {
	return m.query.QueryWithScope(query, scope)
}

// QueryResultWithScope 在返回 entries 的同时保留 graph/decision 调试命中信息。
func (m *MemoryManager) QueryResultWithScope(query MemoryQuery, scope SessionScope) (MemoryQueryResult, error) {
	return m.query.QueryResultWithScope(query, scope)
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

// GraphStats 返回 graph sidecar 的当前统计快照。
func (m *MemoryManager) GraphStats(namespace string) GraphStats {
	if m == nil || m.graph == nil {
		return GraphStats{Namespace: normalizeGraphNamespace(namespace)}
	}
	return m.graph.GraphStats(namespace)
}

// RebuildGraph 从 cold archive + markdown nodes 重建图谱快照。
func (m *MemoryManager) RebuildGraph(opts GraphRebuildOptions) (GraphRebuildStats, error) {
	if m == nil || m.graph == nil {
		return GraphRebuildStats{Namespace: normalizeGraphNamespace(opts.Namespace)}, nil
	}
	return m.graph.Rebuild(opts)
}
