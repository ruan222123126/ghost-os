package config

import (
	"errors"
	"fmt"
	"strings"
)

// Load 从环境变量加载配置并做基础校验与归一化。
func Load() (Config, error) {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return Config{}, err
	}
	return resolveConfig(fileCfg, currentEnv())
}

// loadConfigWithRuntime 在 runtimeConfig 基础上补齐环境默认值与执行期约束。
func loadConfigWithRuntime(runtime runtimeConfig) (Config, error) {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return Config{}, err
	}
	return resolveConfigWithRuntime(fileCfg, currentEnv(), runtime)
}

func resolveConfig(fileCfg bridgeFileConfig, env envSnapshot) (Config, error) {
	runtime, err := resolveRuntimeConfig(fileCfg, env)
	if err != nil {
		return Config{}, err
	}
	return resolveConfigWithRuntime(fileCfg, env, runtime)
}

func resolveConfigWithRuntime(fileCfg bridgeFileConfig, env envSnapshot, runtime runtimeConfig) (Config, error) {
	runtime = normalizeRuntimeConfig(runtime)
	if err := validateRuntimeForExecution(runtime); err != nil {
		return Config{}, err
	}
	headers, promptsDir, err := loadConfigEnvDetails(fileCfg, env)
	if err != nil {
		return Config{}, err
	}
	provider, err := buildProviderConfig(runtime, fileCfg, env, headers)
	if err != nil {
		return Config{}, err
	}
	rss, err := buildRSSConfig(fileCfg, env)
	if err != nil {
		return Config{}, err
	}
	worker, err := buildWorkerConfig(fileCfg, env)
	if err != nil {
		return Config{}, err
	}
	toolSelector, err := buildToolSelectorConfig(fileCfg, env)
	if err != nil {
		return Config{}, err
	}
	toolSearch, err := buildToolSearchConfig(fileCfg, env)
	if err != nil {
		return Config{}, err
	}
	memoryAugmentation, err := buildMemoryAugmentationConfig(fileCfg, env)
	if err != nil {
		return Config{}, err
	}
	proMaxIterations, err := intOrEnvWithEnv(
		fileCfg.ProMaxIterations,
		"pro_max_iterations",
		env,
		"GHOST_PRO_MAX_ITERATIONS",
		defaultProMaxIterations,
	)
	if err != nil {
		return Config{}, err
	}
	maxTurns, err := intOrEnvWithEnv(fileCfg.MaxTurns, "max_turns", env, "GHOST_MAX_TURNS", defaultMaxTurns)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Provider:                provider,
		RSS:                     rss,
		Worker:                  worker,
		GraphQL:                 runtime.GraphQL,
		ToolSelector:            toolSelector,
		ToolSearch:              toolSearch,
		MemoryAugmentation:      memoryAugmentation,
		NativePersistent:        runtime.NativePersistent,
		NativeBinaryPath:        resolveNativeBinaryPath(fileCfg, env),
		NativeBinaryRoots:       resolveNativeBinaryRoots(fileCfg, env),
		NativeBinaryCandidates:  resolveNativeBinaryCandidates(fileCfg, env),
		NativeAllowedReadPaths:  resolveNativeAllowedReadPaths(fileCfg, env),
		NativeAllowedWritePaths: resolveNativeAllowedWritePaths(fileCfg, env),
		ProjectRoot:             runtime.ProjectRoot,
		ChatPath:                runtime.ChatPath,
		PromptsPath:             valueOrEnvWithEnv(fileCfg.PromptsPath, env, "GHOST_PROMPTS_PATH", defaultPromptsPath),
		PromptsDir:              promptsDir,
		PromptsCoreFiles:        promptsCoreFiles(fileCfg, env),
		PromptsRuntimeConstraintFiles: promptPathListWithEnv(
			fileCfg.PromptsRuntimeConstraintFiles,
			env,
			"GHOST_PROMPTS_RUNTIME_CONSTRAINT_FILES",
		),
		PromptsResponseRuleFiles: promptPathListWithEnv(
			fileCfg.PromptsResponseRuleFiles,
			env,
			"GHOST_PROMPTS_RESPONSE_RULE_FILES",
		),
		SessionsPath:          resolveSessionsPath(fileCfg, env),
		WebSearchTavilyAPIKey: runtime.WebSearchTavilyAPIKey,
		WebSearchExaAPIKey:    runtime.WebSearchExaAPIKey,
		ProMaxIterations:      proMaxIterations,
		MaxTurns:              maxTurns,
	}
	return finalizeLoadedConfig(cfg)
}

func loadConfigEnvDetails(fileCfg bridgeFileConfig, env envSnapshot) (map[string]string, string, error) {
	headers, err := headersOrEnvWithEnv(fileCfg.ProviderHeaders, env)
	if err != nil {
		return nil, "", err
	}
	promptsDir, err := resolvePromptsDir(fileCfg, env)
	if err != nil {
		return nil, "", err
	}
	return headers, promptsDir, nil
}

func buildProviderConfig(runtime runtimeConfig, fileCfg bridgeFileConfig, env envSnapshot, headers map[string]string) (ProviderConfig, error) {
	anthropicMaxTokens, err := intOrEnvWithEnv(
		fileCfg.AnthropicMaxTokens,
		"anthropic_max_tokens",
		env,
		"GHOST_ANTHROPIC_MAX_TOKENS",
		defaultAnthropicMaxTokens,
	)
	if err != nil {
		return ProviderConfig{}, err
	}
	return ProviderConfig{
		Type:                       runtime.Provider,
		APIKey:                     runtime.APIKey,
		BaseURL:                    runtime.BaseURL,
		Model:                      runtime.Model,
		Headers:                    headers,
		AnthropicVersion:           valueOrEnvWithEnv(fileCfg.AnthropicVersion, env, "GHOST_ANTHROPIC_VERSION", defaultAnthropicVersion),
		AnthropicMaxTokens:         anthropicMaxTokens,
		ContextWindowTokens:        runtime.ContextWindowTokens,
		ResponseReserveTokens:      runtime.ResponseReserveTokens,
		ModelContextWindowTokens:   cloneModelTokenOverrides(runtime.ModelContextWindowTokens),
		ModelResponseReserveTokens: cloneModelTokenOverrides(runtime.ModelResponseReserveTokens),
	}, nil
}

func buildRSSConfig(fileCfg bridgeFileConfig, env envSnapshot) (RSSConfig, error) {
	pollEnabled, err := boolOrEnvWithEnv(fileCfg.RSSPollEnabled, env, "GHOST_RSS_POLL_ENABLED", true)
	if err != nil {
		return RSSConfig{}, err
	}
	pollInterval, err := durationOrEnvWithEnv(
		fileCfg.RSSPollInterval,
		"rss_poll_interval",
		env,
		"GHOST_RSS_POLL_INTERVAL",
		defaultRSSPollInterval,
	)
	if err != nil {
		return RSSConfig{}, err
	}
	pollMaxItemsPerFeed, err := intOrEnvWithEnv(
		fileCfg.RSSPollMaxItemsPerFeed,
		"rss_poll_max_items_per_feed",
		env,
		"GHOST_RSS_POLL_MAX_ITEMS_PER_FEED",
		defaultRSSPollMaxItemsPerFeed,
	)
	if err != nil {
		return RSSConfig{}, err
	}
	aiBatchSize, err := intOrEnvWithEnv(
		fileCfg.RSSAIBatchSize,
		"rss_ai_batch_size",
		env,
		"GHOST_RSS_AI_BATCH_SIZE",
		defaultRSSAIBatchSize,
	)
	if err != nil {
		return RSSConfig{}, err
	}
	briefingEnabled, err := boolOrEnvWithEnv(fileCfg.RSSBriefingEnabled, env, "GHOST_RSS_BRIEFING_ENABLED", true)
	if err != nil {
		return RSSConfig{}, err
	}
	briefingInterval, err := durationOrEnvWithEnv(
		fileCfg.RSSBriefingInterval,
		"rss_briefing_interval",
		env,
		"GHOST_RSS_BRIEFING_INTERVAL",
		defaultRSSBriefingInterval,
	)
	if err != nil {
		return RSSConfig{}, err
	}
	return RSSConfig{
		FeedsPath:           resolveRSSFeedsPath(fileCfg, env),
		InboxPath:           resolveRSSInboxPath(fileCfg, env),
		BriefingsPath:       resolveRSSBriefingsPath(fileCfg, env),
		ReportsPath:         resolveRSSReportsPath(fileCfg, env),
		PollEnabled:         pollEnabled,
		PollInterval:        pollInterval,
		PollMaxItemsPerFeed: pollMaxItemsPerFeed,
		AIBatchSize:         aiBatchSize,
		BriefingEnabled:     briefingEnabled,
		BriefingInterval:    briefingInterval,
	}, nil
}

func buildWorkerConfig(fileCfg bridgeFileConfig, env envSnapshot) (WorkerConfig, error) {
	maxConcurrency, err := intOrEnvWithEnv(
		fileCfg.WorkerMaxConcurrency,
		"worker_max_concurrency",
		env,
		"GHOST_WORKER_MAX_CONCURRENCY",
		defaultWorkerMaxConcurrency,
	)
	if err != nil {
		return WorkerConfig{}, err
	}
	maxFiles, err := intOrEnvWithEnv(fileCfg.WorkerMaxFiles, "worker_max_files", env, "GHOST_WORKER_MAX_FILES", defaultWorkerMaxFiles)
	if err != nil {
		return WorkerConfig{}, err
	}
	maxFileChunks, err := intOrEnvWithEnv(
		fileCfg.WorkerMaxFileChunks,
		"worker_max_file_chunks",
		env,
		"GHOST_WORKER_MAX_FILE_CHUNKS",
		defaultWorkerMaxFileChunks,
	)
	if err != nil {
		return WorkerConfig{}, err
	}
	return WorkerConfig{
		Model:          valueOrEnvWithEnv(fileCfg.WorkerModel, env, "GHOST_WORKER_MODEL", ""),
		MaxConcurrency: maxConcurrency,
		MaxFiles:       maxFiles,
		MaxFileChunks:  maxFileChunks,
	}, nil
}

func buildToolSelectorConfig(fileCfg bridgeFileConfig, env envSnapshot) (ToolSelectorConfig, error) {
	enabled, err := boolOrEnvWithEnv(fileCfg.ToolSelectorEnabled, env, "GHOST_TOOL_SELECTOR_ENABLED", false)
	if err != nil {
		return ToolSelectorConfig{}, err
	}
	timeoutMS, err := intOrEnvWithEnv(
		fileCfg.ToolSelectorTimeoutMS,
		"tool_selector_timeout_ms",
		env,
		"GHOST_TOOL_SELECTOR_TIMEOUT_MS",
		defaultToolSelectorTimeoutMS,
	)
	if err != nil {
		return ToolSelectorConfig{}, err
	}
	confidence, err := floatOrEnvWithEnv(
		fileCfg.ToolSelectorConfidence,
		"tool_selector_confidence",
		env,
		"GHOST_TOOL_SELECTOR_CONFIDENCE",
		defaultToolSelectorConfidence,
	)
	if err != nil {
		return ToolSelectorConfig{}, err
	}
	shadow, err := boolOrEnvWithEnv(fileCfg.ToolSelectorShadow, env, "GHOST_TOOL_SELECTOR_SHADOW", false)
	if err != nil {
		return ToolSelectorConfig{}, err
	}
	recentMsgs, err := intOrEnvWithEnv(
		fileCfg.ToolSelectorRecentMsgs,
		"tool_selector_recent_messages",
		env,
		"GHOST_TOOL_SELECTOR_RECENT_MESSAGES",
		defaultToolSelectorRecentMsgs,
	)
	if err != nil {
		return ToolSelectorConfig{}, err
	}
	allowlistOnly, err := boolOrEnvWithEnv(fileCfg.ToolAllowlistOnly, env, "GHOST_TOOL_ALLOWLIST_ONLY", false)
	if err != nil {
		return ToolSelectorConfig{}, err
	}
	return ToolSelectorConfig{
		Enabled:       enabled,
		Mode:          strings.ToLower(valueOrEnvWithEnv(fileCfg.ToolSelectorMode, env, "GHOST_TOOL_SELECTOR_MODE", "llm")),
		Model:         valueOrEnvWithEnv(fileCfg.ToolSelectorModel, env, "GHOST_TOOL_SELECTOR_MODEL", ""),
		TimeoutMS:     timeoutMS,
		Confidence:    confidence,
		Shadow:        shadow,
		RecentMsgs:    recentMsgs,
		AllowlistOnly: allowlistOnly,
		Allowlist:     toolNameListOrEnvWithEnv(fileCfg.ToolAllowlist, env, "GHOST_TOOL_ALLOWLIST"),
		Blocklist:     toolNameListOrEnvWithEnv(fileCfg.ToolBlocklist, env, "GHOST_TOOL_BLOCKLIST"),
	}, nil
}

func buildToolSearchConfig(fileCfg bridgeFileConfig, env envSnapshot) (ToolSearchConfig, error) {
	enabled, err := boolOrEnvWithEnv(fileCfg.ToolSearchEnabled, env, "GHOST_TOOL_SEARCH_ENABLED", false)
	if err != nil {
		return ToolSearchConfig{}, err
	}
	idleTurns, err := intOrEnvWithEnv(
		fileCfg.ToolSearchIdleTurns,
		"tool_search_idle_turns",
		env,
		"GHOST_TOOL_SEARCH_IDLE_TURNS",
		defaultToolSearchIdleTurns,
	)
	if err != nil {
		return ToolSearchConfig{}, err
	}
	return ToolSearchConfig{Enabled: enabled, IdleTurns: idleTurns}, nil
}

func buildMemoryAugmentationConfig(fileCfg bridgeFileConfig, env envSnapshot) (MemoryAugmentationConfig, error) {
	enabled, err := boolOrEnvWithEnv(fileCfg.MemoryAugmentationEnabled, env, "GHOST_MEMORY_AUGMENTATION_ENABLED", true)
	if err != nil {
		return MemoryAugmentationConfig{}, err
	}
	learningEnabled, err := boolOrEnvWithEnv(
		fileCfg.MemoryAugmentationLearningEnabled,
		env,
		"GHOST_MEMORY_AUGMENTATION_LEARNING_ENABLED",
		true,
	)
	if err != nil {
		return MemoryAugmentationConfig{}, err
	}
	recallEnabled, err := boolOrEnvWithEnv(
		fileCfg.MemoryAugmentationRecallEnabled,
		env,
		"GHOST_MEMORY_AUGMENTATION_RECALL_ENABLED",
		true,
	)
	if err != nil {
		return MemoryAugmentationConfig{}, err
	}
	maxRecallItems, err := intOrEnvWithEnv(
		fileCfg.MemoryAugmentationMaxRecallItems,
		"memory_augmentation_max_recall_items",
		env,
		"GHOST_MEMORY_AUGMENTATION_MAX_RECALL_ITEMS",
		defaultMemoryRecallItems,
	)
	if err != nil {
		return MemoryAugmentationConfig{}, err
	}
	minConfidence, err := floatOrEnvWithEnv(
		fileCfg.MemoryAugmentationMinConfidence,
		"memory_augmentation_min_confidence",
		env,
		"GHOST_MEMORY_AUGMENTATION_MIN_CONFIDENCE",
		defaultMemoryMinConfidence,
	)
	if err != nil {
		return MemoryAugmentationConfig{}, err
	}
	sessionScopeEnabled, err := boolOrEnvWithEnv(
		fileCfg.MemoryAugmentationSessionScopeEnabled,
		env,
		"GHOST_MEMORY_AUGMENTATION_SESSION_SCOPE_ENABLED",
		true,
	)
	if err != nil {
		return MemoryAugmentationConfig{}, err
	}
	userScopeEnabled, err := boolOrEnvWithEnv(
		fileCfg.MemoryAugmentationUserScopeEnabled,
		env,
		"GHOST_MEMORY_AUGMENTATION_USER_SCOPE_ENABLED",
		true,
	)
	if err != nil {
		return MemoryAugmentationConfig{}, err
	}
	return MemoryAugmentationConfig{
		Enabled:             enabled,
		LearningEnabled:     learningEnabled,
		RecallEnabled:       recallEnabled,
		MaxRecallItems:      maxRecallItems,
		MinConfidence:       minConfidence,
		SessionScopeEnabled: sessionScopeEnabled,
		UserScopeEnabled:    userScopeEnabled,
		LLMModel:            valueOrEnvWithEnv(fileCfg.MemoryAugmentationLLMModel, env, "GHOST_MEMORY_AUGMENTATION_LLM_MODEL", ""),
		UserScopeID:         valueOrEnvWithEnv(fileCfg.MemoryAugmentationUserScopeID, env, "GHOST_MEMORY_AUGMENTATION_USER_SCOPE_ID", defaultMemoryUserScopeID),
	}, nil
}

func finalizeLoadedConfig(cfg Config) (Config, error) {
	allowlist, blocklist, err := normalizeConfiguredToolLists(cfg.ToolSelector.Allowlist, cfg.ToolSelector.Blocklist)
	if err != nil {
		return Config{}, err
	}
	cfg.ToolSelector.Allowlist = allowlist
	cfg.ToolSelector.Blocklist = blocklist
	if cfg.ToolSelector.AllowlistOnly && len(cfg.ToolSelector.Allowlist) == 0 {
		return Config{}, errors.New("tool_allowlist_only requires a non-empty tool_allowlist")
	}
	if cfg.ToolSearch.IdleTurns <= 0 {
		return Config{}, errors.New("tool_search_idle_turns must be > 0")
	}
	if !cfg.MemoryAugmentation.SessionScopeEnabled && !cfg.MemoryAugmentation.UserScopeEnabled {
		cfg.MemoryAugmentation.RecallEnabled = false
		cfg.MemoryAugmentation.LearningEnabled = false
	}

	return cfg, nil
}

func resolvePromptsDir(fileCfg bridgeFileConfig, env envSnapshot) (string, error) {
	raw := valueOrEnvWithEnv(fileCfg.PromptsDir, env, "GHOST_PROMPTS_DIR", defaultPromptsDir)
	resolved, err := resolveUserPath(raw)
	if err != nil {
		return "", fmt.Errorf("resolve prompts_dir: %w", err)
	}
	return resolved, nil
}

func promptsCoreFiles(fileCfg bridgeFileConfig, env envSnapshot) []string {
	return promptPathListWithEnv(fileCfg.PromptsCoreFiles, env, "GHOST_PROMPTS_CORE_FILES")
}

func promptPathListWithEnv(raw []string, env envSnapshot, envName string) []string {
	if raw != nil {
		return normalizeConfiguredPathList(raw)
	}
	return normalizeConfiguredPathList(parseStringCSV(env.defaultValue(envName, "")))
}
