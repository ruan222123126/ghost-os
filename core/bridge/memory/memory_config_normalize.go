package memory

import (
	"path/filepath"
	"strings"
	"time"
)

func normalizeMemoryConfig(config MemoryConfig) MemoryConfig {
	out := applyLegacyMemoryConfig(config)
	explicitTemporalDisable := !config.Warm.TemporalDecayEnabled && config.Warm.TemporalDecayHalfLife < 0
	explicitAnchorDisable := !config.Warm.AnchorEnabled && config.Warm.AnchorMinWeight < 0
	if out.Warm.Capacity <= 0 {
		out.Warm.Capacity = defaultWarmCapacity
	}
	out.Ledger.Namespace = normalizeLedgerNamespace(out.Ledger.Namespace)
	out.Ledger.WorkspaceID = strings.TrimSpace(out.Ledger.WorkspaceID)
	out.Index.CheckpointDir = strings.TrimSpace(out.Index.CheckpointDir)
	out.Index.ShadowCompare = out.Index.ShadowCompare || out.Ledger.ShadowCompare
	if !out.Index.Enabled && out.Ledger.DualWrite && (out.Graph.Enabled || out.Decision.Enabled || out.Vector.Enabled || out.Warm.EvolutionEnabled) {
		out.Index.Enabled = true
	}
	if out.Recall.ShadowEnabled {
		out.Recall.IntentPlannerEnabled = true
		out.Vector.Enabled = true
	}
	if out.Recall.BucketReadEnabled {
		out.Recall.IntentPlannerEnabled = true
	}
	if out.Recall.BucketShadowCompare {
		out.Recall.BucketReadEnabled = true
	}
	out.Truth.Enabled = out.Truth.Enabled || out.Truth.DualWrite
	if out.Truth.Enabled && !out.Truth.ShadowFailOpen {
		out.Truth.ShadowFailOpen = true
	}
	if out.Vector.TopK <= 0 {
		out.Vector.TopK = defaultVectorTopK
	}
	if out.Truth.TopK <= 0 {
		out.Truth.TopK = defaultVectorTopK
	}
	if out.Truth.MinSupportRefs <= 0 {
		out.Truth.MinSupportRefs = 2
	}
	if out.Truth.SchemaVersion <= 0 {
		out.Truth.SchemaVersion = truthSchemaVersion
	}
	if out.Truth.SchemaVersion >= truthSchemaVersion && !config.Truth.ClaimArbitrationEnabled && !config.TruthClaimArbitrationEnabled {
		out.Truth.ClaimArbitrationEnabled = true
	}
	if out.Truth.ClaimArbitrationEnabled && !config.Truth.ClaimStatusProjectionEnabled && !config.TruthClaimStatusProjectionEnabled {
		out.Truth.ClaimStatusProjectionEnabled = true
	}
	if !config.Truth.LegacyObjectProjectionEnabled && !config.TruthLegacyObjectProjectionEnabled {
		out.Truth.LegacyObjectProjectionEnabled = true
	}
	if out.Recall.RecallInjectMinConfidence <= 0 {
		out.Recall.RecallInjectMinConfidence = 0.7
	}
	if out.Recall.ConflictPenalty <= 0 {
		out.Recall.ConflictPenalty = 0.18
	}
	if out.Recall.MaxBuckets <= 0 {
		out.Recall.MaxBuckets = 6
	}
	if out.Recall.MaxRecentBuckets <= 0 {
		out.Recall.MaxRecentBuckets = 2
	}
	if out.Recall.MaxHistoricalBuckets <= 0 {
		out.Recall.MaxHistoricalBuckets = 4
	}
	if out.Recall.BucketSessionBudget <= 0 {
		out.Recall.BucketSessionBudget = 1
	}
	if out.Recall.CompressionMinSupport <= 0 {
		out.Recall.CompressionMinSupport = 2
	}
	if len(out.Recall.CompressionAgeTiers) == 0 {
		out.Recall.CompressionAgeTiers = []CompressionAgeTier{
			{Name: "hot", MinAge: 0, MaxAge: 72 * time.Hour, PreferredLayers: []string{"raw", "warm", "markdown"}},
			{Name: "recent", MinAge: 72 * time.Hour, MaxAge: 30 * 24 * time.Hour, PreferredLayers: []string{"markdown", "decision", "graph"}},
			{Name: "historical", MinAge: 30 * 24 * time.Hour, MaxAge: 180 * 24 * time.Hour, PreferredLayers: []string{"decision", "graph", "markdown"}},
			{Name: "longterm", MinAge: 180 * 24 * time.Hour, PreferredLayers: []string{"decision", "graph", "markdown"}},
		}
	}
	if out.Vector.MinScore <= 0 {
		out.Vector.MinScore = defaultVectorMinScore
	}
	if !config.Hygiene.Enabled && !config.HygieneEnabled && strings.TrimSpace(out.Hygiene.Path) == "" {
		out.Hygiene.Enabled = true
	}
	if out.Warm.AutoRecallLimit <= 0 {
		out.Warm.AutoRecallLimit = defaultAutoRecallLimit
	}
	if out.Warm.TTL <= 0 {
		out.Warm.TTL = defaultWarmTTL
	}
	if out.Warm.TemporalDecayHalfLife <= 0 {
		out.Warm.TemporalDecayHalfLife = defaultTemporalDecayHalfLife
	}
	if out.Warm.AnchorMinWeight <= 0 {
		out.Warm.AnchorMinWeight = defaultAnchorMinWeight
	}
	if out.Warm.EvolutionInterval <= 0 {
		out.Warm.EvolutionInterval = defaultEvolutionInterval
	}
	if out.Warm.EvolutionBatchSize <= 0 {
		out.Warm.EvolutionBatchSize = defaultEvolutionBatchSize
	}
	if out.Index.Workers <= 0 {
		out.Index.Workers = 1
	}
	if out.Index.BatchSize <= 0 {
		out.Index.BatchSize = defaultEvolutionBatchSize
	}
	if out.Index.PollInterval <= 0 {
		out.Index.PollInterval = defaultIndexPollInterval
	}
	if out.Graph.MaxHops <= 0 {
		out.Graph.MaxHops = defaultGraphMaxHops
	}
	if out.Graph.MaxHits <= 0 {
		out.Graph.MaxHits = defaultGraphMaxHits
	}
	if out.Graph.MinConfidence <= 0 {
		out.Graph.MinConfidence = defaultGraphMinConfidence
	}
	if out.Decision.MaxHits <= 0 {
		out.Decision.MaxHits = defaultDecisionMaxHits
	}
	if !out.Decision.CaptureOnTurnSet {
		out.Decision.CaptureOnTurn = true
	}
	if out.Decision.MinConfidence <= 0 {
		out.Decision.MinConfidence = defaultDecisionMinConfidence
	}
	if out.Decision.MinReuseScore <= 0 {
		out.Decision.MinReuseScore = defaultDecisionMinReuseScore
	}
	if out.Decision.Recipe.Interval <= 0 {
		out.Decision.Recipe.Interval = defaultDecisionRecipeInterval
	}
	if out.Decision.Recipe.MinSupport <= 0 {
		out.Decision.Recipe.MinSupport = defaultDecisionRecipeMinSupport
	}
	if !config.Decision.Recipe.ReuseEnabledSet && out.Decision.Recipe.Enabled {
		out.Decision.Recipe.ReuseEnabled = true
	}
	if !config.Decision.Recipe.ExecutionTrackingEnabledSet && out.Decision.Recipe.Enabled {
		out.Decision.Recipe.ExecutionTrackingEnabled = true
	}
	if !config.Decision.Recipe.BackfillEnabledSet && out.Decision.Recipe.Enabled {
		out.Decision.Recipe.BackfillEnabled = true
	}
	if !config.Decision.Recipe.DefaultEnabledSet && out.Decision.Recipe.Enabled {
		out.Decision.Recipe.DefaultEnabled = true
	}
	if !out.Decision.Recipe.ReuseEnabled {
		out.Decision.Recipe.ExecutionTrackingEnabled = false
	}
	if out.Decision.Recipe.DefaultGrayPercent <= 0 {
		out.Decision.Recipe.DefaultGrayPercent = 10
	}
	if out.Decision.Recipe.DefaultGrayPercent > 100 {
		out.Decision.Recipe.DefaultGrayPercent = 100
	}
	if out.Decision.Recipe.MinSelectionConfidence <= 0 {
		out.Decision.Recipe.MinSelectionConfidence = 0.72
	}
	if out.Decision.Recipe.MinSuccessRate <= 0 {
		out.Decision.Recipe.MinSuccessRate = defaultRecipeMinSuccess
	}
	if out.Decision.Recipe.BackfillBatchSize <= 0 {
		out.Decision.Recipe.BackfillBatchSize = 50
	}
	if out.Decision.Recipe.BackfillInterval <= 0 {
		out.Decision.Recipe.BackfillInterval = 30 * time.Minute
	}
	out.Graph.Namespace = normalizeGraphNamespace(out.Graph.Namespace)
	if out.Graph.Path != "" {
		out.Graph.Path = filepath.Clean(out.Graph.Path)
	}
	if strings.TrimSpace(out.Ledger.BaseDir) == "" && strings.TrimSpace(out.Cold.BaseDir) != "" {
		out.Ledger.BaseDir = filepath.Join(out.Cold.BaseDir, "ledger")
	}
	if out.Ledger.BaseDir != "" {
		out.Ledger.BaseDir = filepath.Clean(out.Ledger.BaseDir)
	}
	if out.Index.Enabled && out.Index.CheckpointDir == "" && out.Ledger.BaseDir != "" {
		out.Index.CheckpointDir = filepath.Join(out.Ledger.BaseDir, "_index")
	}
	if out.Index.CheckpointDir != "" {
		out.Index.CheckpointDir = filepath.Clean(out.Index.CheckpointDir)
	}
	if out.Decision.Path != "" {
		out.Decision.Path = filepath.Clean(out.Decision.Path)
	}
	if out.Truth.Enabled && strings.TrimSpace(out.Truth.BaseDir) == "" {
		out.Truth.BaseDir = defaultTruthBaseDir(out.Cold.BaseDir)
	}
	if out.Truth.BaseDir != "" {
		out.Truth.BaseDir = filepath.Clean(out.Truth.BaseDir)
	}
	if out.Vector.Enabled && strings.TrimSpace(out.Vector.Path) == "" {
		out.Vector.Path = defaultVectorBaseDir(firstNonEmpty(out.Truth.BaseDir, out.Cold.BaseDir))
	}
	if out.Vector.Path != "" {
		out.Vector.Path = filepath.Clean(out.Vector.Path)
	}
	if out.Hygiene.Enabled && strings.TrimSpace(out.Hygiene.Path) == "" {
		out.Hygiene.Path = defaultHygieneBaseDir(firstNonEmpty(out.Cold.BaseDir, out.Truth.BaseDir))
	}
	if out.Hygiene.Path != "" {
		out.Hygiene.Path = filepath.Clean(out.Hygiene.Path)
	}
	if explicitTemporalDisable {
		out.Warm.TemporalDecayEnabled = false
	} else {
		out.Warm.TemporalDecayEnabled = true
	}
	if explicitAnchorDisable {
		out.Warm.AnchorEnabled = false
	} else {
		out.Warm.AnchorEnabled = true
	}
	if config.Warm.EvolutionUseWorker {
		out.Warm.EvolutionUseWorker = true
	} else {
		out.Warm.EvolutionUseWorker = hasSummarizer(out.Runtime.Summarizer)
	}
	if out.Cold.BaseDir != "" {
		out.Cold.BaseDir = filepath.Clean(out.Cold.BaseDir)
	}
	return out
}

func hasSummarizer(summarizer Summarizer) bool {
	return summarizer != nil
}

func defaultTruthBaseDir(coldBaseDir string) string {
	trimmed := strings.TrimSpace(coldBaseDir)
	if trimmed == "" {
		return ""
	}
	resolved := filepath.Clean(trimmed)
	parent := filepath.Dir(resolved)
	name := filepath.Base(resolved)
	if strings.TrimSpace(name) == "" || name == "." || name == string(filepath.Separator) {
		name = "cold"
	}
	return filepath.Join(parent, name+"-truth-shadow")
}

func defaultHygieneBaseDir(baseDir string) string {
	trimmed := strings.TrimSpace(baseDir)
	if trimmed == "" {
		return ""
	}
	resolved := filepath.Clean(trimmed)
	parent := filepath.Dir(resolved)
	if strings.TrimSpace(parent) == "" || parent == "." {
		return filepath.Join(resolved, "hygiene")
	}
	return filepath.Join(parent, "hygiene")
}
