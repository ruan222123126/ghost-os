package app

import "strings"

// LoadConfig 从环境变量加载配置并做基础校验与归一化。
func LoadConfig() (Config, error) {
	runtime, err := runtimeConfigFromEnv()
	if err != nil {
		return Config{}, err
	}
	return loadConfigWithRuntime(runtime)
}

// loadConfigWithRuntime 在 runtimeConfig 基础上补齐环境默认值与执行期约束。
func loadConfigWithRuntime(runtime runtimeConfig) (Config, error) {
	runtime = normalizeRuntimeConfig(runtime)
	if err := validateRuntimeForExecution(runtime); err != nil {
		return Config{}, err
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return Config{}, err
	}

	headers, err := headersOrEnv(fileCfg.ProviderHeaders)
	if err != nil {
		return Config{}, err
	}

	recipeReuseEnabled, recipeReuseEnabledSet := memoryDecisionRecipeReuseEnabledFromEnv()
	recipeExecutionTrackingEnabled, recipeExecutionTrackingEnabledSet := memoryDecisionRecipeExecutionTrackingEnabledFromEnv()
	recipeBackfillEnabled, recipeBackfillEnabledSet := memoryDecisionRecipeBackfillEnabledFromEnv()
	recipeDefaultEnabled, recipeDefaultEnabledSet := memoryDecisionRecipeDefaultEnabledFromEnv()

	cfg := Config{
		Provider:                                        runtime.Provider,
		APIKey:                                          runtime.APIKey,
		BaseURL:                                         runtime.BaseURL,
		Model:                                           runtime.Model,
		NativePersistent:                                runtime.NativePersistent,
		WorkerModel:                                     valueOrEnv(fileCfg.WorkerModel, "GHOST_WORKER_MODEL", ""),
		ChatPath:                                        runtime.ChatPath,
		PromptsPath:                                     valueOrEnv(fileCfg.PromptsPath, "GHOST_PROMPTS_PATH", defaultPromptsPath),
		SessionsPath:                                    sessionsPathFromEnv(),
		MemoryWarmPath:                                  memoryWarmPathFromEnv(),
		MemoryColdPath:                                  memoryColdPathFromEnv(),
		MemoryAutoRecallEnabled:                         memoryAutoRecallEnabledFromEnv(),
		MemoryAutoRecallLimit:                           memoryAutoRecallLimitFromEnv(),
		MemoryWarmTTL:                                   memoryWarmTTLFromEnv(),
		MemoryTemporalDecayEnabled:                      memoryTemporalDecayEnabledFromEnv(),
		MemoryTemporalDecayHalfLife:                     memoryTemporalDecayHalfLifeFromEnv(),
		MemoryAnchorEnabled:                             memoryAnchorEnabledFromEnv(),
		MemoryAnchorMinWeight:                           memoryAnchorMinWeightFromEnv(),
		MemoryEvolutionInterval:                         memoryEvolutionIntervalFromEnv(),
		MemoryEvolutionEnabled:                          memoryEvolutionEnabledFromEnv(),
		MemoryEvolutionUseWorker:                        memoryEvolutionUseWorkerFromEnv(),
		MemoryEvolutionBatchSize:                        memoryEvolutionBatchSizeFromEnv(),
		MemoryGraphEnabled:                              memoryGraphEnabledFromEnv(),
		MemoryGraphPath:                                 memoryGraphPathFromEnv(),
		MemoryGraphExtractOnArchive:                     memoryGraphExtractOnArchiveFromEnv(),
		MemoryGraphExtractOnEvolve:                      memoryGraphExtractOnEvolveFromEnv(),
		MemoryGraphMaxHops:                              memoryGraphMaxHopsFromEnv(),
		MemoryGraphMaxHits:                              memoryGraphMaxHitsFromEnv(),
		MemoryGraphMinConfidence:                        memoryGraphMinConfidenceFromEnv(),
		MemoryGraphNamespace:                            memoryGraphNamespaceFromEnv(),
		MemoryGraphDebugEnabled:                         memoryGraphDebugEnabledFromEnv(),
		MemoryDecisionEnabled:                           memoryDecisionEnabledFromEnv(),
		MemoryDecisionCaptureOnTurn:                     memoryDecisionCaptureOnTurnFromEnv(),
		MemoryDecisionPath:                              memoryDecisionPathFromEnv(),
		MemoryDecisionMaxHits:                           memoryDecisionMaxHitsFromEnv(),
		MemoryDecisionMinConfidence:                     memoryDecisionMinConfidenceFromEnv(),
		MemoryDecisionMinReuseScore:                     memoryDecisionMinReuseScoreFromEnv(),
		MemoryDecisionRecipeEnabled:                     memoryDecisionRecipeEnabledFromEnv(),
		MemoryDecisionRecipeInterval:                    memoryDecisionRecipeIntervalFromEnv(),
		MemoryDecisionRecipeMinSupport:                  memoryDecisionRecipeMinSupportFromEnv(),
		MemoryDecisionDebugEnabled:                      memoryDecisionDebugEnabledFromEnv(),
		MemoryDecisionSelectorHintEnabled:               memoryDecisionSelectorHintEnabledFromEnv(),
		MemoryDecisionRecipeReuseEnabled:                recipeReuseEnabled,
		MemoryDecisionRecipeReuseEnabledSet:             recipeReuseEnabledSet,
		MemoryDecisionRecipeExecutionTrackingEnabled:    recipeExecutionTrackingEnabled,
		MemoryDecisionRecipeExecutionTrackingEnabledSet: recipeExecutionTrackingEnabledSet,
		MemoryDecisionRecipeBackfillEnabled:             recipeBackfillEnabled,
		MemoryDecisionRecipeBackfillEnabledSet:          recipeBackfillEnabledSet,
		MemoryDecisionRecipeDefaultEnabled:              recipeDefaultEnabled,
		MemoryDecisionRecipeDefaultEnabledSet:           recipeDefaultEnabledSet,
		MemoryDecisionRecipeDefaultGrayPercent:          memoryDecisionRecipeDefaultGrayPercentFromEnv(),
		MemoryDecisionRecipeMinSelectionConfidence:      memoryDecisionRecipeMinSelectionConfidenceFromEnv(),
		MemoryDecisionRecipeMinSuccessRate:              memoryDecisionRecipeMinSuccessRateFromEnv(),
		MemoryDecisionRecipeBackfillBatchSize:           memoryDecisionRecipeBackfillBatchSizeFromEnv(),
		MemoryDecisionRecipeBackfillInterval:            memoryDecisionRecipeBackfillIntervalFromEnv(),
		ProviderHeaders:                                 headers,
		AnthropicVersion:                                valueOrEnv(fileCfg.AnthropicVersion, "GHOST_ANTHROPIC_VERSION", defaultAnthropicVersion),
		AnthropicMaxTokens:                              intOrEnv(fileCfg.AnthropicMaxTokens, "GHOST_ANTHROPIC_MAX_TOKENS", defaultAnthropicMaxTokens),
		MaxTurns:                                        intOrEnv(fileCfg.MaxTurns, "GHOST_MAX_TURNS", defaultMaxTurns),
		WorkerMaxConcurrency:                            intOrEnv(fileCfg.WorkerMaxConcurrency, "GHOST_WORKER_MAX_CONCURRENCY", defaultWorkerMaxConcurrency),
		WorkerMaxFiles:                                  intOrEnv(fileCfg.WorkerMaxFiles, "GHOST_WORKER_MAX_FILES", defaultWorkerMaxFiles),
		WorkerMaxFileChunks:                             intOrEnv(fileCfg.WorkerMaxFileChunks, "GHOST_WORKER_MAX_FILE_CHUNKS", defaultWorkerMaxFileChunks),
		ToolSelectorEnabled:                             boolOrEnv(fileCfg.ToolSelectorEnabled, "GHOST_TOOL_SELECTOR_ENABLED", false),
		ToolSelectorMode:                                strings.ToLower(valueOrEnv(fileCfg.ToolSelectorMode, "GHOST_TOOL_SELECTOR_MODE", "llm")),
		ToolSelectorModel:                               valueOrEnv(fileCfg.ToolSelectorModel, "GHOST_TOOL_SELECTOR_MODEL", ""),
		ToolSelectorTimeoutMS:                           intOrEnv(fileCfg.ToolSelectorTimeoutMS, "GHOST_TOOL_SELECTOR_TIMEOUT_MS", defaultToolSelectorTimeoutMS),
		ToolSelectorConfidence:                          floatOrEnv(fileCfg.ToolSelectorConfidence, "GHOST_TOOL_SELECTOR_CONFIDENCE", defaultToolSelectorConfidence),
		ToolSelectorShadow:                              boolOrEnv(fileCfg.ToolSelectorShadow, "GHOST_TOOL_SELECTOR_SHADOW", false),
		ToolSelectorRecentMsgs:                          intOrEnv(fileCfg.ToolSelectorRecentMsgs, "GHOST_TOOL_SELECTOR_RECENT_MESSAGES", defaultToolSelectorRecentMsgs),
	}

	return cfg, nil
}
