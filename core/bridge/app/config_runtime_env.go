package app

import (
	"os"
	"strconv"
	"strings"
	"time"
)

func getenvDefault(name, fallback string) string {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return fallback
	}
	return v
}

func boolFromFileOrEnv(raw *bool, envName string, fallback bool) (bool, bool) {
	if raw != nil {
		return *raw, true
	}
	rawValue := strings.TrimSpace(os.Getenv(envName))
	if rawValue == "" {
		return fallback, false
	}
	value, err := strconv.ParseBool(rawValue)
	if err != nil {
		return fallback, false
	}
	return value, true
}

// sessionsPathFromEnv 返回会话持久化目录。
func sessionsPathFromEnv() string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return getenvDefault("GHOST_SESSIONS_PATH", defaultSessionsPath)
	}
	return valueOrEnv(fileCfg.SessionsPath, "GHOST_SESSIONS_PATH", defaultSessionsPath)
}

func rssFeedsPathFromEnv() string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return getenvDefault("GHOST_RSS_FEEDS_PATH", defaultRSSFeedsPath)
	}
	return valueOrEnv(fileCfg.RSSFeedsPath, "GHOST_RSS_FEEDS_PATH", defaultRSSFeedsPath)
}

func rssInboxPathFromEnv() string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return getenvDefault("GHOST_RSS_INBOX_PATH", defaultRSSInboxPath)
	}
	return valueOrEnv(fileCfg.RSSInboxPath, "GHOST_RSS_INBOX_PATH", defaultRSSInboxPath)
}

func rssPollEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_RSS_POLL_ENABLED", true)
	}
	return boolOrEnv(fileCfg.RSSPollEnabled, "GHOST_RSS_POLL_ENABLED", true)
}

func rssPollIntervalFromEnv() time.Duration {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseDurationEnv("GHOST_RSS_POLL_INTERVAL", defaultRSSPollInterval)
	}
	return durationOrEnv(fileCfg.RSSPollInterval, "GHOST_RSS_POLL_INTERVAL", defaultRSSPollInterval)
}

func rssPollMaxItemsPerFeedFromEnv() int {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parsePositiveIntEnv("GHOST_RSS_POLL_MAX_ITEMS_PER_FEED", defaultRSSPollMaxItemsPerFeed)
	}
	return intOrEnv(fileCfg.RSSPollMaxItemsPerFeed, "GHOST_RSS_POLL_MAX_ITEMS_PER_FEED", defaultRSSPollMaxItemsPerFeed)
}

func rssAIBatchSizeFromEnv() int {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parsePositiveIntEnv("GHOST_RSS_AI_BATCH_SIZE", defaultRSSAIBatchSize)
	}
	return intOrEnv(fileCfg.RSSAIBatchSize, "GHOST_RSS_AI_BATCH_SIZE", defaultRSSAIBatchSize)
}

func tasksPathFromEnv() string {
	return getenvDefault("GHOST_TASKS_PATH", defaultTasksPath)
}

// memoryWarmPathFromEnv 返回 warm memory 持久化文件路径。
func memoryWarmPathFromEnv() string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return getenvDefault("GHOST_MEMORY_WARM_PATH", defaultMemoryWarmPath)
	}
	return valueOrEnv(fileCfg.MemoryWarmPath, "GHOST_MEMORY_WARM_PATH", defaultMemoryWarmPath)
}

// memoryColdPathFromEnv 返回 cold memory 根目录路径。
func memoryColdPathFromEnv() string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return getenvDefault("GHOST_MEMORY_COLD_PATH", defaultMemoryColdPath)
	}
	return valueOrEnv(fileCfg.MemoryColdPath, "GHOST_MEMORY_COLD_PATH", defaultMemoryColdPath)
}

func memoryLedgerPathFromEnv() string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return getenvDefault("GHOST_MEMORY_LEDGER_PATH", "")
	}
	return valueOrEnv(fileCfg.MemoryLedgerPath, "GHOST_MEMORY_LEDGER_PATH", "")
}

func memoryLedgerDualWriteFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_LEDGER_DUAL_WRITE", false)
	}
	return boolOrEnv(fileCfg.MemoryLedgerDualWrite, "GHOST_MEMORY_LEDGER_DUAL_WRITE", false)
}

func memoryLedgerReadEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_LEDGER_READ_ENABLED", false)
	}
	return boolOrEnv(fileCfg.MemoryLedgerReadEnabled, "GHOST_MEMORY_LEDGER_READ_ENABLED", false)
}

func memoryLedgerShadowCompareFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_LEDGER_SHADOW_COMPARE", false)
	}
	return boolOrEnv(fileCfg.MemoryLedgerShadowCompare, "GHOST_MEMORY_LEDGER_SHADOW_COMPARE", false)
}

// memoryAutoRecallEnabledFromEnv 控制 warm 自动召回是否启用。
func memoryAutoRecallEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_AUTO_RECALL_ENABLED", true)
	}
	return boolOrEnv(fileCfg.MemoryAutoRecallEnabled, "GHOST_MEMORY_AUTO_RECALL_ENABLED", true)
}

// memoryAutoRecallLimitFromEnv 返回自动召回注入条目上限。
func memoryAutoRecallLimitFromEnv() int {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parsePositiveIntEnv("GHOST_MEMORY_AUTO_RECALL_LIMIT", defaultMemoryAutoRecallLimit)
	}
	return intOrEnv(fileCfg.MemoryAutoRecallLimit, "GHOST_MEMORY_AUTO_RECALL_LIMIT", defaultMemoryAutoRecallLimit)
}

// memoryWarmTTLFromEnv 返回 warm 层默认 TTL。
func memoryWarmTTLFromEnv() time.Duration {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseDurationEnv("GHOST_MEMORY_WARM_TTL", defaultMemoryWarmTTL)
	}
	return durationOrEnv(fileCfg.MemoryWarmTTL, "GHOST_MEMORY_WARM_TTL", defaultMemoryWarmTTL)
}

func memoryTemporalDecayEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_TEMPORAL_DECAY_ENABLED", true)
	}
	return boolOrEnv(fileCfg.MemoryTemporalDecayEnabled, "GHOST_MEMORY_TEMPORAL_DECAY_ENABLED", true)
}

func memoryTemporalDecayHalfLifeFromEnv() time.Duration {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseDurationEnv("GHOST_MEMORY_TEMPORAL_DECAY_HALF_LIFE", defaultMemoryTemporalHalfLife)
	}
	return durationOrEnv(fileCfg.MemoryTemporalDecayHalfLife, "GHOST_MEMORY_TEMPORAL_DECAY_HALF_LIFE", defaultMemoryTemporalHalfLife)
}

func memoryAnchorEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_ANCHOR_ENABLED", true)
	}
	return boolOrEnv(fileCfg.MemoryAnchorEnabled, "GHOST_MEMORY_ANCHOR_ENABLED", true)
}

func memoryAnchorMinWeightFromEnv() float64 {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseFloatEnv("GHOST_MEMORY_ANCHOR_MIN_WEIGHT", defaultMemoryAnchorMinWeight)
	}
	return floatOrEnv(fileCfg.MemoryAnchorMinWeight, "GHOST_MEMORY_ANCHOR_MIN_WEIGHT", defaultMemoryAnchorMinWeight)
}

// memoryEvolutionIntervalFromEnv 返回后台演化间隔。
func memoryEvolutionIntervalFromEnv() time.Duration {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseDurationEnv("GHOST_MEMORY_EVOLUTION_INTERVAL", defaultMemoryEvolutionInterval)
	}
	return durationOrEnv(fileCfg.MemoryEvolutionInterval, "GHOST_MEMORY_EVOLUTION_INTERVAL", defaultMemoryEvolutionInterval)
}

// memoryEvolutionEnabledFromEnv 控制后台演化协程是否启用。
func memoryEvolutionEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_EVOLUTION_ENABLED", false)
	}
	return boolOrEnv(fileCfg.MemoryEvolutionEnabled, "GHOST_MEMORY_EVOLUTION_ENABLED", false)
}

func memoryEvolutionUseWorkerFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_EVOLUTION_USE_WORKER", true)
	}
	return boolOrEnv(fileCfg.MemoryEvolutionUseWorker, "GHOST_MEMORY_EVOLUTION_USE_WORKER", true)
}

func memoryEvolutionBatchSizeFromEnv() int {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parsePositiveIntEnv("GHOST_MEMORY_EVOLUTION_BATCH_SIZE", defaultMemoryEvolutionBatchSize)
	}
	return intOrEnv(fileCfg.MemoryEvolutionBatchSize, "GHOST_MEMORY_EVOLUTION_BATCH_SIZE", defaultMemoryEvolutionBatchSize)
}

func memoryGraphEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_GRAPH_ENABLED", false)
	}
	return boolOrEnv(fileCfg.MemoryGraphEnabled, "GHOST_MEMORY_GRAPH_ENABLED", false)
}

func memoryGraphPathFromEnv() string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return getenvDefault("GHOST_MEMORY_GRAPH_PATH", defaultMemoryGraphPath)
	}
	return valueOrEnv(fileCfg.MemoryGraphPath, "GHOST_MEMORY_GRAPH_PATH", defaultMemoryGraphPath)
}

func memoryGraphExtractOnArchiveFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_GRAPH_EXTRACT_ON_ARCHIVE", true)
	}
	return boolOrEnv(fileCfg.MemoryGraphExtractOnArchive, "GHOST_MEMORY_GRAPH_EXTRACT_ON_ARCHIVE", true)
}

func memoryGraphExtractOnEvolveFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_GRAPH_EXTRACT_ON_EVOLVE", true)
	}
	return boolOrEnv(fileCfg.MemoryGraphExtractOnEvolve, "GHOST_MEMORY_GRAPH_EXTRACT_ON_EVOLVE", true)
}

func memoryGraphMaxHopsFromEnv() int {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parsePositiveIntEnv("GHOST_MEMORY_GRAPH_MAX_HOPS", defaultMemoryGraphMaxHops)
	}
	return intOrEnv(fileCfg.MemoryGraphMaxHops, "GHOST_MEMORY_GRAPH_MAX_HOPS", defaultMemoryGraphMaxHops)
}

func memoryGraphMaxHitsFromEnv() int {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parsePositiveIntEnv("GHOST_MEMORY_GRAPH_MAX_HITS", defaultMemoryGraphMaxHits)
	}
	return intOrEnv(fileCfg.MemoryGraphMaxHits, "GHOST_MEMORY_GRAPH_MAX_HITS", defaultMemoryGraphMaxHits)
}

func memoryGraphMinConfidenceFromEnv() float64 {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseFloatEnv("GHOST_MEMORY_GRAPH_MIN_CONFIDENCE", defaultMemoryGraphMinConfidence)
	}
	return floatOrEnv(fileCfg.MemoryGraphMinConfidence, "GHOST_MEMORY_GRAPH_MIN_CONFIDENCE", defaultMemoryGraphMinConfidence)
}

func memoryGraphNamespaceFromEnv() string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return getenvDefault("GHOST_MEMORY_GRAPH_NAMESPACE", defaultMemoryGraphNamespace)
	}
	return valueOrEnv(fileCfg.MemoryGraphNamespace, "GHOST_MEMORY_GRAPH_NAMESPACE", defaultMemoryGraphNamespace)
}

func memoryGraphDebugEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_GRAPH_DEBUG_ENABLED", false)
	}
	return boolOrEnv(fileCfg.MemoryGraphDebugEnabled, "GHOST_MEMORY_GRAPH_DEBUG_ENABLED", false)
}

func memoryDecisionEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_DECISION_ENABLED", false)
	}
	return boolOrEnv(fileCfg.MemoryDecisionEnabled, "GHOST_MEMORY_DECISION_ENABLED", false)
}

func memoryDecisionPathFromEnv() string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return getenvDefault("GHOST_MEMORY_DECISION_PATH", defaultMemoryDecisionPath)
	}
	return valueOrEnv(fileCfg.MemoryDecisionPath, "GHOST_MEMORY_DECISION_PATH", defaultMemoryDecisionPath)
}

func memoryDecisionCaptureOnTurnFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_DECISION_CAPTURE_ON_TURN", defaultMemoryDecisionCaptureOnTurn)
	}
	return boolOrEnv(fileCfg.MemoryDecisionCaptureOnTurn, "GHOST_MEMORY_DECISION_CAPTURE_ON_TURN", defaultMemoryDecisionCaptureOnTurn)
}

func memoryDecisionMaxHitsFromEnv() int {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parsePositiveIntEnv("GHOST_MEMORY_DECISION_MAX_HITS", defaultMemoryDecisionMaxHits)
	}
	return intOrEnv(fileCfg.MemoryDecisionMaxHits, "GHOST_MEMORY_DECISION_MAX_HITS", defaultMemoryDecisionMaxHits)
}

func memoryDecisionMinConfidenceFromEnv() float64 {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseFloatEnv("GHOST_MEMORY_DECISION_MIN_CONFIDENCE", defaultMemoryDecisionMinConfidence)
	}
	return floatOrEnv(fileCfg.MemoryDecisionMinConfidence, "GHOST_MEMORY_DECISION_MIN_CONFIDENCE", defaultMemoryDecisionMinConfidence)
}

func memoryDecisionMinReuseScoreFromEnv() float64 {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseFloatEnv("GHOST_MEMORY_DECISION_MIN_REUSE_SCORE", defaultMemoryDecisionMinReuseScore)
	}
	return floatOrEnv(fileCfg.MemoryDecisionMinReuseScore, "GHOST_MEMORY_DECISION_MIN_REUSE_SCORE", defaultMemoryDecisionMinReuseScore)
}

func memoryDecisionRecipeEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_DECISION_RECIPE_ENABLED", true)
	}
	return boolOrEnv(fileCfg.MemoryDecisionRecipeEnabled, "GHOST_MEMORY_DECISION_RECIPE_ENABLED", true)
}

func memoryDecisionRecipeIntervalFromEnv() time.Duration {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseDurationEnv("GHOST_MEMORY_DECISION_RECIPE_INTERVAL", defaultMemoryDecisionRecipeInterval)
	}
	return durationOrEnv(fileCfg.MemoryDecisionRecipeInterval, "GHOST_MEMORY_DECISION_RECIPE_INTERVAL", defaultMemoryDecisionRecipeInterval)
}

func memoryDecisionRecipeMinSupportFromEnv() int {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parsePositiveIntEnv("GHOST_MEMORY_DECISION_RECIPE_MIN_SUPPORT", defaultMemoryDecisionRecipeMinSupport)
	}
	return intOrEnv(fileCfg.MemoryDecisionRecipeMinSupport, "GHOST_MEMORY_DECISION_RECIPE_MIN_SUPPORT", defaultMemoryDecisionRecipeMinSupport)
}

func memoryDecisionDebugEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_DECISION_DEBUG_ENABLED", false)
	}
	return boolOrEnv(fileCfg.MemoryDecisionDebugEnabled, "GHOST_MEMORY_DECISION_DEBUG_ENABLED", false)
}

func memoryDecisionSelectorHintEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_DECISION_SELECTOR_HINT_ENABLED", defaultMemoryDecisionSelectorHintEnabled)
	}
	return boolOrEnv(fileCfg.MemoryDecisionSelectorHintEnabled, "GHOST_MEMORY_DECISION_SELECTOR_HINT_ENABLED", defaultMemoryDecisionSelectorHintEnabled)
}

func memoryDecisionRecipeReuseEnabledFromEnv() (bool, bool) {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return boolFromFileOrEnv(nil, "GHOST_MEMORY_DECISION_RECIPE_REUSE_ENABLED", false)
	}
	return boolFromFileOrEnv(fileCfg.MemoryDecisionRecipeReuseEnabled, "GHOST_MEMORY_DECISION_RECIPE_REUSE_ENABLED", false)
}

func memoryDecisionRecipeExecutionTrackingEnabledFromEnv() (bool, bool) {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return boolFromFileOrEnv(nil, "GHOST_MEMORY_DECISION_RECIPE_EXECUTION_TRACKING_ENABLED", false)
	}
	return boolFromFileOrEnv(fileCfg.MemoryDecisionRecipeExecutionTrackingEnabled, "GHOST_MEMORY_DECISION_RECIPE_EXECUTION_TRACKING_ENABLED", false)
}

func memoryDecisionRecipeBackfillEnabledFromEnv() (bool, bool) {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return boolFromFileOrEnv(nil, "GHOST_MEMORY_DECISION_RECIPE_BACKFILL_ENABLED", false)
	}
	return boolFromFileOrEnv(fileCfg.MemoryDecisionRecipeBackfillEnabled, "GHOST_MEMORY_DECISION_RECIPE_BACKFILL_ENABLED", false)
}

func memoryDecisionRecipeDefaultEnabledFromEnv() (bool, bool) {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return boolFromFileOrEnv(nil, "GHOST_MEMORY_DECISION_RECIPE_DEFAULT_ENABLED", false)
	}
	return boolFromFileOrEnv(fileCfg.MemoryDecisionRecipeDefaultEnabled, "GHOST_MEMORY_DECISION_RECIPE_DEFAULT_ENABLED", false)
}

func memoryDecisionRecipeDefaultGrayPercentFromEnv() int {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parsePositiveIntEnv("GHOST_MEMORY_DECISION_RECIPE_DEFAULT_GRAY_PERCENT", defaultMemoryDecisionRecipeDefaultGrayPercent)
	}
	return intOrEnv(fileCfg.MemoryDecisionRecipeDefaultGrayPercent, "GHOST_MEMORY_DECISION_RECIPE_DEFAULT_GRAY_PERCENT", defaultMemoryDecisionRecipeDefaultGrayPercent)
}

func memoryDecisionRecipeMinSelectionConfidenceFromEnv() float64 {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseFloatEnv("GHOST_MEMORY_DECISION_RECIPE_MIN_SELECTION_CONFIDENCE", defaultMemoryDecisionRecipeMinSelectionConfidence)
	}
	return floatOrEnv(fileCfg.MemoryDecisionRecipeMinSelectionConfidence, "GHOST_MEMORY_DECISION_RECIPE_MIN_SELECTION_CONFIDENCE", defaultMemoryDecisionRecipeMinSelectionConfidence)
}

func memoryDecisionRecipeMinSuccessRateFromEnv() float64 {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseFloatEnv("GHOST_MEMORY_DECISION_RECIPE_MIN_SUCCESS_RATE", defaultMemoryDecisionRecipeMinSuccessRate)
	}
	return floatOrEnv(fileCfg.MemoryDecisionRecipeMinSuccessRate, "GHOST_MEMORY_DECISION_RECIPE_MIN_SUCCESS_RATE", defaultMemoryDecisionRecipeMinSuccessRate)
}

func memoryDecisionRecipeBackfillBatchSizeFromEnv() int {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parsePositiveIntEnv("GHOST_MEMORY_DECISION_RECIPE_BACKFILL_BATCH_SIZE", defaultMemoryDecisionRecipeBackfillBatchSize)
	}
	return intOrEnv(fileCfg.MemoryDecisionRecipeBackfillBatchSize, "GHOST_MEMORY_DECISION_RECIPE_BACKFILL_BATCH_SIZE", defaultMemoryDecisionRecipeBackfillBatchSize)
}

func memoryDecisionRecipeBackfillIntervalFromEnv() time.Duration {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseDurationEnv("GHOST_MEMORY_DECISION_RECIPE_BACKFILL_INTERVAL", defaultMemoryDecisionRecipeBackfillInterval)
	}
	return durationOrEnv(fileCfg.MemoryDecisionRecipeBackfillInterval, "GHOST_MEMORY_DECISION_RECIPE_BACKFILL_INTERVAL", defaultMemoryDecisionRecipeBackfillInterval)
}
