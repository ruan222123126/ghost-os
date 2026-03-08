package memory

import (
	"log"
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

// CaptureDecisionTurn 在单轮完成后提取并持久化 decision memo。
func (m *MemoryManager) CaptureDecisionTurn(input DecisionCaptureInput) error {
	if m == nil || m.decision == nil {
		return nil
	}
	return m.decision.CaptureTurn(input)
}

// BuildContextWindow 根据当前输入自动注入可见的 recall context。
func (m *MemoryManager) BuildContextWindow(sessionID string, userInput string) ([]llm.Message, error) {
	return m.BuildContextWindowWithScope(SessionScope{SessionID: sessionID}, userInput)
}

// BuildContextWindowWithScope 使用显式 scope 生成 recall context。
func (m *MemoryManager) BuildContextWindowWithScope(scope SessionScope, userInput string) ([]llm.Message, error) {
	return m.query.BuildContextWindow(scope, userInput)
}

// BuildDecisionSelectorHintWithScope 构建 recipe/memo 选择提示。
func (m *MemoryManager) BuildDecisionSelectorHintWithScope(scope SessionScope, userInput string) (string, []DecisionHit, error) {
	if m == nil || m.decision == nil || !m.decision.Enabled() {
		return "", nil, nil
	}
	query := MemoryQuery{
		Limit:             max(defaultDecisionMaxHits, 6),
		SemanticQuery:     strings.TrimSpace(userInput),
		Keywords:          extractKeywords(userInput),
		IncludeDecision:   true,
		DecisionReuseOnly: true,
		DecisionTypes:     []string{DecisionHitTypeRecipe, DecisionHitTypeMemo, DecisionHitTypeWarning},
		EnvironmentStrict: scope.Environment != nil,
		PreferRecent:      true,
	}
	if scope.Environment != nil {
		env := cloneDecisionEnvFingerprint(*scope.Environment)
		query.Environment = &env
	}
	if sid := strings.TrimSpace(scope.SessionID); sid != "" {
		query.Metadata = map[string]any{"session_id": sid}
	}
	return m.decision.BuildSelectorHint(query, scope)
}

// QueryResult 返回 recall 的完整调试视图。
func (m *MemoryManager) QueryResult(query MemoryQuery) (MemoryQueryResult, error) {
	return m.QueryResultWithScope(query, SessionScope{})
}

// Query 仅返回排序后的条目列表。
func (m *MemoryManager) Query(query MemoryQuery) ([]MemoryEntry, error) {
	return m.QueryWithScope(query, SessionScope{})
}

// QueryWithScope 使用显式 scope 执行 recall。
func (m *MemoryManager) QueryWithScope(query MemoryQuery, scope SessionScope) ([]MemoryEntry, error) {
	result, err := m.QueryResultWithScope(query, scope)
	if err != nil {
		return nil, err
	}
	return result.Entries, nil
}

// QueryResultWithScope 返回带调试信息的 recall 结果。
func (m *MemoryManager) QueryResultWithScope(query MemoryQuery, scope SessionScope) (MemoryQueryResult, error) {
	return m.query.QueryResultWithScope(query, scope)
}

// PromoteToWarm 将完整会话投影到 warm 层。
func (m *MemoryManager) PromoteToWarm(sessionID string) error {
	return m.lifecycle.PromoteToWarm(sessionID)
}

// ArchiveToCold 将完整会话归档到 cold 层并触发 graph/decision sidecar。
func (m *MemoryManager) ArchiveToCold(sessionID string) error {
	return m.lifecycle.ArchiveToCold(sessionID)
}

// StoreWarmMessages 将新增会话消息写入 warm 层，供后续自动召回使用。
func (m *MemoryManager) StoreWarmMessages(sessionID string, startIndex int, messages []llm.Message) error {
	return m.lifecycle.StoreWarmMessages(sessionID, startIndex, messages)
}

// AppendLedgerTurn 在每轮持久化后追加原始 ledger 事件。
func (m *MemoryManager) AppendLedgerTurn(sessionID string, traceID string, startIndex int, messages []llm.Message) error {
	if m == nil || m.cold == nil {
		return nil
	}
	_, err := m.cold.AppendTurn(sessionID, "", traceID, startIndex, messages, time.Now().UTC())
	return err
}

// ReplayLedgerSession 回放单个 session 的 ledger 原始事件并投影为兼容视图。
func (m *MemoryManager) ReplayLedgerSession(sessionID string) (LedgerReplaySession, error) {
	if m == nil || m.cold == nil {
		return LedgerReplaySession{SessionID: strings.TrimSpace(sessionID)}, nil
	}
	return m.cold.ReplayLedgerSession(sessionID)
}

// BackfillLedger 将 legacy cold snapshot 回填到 ledger append-only 账本。
func (m *MemoryManager) BackfillLedger(opts LedgerBackfillOptions) (LedgerBackfillStats, error) {
	if m == nil || m.cold == nil {
		return LedgerBackfillStats{Namespace: normalizeLedgerNamespace(opts.Namespace), WorkspaceID: strings.TrimSpace(opts.WorkspaceID)}, nil
	}
	return m.cold.BackfillLedger(opts)
}

// LedgerStats 返回 cold ledger 的运行时开关与布局信息。
func (m *MemoryManager) LedgerStats() LedgerRuntimeStats {
	if m == nil || m.cold == nil {
		return LedgerRuntimeStats{}
	}
	return m.cold.LedgerRuntimeStats()
}

// Evolve 扫描 warm 层低重要度旧条目，沉淀为 markdown 记忆节点。
func (m *MemoryManager) Evolve() (EvolutionStats, error) {
	return m.evolver.Evolve()
}

// StartDreaming 启动后台演化协程。
func (m *MemoryManager) StartDreaming() {
	if m != nil && m.indexer != nil {
		m.indexer.Start()
	}
	if m != nil && m.decision != nil && m.decision.HasDistiller() {
		m.decision.StartDistiller()
	}
	if m != nil && m.evolver != nil {
		m.evolver.StartDreaming()
	}
}

// StopDreaming 停止后台演化协程，并等待其优雅退出。
func (m *MemoryManager) StopDreaming() {
	if m == nil {
		return
	}
	if m.decision != nil && m.decision.HasDistiller() {
		m.decision.StopDistiller()
	}
	if m.indexer != nil {
		m.indexer.Stop()
	}
	if m.evolver != nil {
		m.evolver.StopDreaming()
	}
}

// Metrics 返回当前内存管理器的指标快照。
func (m *MemoryManager) Metrics() MemoryMetrics {
	if m == nil || m.metrics == nil {
		return MemoryMetrics{}
	}
	return m.metrics.snapshot()
}

// ReplayTruth 从 event log 重建 object/claim snapshot。
func (m *MemoryManager) ReplayTruth() (TruthWriteResult, error) {
	if m == nil || m.truth == nil {
		return TruthWriteResult{}, nil
	}
	return m.truth.Replay()
}

// VerifyTruthReplay 比对 live dual-write 快照与 replay 结果是否一致。
func (m *MemoryManager) VerifyTruthReplay() (TruthVerifyResult, error) {
	if m == nil || m.verifier == nil {
		return TruthVerifyResult{Match: true}, nil
	}
	return m.verifier.Verify()
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

func (m *MemoryManager) BackfillMarkdownLineage() (MarkdownBackfillStats, error) {
	if m == nil || m.cold == nil {
		return MarkdownBackfillStats{}, nil
	}
	return m.cold.BackfillMarkdownLineage()
}

// GraphStats 返回 graph sidecar 的当前统计快照。
func (m *MemoryManager) GraphStats(namespace string) GraphStats {
	if m == nil || m.graph == nil {
		return GraphStats{Namespace: normalizeGraphNamespace(namespace)}
	}
	return m.graph.GraphStats(namespace)
}

// DecisionQuery 仅查询 decision sidecar，方便 debug 与运维动作复用。
func (m *MemoryManager) DecisionQuery(query MemoryQuery) ([]MemoryEntry, []DecisionHit, error) {
	return m.DecisionQueryWithScope(query, SessionScope{})
}

// DecisionQueryWithScope 允许调用方提供显式环境指纹，避免混入其他 memory layer。
func (m *MemoryManager) DecisionQueryWithScope(query MemoryQuery, scope SessionScope) ([]MemoryEntry, []DecisionHit, error) {
	if m == nil || m.decision == nil || !m.decision.Enabled() {
		return nil, nil, nil
	}
	return m.decision.Retrieve(query, scope)
}

// DecisionStats 返回 decision sidecar 的当前统计快照。
func (m *MemoryManager) DecisionStats(namespace string) DecisionStats {
	if m == nil || m.decision == nil {
		return DecisionStats{Namespace: normalizeDecisionNamespace(namespace)}
	}
	return m.decision.Stats(namespace)
}

// DistillDecisionRecipes 手动触发一次 recipe 蒸馏。
func (m *MemoryManager) DistillDecisionRecipes(namespace string) (DecisionDistillStats, error) {
	if m == nil || m.decision == nil || !m.decision.HasDistiller() {
		return DecisionDistillStats{Namespace: distillStatsNamespace(namespace)}, nil
	}
	return m.decision.DistillAll(namespace)
}

// RebuildDecision 从 cold archive + markdown nodes 重建 decision memo/recipe。
func (m *MemoryManager) RebuildDecision(opts DecisionRebuildOptions) (DecisionRebuildStats, error) {
	if m == nil || m.decision == nil {
		return DecisionRebuildStats{Namespace: normalizeDecisionNamespace(opts.Namespace)}, nil
	}
	stats, err := m.decision.Rebuild(opts)
	if err != nil {
		return stats, err
	}
	if !opts.DryRun && m.vector != nil && m.vector.Enabled() {
		if rebuildErr := m.vector.RebuildFromTruthSnapshot(); rebuildErr != nil && m.decision.DebugEnabled() {
			log.Printf("[MEMORY] decision rebuild vector refresh failed: %v", rebuildErr)
		}
	}
	return stats, nil
}

// RebuildGraph 从 cold archive + markdown nodes 重建图谱快照。
func (m *MemoryManager) RebuildGraph(opts GraphRebuildOptions) (GraphRebuildStats, error) {
	if m == nil || m.graph == nil {
		return GraphRebuildStats{Namespace: normalizeGraphNamespace(opts.Namespace)}, nil
	}
	return m.graph.Rebuild(opts)
}
