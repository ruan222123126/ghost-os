package config

import (
	"errors"
	"fmt"
	"strings"
)

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
	headers, promptsDir, err := loadConfigEnvDetails(fileCfg)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Provider:                buildProviderConfig(runtime, fileCfg, headers),
		RSS:                     buildRSSConfig(),
		Worker:                  buildWorkerConfig(fileCfg),
		ToolSelector:            buildToolSelectorConfig(fileCfg),
		ToolSearch:              buildToolSearchConfig(fileCfg),
		MemoryAugmentation:      buildMemoryAugmentationConfig(fileCfg),
		NativePersistent:        runtime.NativePersistent,
		NativeBinaryPath:        nativeBinaryPathFromEnv(),
		NativeBinaryRoots:       nativeBinaryRootsFromEnv(),
		NativeBinaryCandidates:  nativeBinaryCandidatesFromEnv(),
		NativeAllowedReadPaths:  nativeAllowedReadPathsFromEnv(),
		NativeAllowedWritePaths: nativeAllowedWritePathsFromEnv(),
		ProjectRoot:             runtime.ProjectRoot,
		ChatPath:                runtime.ChatPath,
		PromptsPath:             valueOrEnv(fileCfg.PromptsPath, "GHOST_PROMPTS_PATH", defaultPromptsPath),
		PromptsDir:              promptsDir,
		PromptsCoreFiles:        promptsCoreFiles(fileCfg),
		SessionsPath:            sessionsPathFromEnv(),
		WebSearchTavilyAPIKey:   runtime.WebSearchTavilyAPIKey,
		WebSearchExaAPIKey:      runtime.WebSearchExaAPIKey,
		ProMaxIterations:        intOrEnv(fileCfg.ProMaxIterations, "GHOST_PRO_MAX_ITERATIONS", defaultProMaxIterations),
		MaxTurns:                intOrEnv(fileCfg.MaxTurns, "GHOST_MAX_TURNS", defaultMaxTurns),
	}
	return finalizeLoadedConfig(cfg)
}

func loadConfigEnvDetails(fileCfg bridgeFileConfig) (map[string]string, string, error) {
	headers, err := headersOrEnv(fileCfg.ProviderHeaders)
	if err != nil {
		return nil, "", err
	}
	promptsDir, err := resolvePromptsDir(fileCfg)
	if err != nil {
		return nil, "", err
	}
	return headers, promptsDir, nil
}

func buildProviderConfig(runtime runtimeConfig, fileCfg bridgeFileConfig, headers map[string]string) ProviderConfig {
	return ProviderConfig{
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
	}
}

func buildRSSConfig() RSSConfig {
	return RSSConfig{
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
	}
}

func buildWorkerConfig(fileCfg bridgeFileConfig) WorkerConfig {
	return WorkerConfig{
		Model:          valueOrEnv(fileCfg.WorkerModel, "GHOST_WORKER_MODEL", ""),
		MaxConcurrency: intOrEnv(fileCfg.WorkerMaxConcurrency, "GHOST_WORKER_MAX_CONCURRENCY", defaultWorkerMaxConcurrency),
		MaxFiles:       intOrEnv(fileCfg.WorkerMaxFiles, "GHOST_WORKER_MAX_FILES", defaultWorkerMaxFiles),
		MaxFileChunks:  intOrEnv(fileCfg.WorkerMaxFileChunks, "GHOST_WORKER_MAX_FILE_CHUNKS", defaultWorkerMaxFileChunks),
	}
}

func buildToolSelectorConfig(fileCfg bridgeFileConfig) ToolSelectorConfig {
	return ToolSelectorConfig{
		Enabled:       boolOrEnv(fileCfg.ToolSelectorEnabled, "GHOST_TOOL_SELECTOR_ENABLED", false),
		Mode:          strings.ToLower(valueOrEnv(fileCfg.ToolSelectorMode, "GHOST_TOOL_SELECTOR_MODE", "llm")),
		Model:         valueOrEnv(fileCfg.ToolSelectorModel, "GHOST_TOOL_SELECTOR_MODEL", ""),
		TimeoutMS:     intOrEnv(fileCfg.ToolSelectorTimeoutMS, "GHOST_TOOL_SELECTOR_TIMEOUT_MS", defaultToolSelectorTimeoutMS),
		Confidence:    floatOrEnv(fileCfg.ToolSelectorConfidence, "GHOST_TOOL_SELECTOR_CONFIDENCE", defaultToolSelectorConfidence),
		Shadow:        boolOrEnv(fileCfg.ToolSelectorShadow, "GHOST_TOOL_SELECTOR_SHADOW", false),
		RecentMsgs:    intOrEnv(fileCfg.ToolSelectorRecentMsgs, "GHOST_TOOL_SELECTOR_RECENT_MESSAGES", defaultToolSelectorRecentMsgs),
		AllowlistOnly: boolOrEnv(fileCfg.ToolAllowlistOnly, "GHOST_TOOL_ALLOWLIST_ONLY", false),
		Allowlist:     toolNameListOrEnv(fileCfg.ToolAllowlist, "GHOST_TOOL_ALLOWLIST"),
		Blocklist:     toolNameListOrEnv(fileCfg.ToolBlocklist, "GHOST_TOOL_BLOCKLIST"),
	}
}

func buildToolSearchConfig(fileCfg bridgeFileConfig) ToolSearchConfig {
	return ToolSearchConfig{
		Enabled:   boolOrEnv(fileCfg.ToolSearchEnabled, "GHOST_TOOL_SEARCH_ENABLED", false),
		IdleTurns: intOrEnv(fileCfg.ToolSearchIdleTurns, "GHOST_TOOL_SEARCH_IDLE_TURNS", defaultToolSearchIdleTurns),
	}
}

func buildMemoryAugmentationConfig(fileCfg bridgeFileConfig) MemoryAugmentationConfig {
	return MemoryAugmentationConfig{
		Enabled:             boolOrEnv(fileCfg.MemoryAugmentationEnabled, "GHOST_MEMORY_AUGMENTATION_ENABLED", true),
		LearningEnabled:     boolOrEnv(fileCfg.MemoryAugmentationLearningEnabled, "GHOST_MEMORY_AUGMENTATION_LEARNING_ENABLED", true),
		RecallEnabled:       boolOrEnv(fileCfg.MemoryAugmentationRecallEnabled, "GHOST_MEMORY_AUGMENTATION_RECALL_ENABLED", true),
		MaxRecallItems:      intOrEnv(fileCfg.MemoryAugmentationMaxRecallItems, "GHOST_MEMORY_AUGMENTATION_MAX_RECALL_ITEMS", defaultMemoryRecallItems),
		MinConfidence:       floatOrEnv(fileCfg.MemoryAugmentationMinConfidence, "GHOST_MEMORY_AUGMENTATION_MIN_CONFIDENCE", defaultMemoryMinConfidence),
		SessionScopeEnabled: boolOrEnv(fileCfg.MemoryAugmentationSessionScopeEnabled, "GHOST_MEMORY_AUGMENTATION_SESSION_SCOPE_ENABLED", true),
		UserScopeEnabled:    boolOrEnv(fileCfg.MemoryAugmentationUserScopeEnabled, "GHOST_MEMORY_AUGMENTATION_USER_SCOPE_ENABLED", true),
		LLMModel:            valueOrEnv(fileCfg.MemoryAugmentationLLMModel, "GHOST_MEMORY_AUGMENTATION_LLM_MODEL", ""),
		UserScopeID:         valueOrEnv(fileCfg.MemoryAugmentationUserScopeID, "GHOST_MEMORY_AUGMENTATION_USER_SCOPE_ID", defaultMemoryUserScopeID),
	}
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

func resolvePromptsDir(fileCfg bridgeFileConfig) (string, error) {
	raw := valueOrEnv(fileCfg.PromptsDir, "GHOST_PROMPTS_DIR", defaultPromptsDir)
	resolved, err := resolveUserPath(raw)
	if err != nil {
		return "", fmt.Errorf("resolve prompts_dir: %w", err)
	}
	return resolved, nil
}

func promptsCoreFiles(fileCfg bridgeFileConfig) []string {
	if fileCfg.PromptsCoreFiles != nil {
		return normalizeConfiguredPathList(fileCfg.PromptsCoreFiles)
	}
	return normalizeConfiguredPathList(parseStringCSV(getenvDefault("GHOST_PROMPTS_CORE_FILES", "")))
}
