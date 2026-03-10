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
		Provider: ProviderConfig{
			Type:                       runtime.Provider,
			APIKey:                     runtime.APIKey,
			BaseURL:                    runtime.BaseURL,
			Model:                      runtime.Model,
			Headers:                    headers,
			AnthropicVersion:           valueOrEnv(fileCfg.AnthropicVersion, "GHOST_ANTHROPIC_VERSION", defaultAnthropicVersion),
			AnthropicMaxTokens:         intOrEnv(fileCfg.AnthropicMaxTokens, "GHOST_ANTHROPIC_MAX_TOKENS", defaultAnthropicMaxTokens),
			ContextWindowTokens:        runtime.ContextWindowTokens,
			ResponseReserveTokens:      runtime.ResponseReserveTokens,
			ModelContextWindowTokens:   cloneModelTokenOverrides(runtime.ModelContextWindowTokens),
			ModelResponseReserveTokens: cloneModelTokenOverrides(runtime.ModelResponseReserveTokens),
		},
		Memory: MemoryRuntimeConfig{
			WarmPath:                                  memoryWarmPathFromEnv(),
			ColdPath:                                  memoryColdPathFromEnv(),
			LedgerPath:                                memoryLedgerPathFromEnv(),
			LedgerReadEnabled:                         memoryLedgerReadEnabledFromEnv(),
			AutoRecallEnabled:                         memoryAutoRecallEnabledFromEnv(),
			AutoRecallLimit:                           memoryAutoRecallLimitFromEnv(),
			WarmTTL:                                   memoryWarmTTLFromEnv(),
			TemporalDecayEnabled:                      memoryTemporalDecayEnabledFromEnv(),
			TemporalDecayHalfLife:                     memoryTemporalDecayHalfLifeFromEnv(),
			AnchorEnabled:                             memoryAnchorEnabledFromEnv(),
			AnchorMinWeight:                           memoryAnchorMinWeightFromEnv(),
			EvolutionInterval:                         memoryEvolutionIntervalFromEnv(),
			EvolutionEnabled:                          memoryEvolutionEnabledFromEnv(),
			EvolutionUseWorker:                        memoryEvolutionUseWorkerFromEnv(),
			EvolutionBatchSize:                        memoryEvolutionBatchSizeFromEnv(),
			GraphEnabled:                              memoryGraphEnabledFromEnv(),
			GraphPath:                                 memoryGraphPathFromEnv(),
			GraphExtractOnArchive:                     memoryGraphExtractOnArchiveFromEnv(),
			GraphExtractOnEvolve:                      memoryGraphExtractOnEvolveFromEnv(),
			GraphMaxHops:                              memoryGraphMaxHopsFromEnv(),
			GraphMaxHits:                              memoryGraphMaxHitsFromEnv(),
			GraphMinConfidence:                        memoryGraphMinConfidenceFromEnv(),
			GraphNamespace:                            memoryGraphNamespaceFromEnv(),
			GraphDebugEnabled:                         memoryGraphDebugEnabledFromEnv(),
			DecisionEnabled:                           memoryDecisionEnabledFromEnv(),
			DecisionCaptureOnTurn:                     memoryDecisionCaptureOnTurnFromEnv(),
			DecisionPath:                              memoryDecisionPathFromEnv(),
			DecisionMaxHits:                           memoryDecisionMaxHitsFromEnv(),
			DecisionMinConfidence:                     memoryDecisionMinConfidenceFromEnv(),
			DecisionMinReuseScore:                     memoryDecisionMinReuseScoreFromEnv(),
			DecisionRecipeEnabled:                     memoryDecisionRecipeEnabledFromEnv(),
			DecisionRecipeInterval:                    memoryDecisionRecipeIntervalFromEnv(),
			DecisionRecipeMinSupport:                  memoryDecisionRecipeMinSupportFromEnv(),
			DecisionDebugEnabled:                      memoryDecisionDebugEnabledFromEnv(),
			DecisionSelectorHintEnabled:               memoryDecisionSelectorHintEnabledFromEnv(),
			DecisionRecipeReuseEnabled:                recipeReuseEnabled,
			DecisionRecipeReuseEnabledSet:             recipeReuseEnabledSet,
			DecisionRecipeExecutionTrackingEnabled:    recipeExecutionTrackingEnabled,
			DecisionRecipeExecutionTrackingEnabledSet: recipeExecutionTrackingEnabledSet,
			DecisionRecipeBackfillEnabled:             recipeBackfillEnabled,
			DecisionRecipeBackfillEnabledSet:          recipeBackfillEnabledSet,
			DecisionRecipeDefaultEnabled:              recipeDefaultEnabled,
			DecisionRecipeDefaultEnabledSet:           recipeDefaultEnabledSet,
			DecisionRecipeDefaultGrayPercent:          memoryDecisionRecipeDefaultGrayPercentFromEnv(),
			DecisionRecipeMinSelectionConfidence:      memoryDecisionRecipeMinSelectionConfidenceFromEnv(),
			DecisionRecipeMinSuccessRate:              memoryDecisionRecipeMinSuccessRateFromEnv(),
			DecisionRecipeBackfillBatchSize:           memoryDecisionRecipeBackfillBatchSizeFromEnv(),
			DecisionRecipeBackfillInterval:            memoryDecisionRecipeBackfillIntervalFromEnv(),
		},
		RSS: RSSConfig{
			FeedsPath:           rssFeedsPathFromEnv(),
			InboxPath:           rssInboxPathFromEnv(),
			BriefingsPath:       rssBriefingsPathFromEnv(),
			ReportsPath:         rssReportsPathFromEnv(),
			PollEnabled:         rssPollEnabledFromEnv(),
			PollInterval:        rssPollIntervalFromEnv(),
			PollMaxItemsPerFeed: rssPollMaxItemsPerFeedFromEnv(),
			AIBatchSize:         rssAIBatchSizeFromEnv(),
			BriefingEnabled:     rssBriefingEnabledFromEnv(),
			BriefingInterval:    rssBriefingIntervalFromEnv(),
		},
		Worker: WorkerConfig{
			Model:          valueOrEnv(fileCfg.WorkerModel, "GHOST_WORKER_MODEL", ""),
			MaxConcurrency: intOrEnv(fileCfg.WorkerMaxConcurrency, "GHOST_WORKER_MAX_CONCURRENCY", defaultWorkerMaxConcurrency),
			MaxFiles:       intOrEnv(fileCfg.WorkerMaxFiles, "GHOST_WORKER_MAX_FILES", defaultWorkerMaxFiles),
			MaxFileChunks:  intOrEnv(fileCfg.WorkerMaxFileChunks, "GHOST_WORKER_MAX_FILE_CHUNKS", defaultWorkerMaxFileChunks),
		},
		ToolSelector: ToolSelectorConfig{
			Enabled:    boolOrEnv(fileCfg.ToolSelectorEnabled, "GHOST_TOOL_SELECTOR_ENABLED", false),
			Mode:       strings.ToLower(valueOrEnv(fileCfg.ToolSelectorMode, "GHOST_TOOL_SELECTOR_MODE", "llm")),
			Model:      valueOrEnv(fileCfg.ToolSelectorModel, "GHOST_TOOL_SELECTOR_MODEL", ""),
			TimeoutMS:  intOrEnv(fileCfg.ToolSelectorTimeoutMS, "GHOST_TOOL_SELECTOR_TIMEOUT_MS", defaultToolSelectorTimeoutMS),
			Confidence: floatOrEnv(fileCfg.ToolSelectorConfidence, "GHOST_TOOL_SELECTOR_CONFIDENCE", defaultToolSelectorConfidence),
			Shadow:     boolOrEnv(fileCfg.ToolSelectorShadow, "GHOST_TOOL_SELECTOR_SHADOW", false),
			RecentMsgs: intOrEnv(fileCfg.ToolSelectorRecentMsgs, "GHOST_TOOL_SELECTOR_RECENT_MESSAGES", defaultToolSelectorRecentMsgs),
			Allowlist:  toolNameListOrEnv(fileCfg.ToolAllowlist, "GHOST_TOOL_ALLOWLIST"),
			Blocklist:  toolNameListOrEnv(fileCfg.ToolBlocklist, "GHOST_TOOL_BLOCKLIST"),
		},
		NativePersistent:        runtime.NativePersistent,
		NativeBinaryPath:        nativeBinaryPathFromEnv(),
		NativeBinaryRoots:       nativeBinaryRootsFromEnv(),
		NativeBinaryCandidates:  nativeBinaryCandidatesFromEnv(),
		NativeAllowedReadPaths:  nativeAllowedReadPathsFromEnv(),
		NativeAllowedWritePaths: nativeAllowedWritePathsFromEnv(),
		ChatPath:                runtime.ChatPath,
		PromptsPath:             valueOrEnv(fileCfg.PromptsPath, "GHOST_PROMPTS_PATH", defaultPromptsPath),
		SessionsPath:            sessionsPathFromEnv(),
		WebSearchTavilyAPIKey:   webSearchTavilyAPIKeyFromEnv(),
		MaxTurns:                intOrEnv(fileCfg.MaxTurns, "GHOST_MAX_TURNS", defaultMaxTurns),
	}
	allowlist, blocklist, err := normalizeConfiguredToolLists(cfg.ToolSelector.Allowlist, cfg.ToolSelector.Blocklist)
	if err != nil {
		return Config{}, err
	}
	cfg.ToolSelector.Allowlist = allowlist
	cfg.ToolSelector.Blocklist = blocklist

	return cfg, nil
}
