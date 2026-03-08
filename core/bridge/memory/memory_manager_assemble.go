package memory

import (
	"log"
	"strings"
)

func NewMemoryManager(config MemoryConfig) *MemoryManager {
	normalized := normalizeMemoryConfig(config)
	metrics := &memoryCounters{}
	truth := NewTruthWriter(normalized, metrics)
	truthMapper := NewTruthMapper()

	warm := NewWarmMemoryWithTTL(normalized.Warm.Capacity, normalized.Warm.Path, normalized.Warm.TTL)
	warm.SetScoringConfig(newMemoryScoringConfig(normalized))
	_ = warm.Load()
	cold := NewColdMemoryWithConfig(normalized.Cold.BaseDir, ColdMemoryConfig{
		LedgerBaseDir:       normalized.Ledger.BaseDir,
		LedgerNamespace:     normalized.Ledger.Namespace,
		LedgerWorkspaceID:   normalized.Ledger.WorkspaceID,
		LedgerDualWrite:     normalized.Ledger.DualWrite,
		LedgerReadEnabled:   normalized.Ledger.ReadEnabled,
		LedgerShadowCompare: normalized.Ledger.ShadowCompare,
		Metrics:             metrics,
	})
	cold.SetTruthShadow(truth, truthMapper)
	graph := NewGraphService(normalized, cold, normalized.Runtime.Summarizer)
	decision := NewDecisionService(normalized, cold)
	decision.SetMetrics(metrics)
	decision.SetTruthShadow(truth, truthMapper)
	planner := NewIntentPlanner(normalized.Recall.IntentPlannerEnabled, metrics)
	vector := NewVectorSidecar(normalized, truth, metrics)
	truthReader := NewTruthReader(normalized, truth, metrics)
	hygiene := NewHygieneService(normalized.Hygiene)
	if truth != nil {
		truth.SetObjectSidecar(vector)
		truth.AddObjectSidecar(truthReader)
	}
	manager := &MemoryManager{
		warm:        warm,
		cold:        cold,
		graph:       graph,
		decision:    decision,
		truth:       truth,
		verifier:    NewTruthVerifier(truth),
		planner:     planner,
		vector:      vector,
		hygiene:     hygiene,
		truthReader: truthReader,
		metrics:     metrics,
	}
	manager.query = NewQueryService(normalized, warm, cold, graph, decision, planner, vector, truthReader, metrics)
	manager.evolver = NewEvolver(normalized, warm, cold, graph, normalized.Runtime.Summarizer, metrics)
	manager.indexer = NewIndexRuntime(normalized, warm, cold, graph, decision, vector, manager.evolver, metrics)
	manager.lifecycle = NewMemoryLifecycle(normalized, warm, cold, graph, manager.indexer, normalized.Runtime.SessionStore)
	logMemoryAssembly(normalized, cold, truth, graph, decision, planner, vector, hygiene, truthReader, manager.indexer)

	if normalized.Warm.EvolutionEnabled || (manager.indexer != nil && manager.indexer.Enabled()) {
		manager.StartDreaming()
	} else if decision.Enabled() && normalized.Decision.Recipe.Enabled && decision.HasDistiller() {
		decision.StartDistiller()
	}
	return manager
}

func logMemoryAssembly(config MemoryConfig, cold *ColdMemory, truth *TruthWriter, graph *GraphService, decision *DecisionService, planner *IntentPlanner, vector *VectorSidecar, hygiene *HygieneService, truthReader *TruthReader, index *IndexRuntime) {
	if graph.Enabled() {
		log.Printf("[MEMORY] graph sidecar enabled: path=%s namespace=%s", config.Graph.Path, config.Graph.Namespace)
	}
	if decision.Enabled() {
		log.Printf("[MEMORY] decision sidecar enabled: path=%s", config.Decision.Path)
	}
	if planner != nil && planner.Enabled() {
		log.Printf("[MEMORY] intent planner enabled")
	}
	if vector != nil && vector.Enabled() {
		log.Printf("[MEMORY] vector sidecar enabled: path=%s", config.Vector.Path)
	}
	if hygiene != nil && hygiene.Enabled() {
		log.Printf("[MEMORY] hygiene sidecar enabled: path=%s", config.Hygiene.Path)
	}
	if truthReader != nil && truthReader.Enabled() {
		log.Printf("[MEMORY] truth live reader enabled: top_k=%d", config.Truth.TopK)
	}
	if stats := cold.LedgerRuntimeStats(); strings.TrimSpace(stats.BaseDir) != "" {
		log.Printf("[MEMORY] cold ledger enabled: base=%s dual_write=%t read_enabled=%t shadow_compare=%t", stats.BaseDir, stats.DualWrite, stats.ReadEnabled, stats.ShadowCompare)
	}
	if truth != nil && truth.Enabled() {
		log.Printf("[MEMORY] truth shadow enabled: base=%s dual_write=%t", truth.BaseDir(), truth.DualWriteEnabled())
	}
	if index != nil && index.Enabled() {
		log.Printf("[MEMORY] index runtime enabled: workers=%d batch_size=%d poll_interval=%s checkpoint_dir=%s shadow_compare=%t", config.Index.Workers, config.Index.BatchSize, config.Index.PollInterval, config.Index.CheckpointDir, config.Index.ShadowCompare)
	}
}
