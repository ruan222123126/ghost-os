package memory

import (
	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
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
	L1Hits                        uint64  `json:"l1_hits"`
	L2Hits                        uint64  `json:"l2_hits"`
	L3Hits                        uint64  `json:"l3_hits"`
	MarkdownHits                  uint64  `json:"markdown_hits"`
	GraphHits                     uint64  `json:"graph_hits"`
	EvolutionRuns                 uint64  `json:"evolution_runs"`
	NodesCreated                  uint64  `json:"nodes_created"`
	EntriesEvolved                uint64  `json:"entries_evolved"`
	PlannerRuns                   uint64  `json:"planner_runs"`
	PlannerErrors                 uint64  `json:"planner_errors"`
	TruthQueries                  uint64  `json:"truth_queries"`
	TruthHits                     uint64  `json:"truth_hits"`
	TruthVerifiedHits             uint64  `json:"truth_verified_hits"`
	TruthConflictedHits           uint64  `json:"truth_conflicted_hits"`
	HybridRerankRuns              uint64  `json:"hybrid_rerank_runs"`
	VectorPromotedHits            uint64  `json:"vector_promoted_hits"`
	AvgResultConfidence           float64 `json:"avg_result_confidence"`
	LowConfidenceFiltered         uint64  `json:"low_confidence_filtered"`
	ContextInjectionFiltered      uint64  `json:"context_injection_filtered"`
	VectorDocsIndexed             uint64  `json:"vector_docs_indexed"`
	VectorShadowHits              uint64  `json:"vector_shadow_hits"`
	ShadowOverlapRate             float64 `json:"shadow_overlap_rate"`
	ShadowOnlyCandidates          uint64  `json:"shadow_only_candidates"`
	ShadowLatencyMs               uint64  `json:"shadow_latency_ms"`
	ShadowWouldHelpRate           float64 `json:"shadow_would_help_rate"`
	TruthEventsWritten            uint64  `json:"truth_events_written"`
	LedgerEventsWritten           uint64  `json:"ledger_events_written"`
	LedgerShadowMismatchTotal     uint64  `json:"ledger_shadow_mismatch_total"`
	LedgerReplayLatencyMs         uint64  `json:"ledger_replay_latency_ms"`
	LedgerBackfillProgress        uint64  `json:"ledger_backfill_progress"`
	TruthObjectsUpserted          uint64  `json:"truth_objects_upserted"`
	TruthClaimsUpserted           uint64  `json:"truth_claims_upserted"`
	TruthErrors                   uint64  `json:"truth_errors"`
	TruthReplays                  uint64  `json:"truth_replays"`
	RecipeSelectedCount           uint64  `json:"recipe_selected_count"`
	RecipeAppliedCount            uint64  `json:"recipe_applied_count"`
	RecipeSuccessRate             float64 `json:"recipe_success_rate"`
	RecipePartialRate             float64 `json:"recipe_partial_rate"`
	RecipeFailureRate             float64 `json:"recipe_failure_rate"`
	RecipeHumanBlockedRate        float64 `json:"recipe_human_blocked_rate"`
	RecipeDeviationRate           float64 `json:"recipe_deviation_rate"`
	RecipeFallbackRate            float64 `json:"recipe_fallback_rate"`
	RecipeBackfillSessionsScanned uint64  `json:"recipe_backfill_sessions_scanned"`
	RecipeBackfillCreated         uint64  `json:"recipe_backfill_created"`
	RecipeBackfillUpdated         uint64  `json:"recipe_backfill_updated"`
	RecipeDefaultGrayHitRate      float64 `json:"recipe_default_gray_hit_rate"`
	BucketCandidatesTotal         uint64  `json:"bucket_candidates_total"`
	BucketSelectedTotal           uint64  `json:"bucket_selected_total"`
	BucketScannedTotal            uint64  `json:"bucket_scanned_total"`
	BucketScanLatencyMs           uint64  `json:"bucket_scan_latency_ms"`
	BucketRecallOnlyHits          uint64  `json:"bucket_recall_only_hits"`
	BucketShadowOverlapRate       float64 `json:"bucket_shadow_overlap_rate"`
	CompressionMarkdownRatio      float64 `json:"compression_markdown_ratio"`
	CompressionDecisionRatio      float64 `json:"compression_decision_ratio"`
	CompressionGraphRatio         float64 `json:"compression_graph_ratio"`
	ProvenanceMissingTotal        uint64  `json:"provenance_missing_total"`
	ViewStalenessMs               map[string]uint64 `json:"view_staleness_ms,omitempty"`
}

// MemoryManager 保留对外 façade，内部只装配各 memory 子能力。
type MemoryManager struct {
	warm        *WarmMemory
	cold        *ColdMemory
	graph       *GraphService
	decision    *DecisionService
	truth       *TruthWriter
	verifier    *TruthVerifier
	planner     *IntentPlanner
	vector      *VectorSidecar
	hygiene     *HygieneService
	truthReader *TruthReader

	query     *QueryService
	lifecycle *MemoryLifecycle
	evolver   *Evolver
	indexer   *IndexRuntime
	metrics   *memoryCounters
}
