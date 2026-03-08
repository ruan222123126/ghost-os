package memory

import "strings"

func applyLegacyMemoryConfig(config MemoryConfig) MemoryConfig {
	out := config
	if out.Warm.Capacity == 0 && config.WarmCapacity != 0 {
		out.Warm.Capacity = config.WarmCapacity
	}
	if strings.TrimSpace(out.Warm.Path) == "" && strings.TrimSpace(config.WarmPath) != "" {
		out.Warm.Path = config.WarmPath
	}
	if strings.TrimSpace(out.Cold.BaseDir) == "" && strings.TrimSpace(config.ColdBaseDir) != "" {
		out.Cold.BaseDir = config.ColdBaseDir
	}
	if strings.TrimSpace(out.Ledger.BaseDir) == "" && strings.TrimSpace(config.LedgerBaseDir) != "" {
		out.Ledger.BaseDir = config.LedgerBaseDir
	}
	if strings.TrimSpace(out.Ledger.Namespace) == "" && strings.TrimSpace(config.LedgerNamespace) != "" {
		out.Ledger.Namespace = config.LedgerNamespace
	}
	if strings.TrimSpace(out.Ledger.WorkspaceID) == "" && strings.TrimSpace(config.LedgerWorkspaceID) != "" {
		out.Ledger.WorkspaceID = config.LedgerWorkspaceID
	}
	out.Ledger.DualWrite = out.Ledger.DualWrite || config.LedgerDualWrite
	out.Ledger.ReadEnabled = out.Ledger.ReadEnabled || config.LedgerReadEnabled
	out.Ledger.ShadowCompare = out.Ledger.ShadowCompare || config.LedgerShadowCompare
	out.Truth.Enabled = out.Truth.Enabled || config.TruthEnabled
	out.Truth.DualWrite = out.Truth.DualWrite || config.TruthDualWrite
	if strings.TrimSpace(out.Truth.BaseDir) == "" && strings.TrimSpace(config.TruthBaseDir) != "" {
		out.Truth.BaseDir = config.TruthBaseDir
	}
	out.Truth.ShadowFailOpen = out.Truth.ShadowFailOpen || config.TruthShadowFailOpen
	out.Truth.ReadEnabled = out.Truth.ReadEnabled || config.TruthReadEnabled
	if out.Truth.TopK == 0 && config.TruthTopK != 0 {
		out.Truth.TopK = config.TruthTopK
	}
	if out.Truth.MinSupportRefs == 0 && config.TruthMinSupportRefs != 0 {
		out.Truth.MinSupportRefs = config.TruthMinSupportRefs
	}
	out.Recall.IntentPlannerEnabled = out.Recall.IntentPlannerEnabled || config.IntentPlannerEnabled
	out.Vector.Enabled = out.Vector.Enabled || config.VectorEnabled
	if strings.TrimSpace(out.Vector.Path) == "" && strings.TrimSpace(config.VectorPath) != "" {
		out.Vector.Path = config.VectorPath
	}
	if out.Vector.TopK == 0 && config.VectorTopK != 0 {
		out.Vector.TopK = config.VectorTopK
	}
	if out.Vector.MinScore == 0 && config.VectorMinScore != 0 {
		out.Vector.MinScore = config.VectorMinScore
	}
	out.Recall.ShadowEnabled = out.Recall.ShadowEnabled || config.ShadowRecallEnabled
	out.Recall.HybridRerankEnabled = out.Recall.HybridRerankEnabled || config.HybridRerankEnabled
	out.Recall.RerankDebugEnabled = out.Recall.RerankDebugEnabled || config.RerankDebugEnabled
	out.Recall.BucketReadEnabled = out.Recall.BucketReadEnabled || config.BucketReadEnabled
	out.Recall.BucketShadowCompare = out.Recall.BucketShadowCompare || config.BucketShadowCompare
	if out.Recall.RecallInjectMinConfidence == 0 && config.RecallInjectMinConfidence != 0 {
		out.Recall.RecallInjectMinConfidence = config.RecallInjectMinConfidence
	}
	if out.Recall.ConflictPenalty == 0 && config.ConflictPenalty != 0 {
		out.Recall.ConflictPenalty = config.ConflictPenalty
	}
	if out.Recall.MaxBuckets == 0 && config.MaxBuckets != 0 {
		out.Recall.MaxBuckets = config.MaxBuckets
	}
	if out.Recall.MaxRecentBuckets == 0 && config.MaxRecentBuckets != 0 {
		out.Recall.MaxRecentBuckets = config.MaxRecentBuckets
	}
	if out.Recall.MaxHistoricalBuckets == 0 && config.MaxHistoricalBuckets != 0 {
		out.Recall.MaxHistoricalBuckets = config.MaxHistoricalBuckets
	}
	if out.Recall.BucketSessionBudget == 0 && config.BucketSessionBudget != 0 {
		out.Recall.BucketSessionBudget = config.BucketSessionBudget
	}
	out.Recall.BucketDebug = out.Recall.BucketDebug || config.BucketDebug
	out.Recall.CompressionEnabled = out.Recall.CompressionEnabled || config.CompressionEnabled
	if out.Recall.CompressionMinSupport == 0 && config.CompressionMinSupport != 0 {
		out.Recall.CompressionMinSupport = config.CompressionMinSupport
	}
	if len(out.Recall.CompressionAgeTiers) == 0 && len(config.CompressionAgeTiers) > 0 {
		out.Recall.CompressionAgeTiers = append([]CompressionAgeTier(nil), config.CompressionAgeTiers...)
	}
	out.Warm.AutoRecallEnabled = out.Warm.AutoRecallEnabled || config.AutoRecallEnabled
	if out.Warm.AutoRecallLimit == 0 && config.AutoRecallLimit != 0 {
		out.Warm.AutoRecallLimit = config.AutoRecallLimit
	}
	if out.Warm.TTL == 0 && config.WarmTTL != 0 {
		out.Warm.TTL = config.WarmTTL
	}
	out.Warm.TemporalDecayEnabled = out.Warm.TemporalDecayEnabled || config.TemporalDecayEnabled
	if out.Warm.TemporalDecayHalfLife == 0 && config.TemporalDecayHalfLife != 0 {
		out.Warm.TemporalDecayHalfLife = config.TemporalDecayHalfLife
	}
	out.Warm.AnchorEnabled = out.Warm.AnchorEnabled || config.AnchorEnabled
	if out.Warm.AnchorMinWeight == 0 && config.AnchorMinWeight != 0 {
		out.Warm.AnchorMinWeight = config.AnchorMinWeight
	}
	if out.Warm.EvolutionInterval == 0 && config.EvolutionInterval != 0 {
		out.Warm.EvolutionInterval = config.EvolutionInterval
	}
	out.Warm.EvolutionEnabled = out.Warm.EvolutionEnabled || config.EvolutionEnabled
	out.Warm.EvolutionUseWorker = out.Warm.EvolutionUseWorker || config.EvolutionUseWorker
	if out.Warm.EvolutionBatchSize == 0 && config.EvolutionBatchSize != 0 {
		out.Warm.EvolutionBatchSize = config.EvolutionBatchSize
	}
	out.Graph.Enabled = out.Graph.Enabled || config.GraphEnabled
	if strings.TrimSpace(out.Graph.Path) == "" && strings.TrimSpace(config.GraphPath) != "" {
		out.Graph.Path = config.GraphPath
	}
	out.Graph.ExtractOnArchive = out.Graph.ExtractOnArchive || config.GraphExtractOnArchive
	out.Graph.ExtractOnEvolve = out.Graph.ExtractOnEvolve || config.GraphExtractOnEvolve
	if out.Graph.MaxHops == 0 && config.GraphMaxHops != 0 {
		out.Graph.MaxHops = config.GraphMaxHops
	}
	if out.Graph.MaxHits == 0 && config.GraphMaxHits != 0 {
		out.Graph.MaxHits = config.GraphMaxHits
	}
	if out.Graph.MinConfidence == 0 && config.GraphMinConfidence != 0 {
		out.Graph.MinConfidence = config.GraphMinConfidence
	}
	if strings.TrimSpace(out.Graph.Namespace) == "" && strings.TrimSpace(config.GraphNamespace) != "" {
		out.Graph.Namespace = config.GraphNamespace
	}
	out.Graph.DebugEnabled = out.Graph.DebugEnabled || config.GraphDebugEnabled
	out.Decision.Enabled = out.Decision.Enabled || config.DecisionEnabled
	out.Decision.CaptureOnTurn = out.Decision.CaptureOnTurn || config.DecisionCaptureOnTurn
	out.Decision.CaptureOnTurnSet = out.Decision.CaptureOnTurnSet || config.DecisionCaptureOnTurnSet
	if strings.TrimSpace(out.Decision.Path) == "" && strings.TrimSpace(config.DecisionPath) != "" {
		out.Decision.Path = config.DecisionPath
	}
	if out.Decision.MaxHits == 0 && config.DecisionMaxHits != 0 {
		out.Decision.MaxHits = config.DecisionMaxHits
	}
	if out.Decision.MinConfidence == 0 && config.DecisionMinConfidence != 0 {
		out.Decision.MinConfidence = config.DecisionMinConfidence
	}
	if out.Decision.MinReuseScore == 0 && config.DecisionMinReuseScore != 0 {
		out.Decision.MinReuseScore = config.DecisionMinReuseScore
	}
	out.Decision.DebugEnabled = out.Decision.DebugEnabled || config.DecisionDebugEnabled
	out.Decision.Recipe.Enabled = out.Decision.Recipe.Enabled || config.DecisionRecipeEnabled
	if out.Decision.Recipe.Interval == 0 && config.DecisionRecipeInterval != 0 {
		out.Decision.Recipe.Interval = config.DecisionRecipeInterval
	}
	if out.Decision.Recipe.MinSupport == 0 && config.DecisionRecipeMinSupport != 0 {
		out.Decision.Recipe.MinSupport = config.DecisionRecipeMinSupport
	}
	out.Decision.Recipe.ReuseEnabled = out.Decision.Recipe.ReuseEnabled || config.RecipeReuseEnabled
	out.Decision.Recipe.ReuseEnabledSet = out.Decision.Recipe.ReuseEnabledSet || config.RecipeReuseEnabledSet
	out.Decision.Recipe.ExecutionTrackingEnabled = out.Decision.Recipe.ExecutionTrackingEnabled || config.RecipeExecutionTrackingEnabled
	out.Decision.Recipe.ExecutionTrackingEnabledSet = out.Decision.Recipe.ExecutionTrackingEnabledSet || config.RecipeExecutionTrackingEnabledSet
	out.Decision.Recipe.BackfillEnabled = out.Decision.Recipe.BackfillEnabled || config.RecipeBackfillEnabled
	out.Decision.Recipe.BackfillEnabledSet = out.Decision.Recipe.BackfillEnabledSet || config.RecipeBackfillEnabledSet
	out.Decision.Recipe.DefaultEnabled = out.Decision.Recipe.DefaultEnabled || config.RecipeDefaultEnabled
	out.Decision.Recipe.DefaultEnabledSet = out.Decision.Recipe.DefaultEnabledSet || config.RecipeDefaultEnabledSet
	if out.Decision.Recipe.DefaultGrayPercent == 0 && config.RecipeDefaultGrayPercent != 0 {
		out.Decision.Recipe.DefaultGrayPercent = config.RecipeDefaultGrayPercent
	}
	if out.Decision.Recipe.MinSelectionConfidence == 0 && config.RecipeMinSelectionConfidence != 0 {
		out.Decision.Recipe.MinSelectionConfidence = config.RecipeMinSelectionConfidence
	}
	if out.Decision.Recipe.MinSuccessRate == 0 && config.RecipeMinSuccessRate != 0 {
		out.Decision.Recipe.MinSuccessRate = config.RecipeMinSuccessRate
	}
	if out.Decision.Recipe.BackfillBatchSize == 0 && config.RecipeBackfillBatchSize != 0 {
		out.Decision.Recipe.BackfillBatchSize = config.RecipeBackfillBatchSize
	}
	if out.Decision.Recipe.BackfillInterval == 0 && config.RecipeBackfillInterval != 0 {
		out.Decision.Recipe.BackfillInterval = config.RecipeBackfillInterval
	}
	if out.Runtime.SessionStore == nil && config.SessionStore != nil {
		out.Runtime.SessionStore = config.SessionStore
	}
	if out.Runtime.Summarizer == nil && config.Summarizer != nil {
		out.Runtime.Summarizer = config.Summarizer
	}
	return out
}
