package config

import (
	"errors"
	"fmt"
	"strings"
)

// LoadConfig 从环境变量加载配置并做基础校验与归一化。
func LoadConfig() (Config, error) {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return Config{}, err
	}
	return Resolve(fileCfg, CurrentEnv())
}

// Resolve 把已读取的文件配置和环境快照解析成运行时配置。
func Resolve(fileCfg FileConfig, env Env) (Config, error) {
	return resolveConfig(bridgeFileConfig(fileCfg), env)
}

// loadConfigWithRuntime 在 runtimeConfig 基础上补齐环境默认值与执行期约束。
func loadConfigWithRuntime(runtime runtimeConfig) (Config, error) {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return Config{}, err
	}
	return resolveConfigWithRuntime(fileCfg, CurrentEnv(), runtime)
}

func resolveConfig(fileCfg bridgeFileConfig, env Env) (Config, error) {
	runtime, err := resolveRuntimeConfig(fileCfg, env)
	if err != nil {
		return Config{}, err
	}
	return resolveConfigWithRuntime(fileCfg, env, runtime)
}

func resolveConfigWithRuntime(fileCfg bridgeFileConfig, env Env, runtime runtimeConfig) (Config, error) {
	runtime = normalizeRuntimeConfig(runtime)
	if err := validateRuntimeForExecution(runtime); err != nil {
		return Config{}, err
	}
	headers, promptsDir, err := loadConfigEnvDetails(fileCfg, env)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Provider:                buildProviderConfig(runtime, fileCfg, env, headers),
		RSS:                     buildRSSConfig(fileCfg, env),
		Worker:                  buildWorkerConfig(fileCfg, env),
		GraphQL:                 runtime.GraphQL,
		ToolSelector:            buildToolSelectorConfig(fileCfg, env),
		ToolSearch:              buildToolSearchConfig(fileCfg, env),
		MemoryAugmentation:      buildMemoryAugmentationConfig(fileCfg, env),
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
		ProMaxIterations:      intOrEnvWithEnv(fileCfg.ProMaxIterations, env, "GHOST_PRO_MAX_ITERATIONS", defaultProMaxIterations),
		MaxTurns:              intOrEnvWithEnv(fileCfg.MaxTurns, env, "GHOST_MAX_TURNS", defaultMaxTurns),
	}
	return finalizeLoadedConfig(cfg)
}

func loadConfigEnvDetails(fileCfg bridgeFileConfig, env Env) (map[string]string, string, error) {
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

func buildProviderConfig(runtime runtimeConfig, fileCfg bridgeFileConfig, env Env, headers map[string]string) ProviderConfig {
	return ProviderConfig{
		Type:                       runtime.Provider,
		APIKey:                     runtime.APIKey,
		BaseURL:                    runtime.BaseURL,
		Model:                      runtime.Model,
		Headers:                    headers,
		AnthropicVersion:           valueOrEnvWithEnv(fileCfg.AnthropicVersion, env, "GHOST_ANTHROPIC_VERSION", defaultAnthropicVersion),
		AnthropicMaxTokens:         intOrEnvWithEnv(fileCfg.AnthropicMaxTokens, env, "GHOST_ANTHROPIC_MAX_TOKENS", defaultAnthropicMaxTokens),
		ContextWindowTokens:        runtime.ContextWindowTokens,
		ResponseReserveTokens:      runtime.ResponseReserveTokens,
		ModelContextWindowTokens:   cloneModelTokenOverrides(runtime.ModelContextWindowTokens),
		ModelResponseReserveTokens: cloneModelTokenOverrides(runtime.ModelResponseReserveTokens),
	}
}

func buildRSSConfig(fileCfg bridgeFileConfig, env Env) RSSConfig {
	return RSSConfig{
		FeedsPath:           resolveRSSFeedsPath(fileCfg, env),
		InboxPath:           resolveRSSInboxPath(fileCfg, env),
		BriefingsPath:       resolveRSSBriefingsPath(fileCfg, env),
		ReportsPath:         resolveRSSReportsPath(fileCfg, env),
		PollEnabled:         boolOrEnvWithEnv(fileCfg.RSSPollEnabled, env, "GHOST_RSS_POLL_ENABLED", true),
		PollInterval:        durationOrEnvWithEnv(fileCfg.RSSPollInterval, env, "GHOST_RSS_POLL_INTERVAL", defaultRSSPollInterval),
		PollMaxItemsPerFeed: intOrEnvWithEnv(fileCfg.RSSPollMaxItemsPerFeed, env, "GHOST_RSS_POLL_MAX_ITEMS_PER_FEED", defaultRSSPollMaxItemsPerFeed),
		AIBatchSize:         intOrEnvWithEnv(fileCfg.RSSAIBatchSize, env, "GHOST_RSS_AI_BATCH_SIZE", defaultRSSAIBatchSize),
		BriefingEnabled:     boolOrEnvWithEnv(fileCfg.RSSBriefingEnabled, env, "GHOST_RSS_BRIEFING_ENABLED", true),
		BriefingInterval:    durationOrEnvWithEnv(fileCfg.RSSBriefingInterval, env, "GHOST_RSS_BRIEFING_INTERVAL", defaultRSSBriefingInterval),
	}
}

func buildWorkerConfig(fileCfg bridgeFileConfig, env Env) WorkerConfig {
	return WorkerConfig{
		Model:          valueOrEnvWithEnv(fileCfg.WorkerModel, env, "GHOST_WORKER_MODEL", ""),
		MaxConcurrency: intOrEnvWithEnv(fileCfg.WorkerMaxConcurrency, env, "GHOST_WORKER_MAX_CONCURRENCY", defaultWorkerMaxConcurrency),
		MaxFiles:       intOrEnvWithEnv(fileCfg.WorkerMaxFiles, env, "GHOST_WORKER_MAX_FILES", defaultWorkerMaxFiles),
		MaxFileChunks:  intOrEnvWithEnv(fileCfg.WorkerMaxFileChunks, env, "GHOST_WORKER_MAX_FILE_CHUNKS", defaultWorkerMaxFileChunks),
	}
}

func buildToolSelectorConfig(fileCfg bridgeFileConfig, env Env) ToolSelectorConfig {
	return ToolSelectorConfig{
		Enabled:       boolOrEnvWithEnv(fileCfg.ToolSelectorEnabled, env, "GHOST_TOOL_SELECTOR_ENABLED", false),
		Mode:          strings.ToLower(valueOrEnvWithEnv(fileCfg.ToolSelectorMode, env, "GHOST_TOOL_SELECTOR_MODE", "llm")),
		Model:         valueOrEnvWithEnv(fileCfg.ToolSelectorModel, env, "GHOST_TOOL_SELECTOR_MODEL", ""),
		TimeoutMS:     intOrEnvWithEnv(fileCfg.ToolSelectorTimeoutMS, env, "GHOST_TOOL_SELECTOR_TIMEOUT_MS", defaultToolSelectorTimeoutMS),
		Confidence:    floatOrEnvWithEnv(fileCfg.ToolSelectorConfidence, env, "GHOST_TOOL_SELECTOR_CONFIDENCE", defaultToolSelectorConfidence),
		Shadow:        boolOrEnvWithEnv(fileCfg.ToolSelectorShadow, env, "GHOST_TOOL_SELECTOR_SHADOW", false),
		RecentMsgs:    intOrEnvWithEnv(fileCfg.ToolSelectorRecentMsgs, env, "GHOST_TOOL_SELECTOR_RECENT_MESSAGES", defaultToolSelectorRecentMsgs),
		AllowlistOnly: boolOrEnvWithEnv(fileCfg.ToolAllowlistOnly, env, "GHOST_TOOL_ALLOWLIST_ONLY", false),
		Allowlist:     toolNameListOrEnvWithEnv(fileCfg.ToolAllowlist, env, "GHOST_TOOL_ALLOWLIST"),
		Blocklist:     toolNameListOrEnvWithEnv(fileCfg.ToolBlocklist, env, "GHOST_TOOL_BLOCKLIST"),
	}
}

func buildToolSearchConfig(fileCfg bridgeFileConfig, env Env) ToolSearchConfig {
	return ToolSearchConfig{
		Enabled:   boolOrEnvWithEnv(fileCfg.ToolSearchEnabled, env, "GHOST_TOOL_SEARCH_ENABLED", false),
		IdleTurns: intOrEnvWithEnv(fileCfg.ToolSearchIdleTurns, env, "GHOST_TOOL_SEARCH_IDLE_TURNS", defaultToolSearchIdleTurns),
	}
}

func buildMemoryAugmentationConfig(fileCfg bridgeFileConfig, env Env) MemoryAugmentationConfig {
	return MemoryAugmentationConfig{
		Enabled:             boolOrEnvWithEnv(fileCfg.MemoryAugmentationEnabled, env, "GHOST_MEMORY_AUGMENTATION_ENABLED", true),
		LearningEnabled:     boolOrEnvWithEnv(fileCfg.MemoryAugmentationLearningEnabled, env, "GHOST_MEMORY_AUGMENTATION_LEARNING_ENABLED", true),
		RecallEnabled:       boolOrEnvWithEnv(fileCfg.MemoryAugmentationRecallEnabled, env, "GHOST_MEMORY_AUGMENTATION_RECALL_ENABLED", true),
		MaxRecallItems:      intOrEnvWithEnv(fileCfg.MemoryAugmentationMaxRecallItems, env, "GHOST_MEMORY_AUGMENTATION_MAX_RECALL_ITEMS", defaultMemoryRecallItems),
		MinConfidence:       floatOrEnvWithEnv(fileCfg.MemoryAugmentationMinConfidence, env, "GHOST_MEMORY_AUGMENTATION_MIN_CONFIDENCE", defaultMemoryMinConfidence),
		SessionScopeEnabled: boolOrEnvWithEnv(fileCfg.MemoryAugmentationSessionScopeEnabled, env, "GHOST_MEMORY_AUGMENTATION_SESSION_SCOPE_ENABLED", true),
		UserScopeEnabled:    boolOrEnvWithEnv(fileCfg.MemoryAugmentationUserScopeEnabled, env, "GHOST_MEMORY_AUGMENTATION_USER_SCOPE_ENABLED", true),
		LLMModel:            valueOrEnvWithEnv(fileCfg.MemoryAugmentationLLMModel, env, "GHOST_MEMORY_AUGMENTATION_LLM_MODEL", ""),
		UserScopeID:         valueOrEnvWithEnv(fileCfg.MemoryAugmentationUserScopeID, env, "GHOST_MEMORY_AUGMENTATION_USER_SCOPE_ID", defaultMemoryUserScopeID),
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

func resolvePromptsDir(fileCfg bridgeFileConfig, env Env) (string, error) {
	raw := valueOrEnvWithEnv(fileCfg.PromptsDir, env, "GHOST_PROMPTS_DIR", defaultPromptsDir)
	resolved, err := resolveUserPath(raw)
	if err != nil {
		return "", fmt.Errorf("resolve prompts_dir: %w", err)
	}
	return resolved, nil
}

func promptsCoreFiles(fileCfg bridgeFileConfig, env Env) []string {
	return promptPathListWithEnv(fileCfg.PromptsCoreFiles, env, "GHOST_PROMPTS_CORE_FILES")
}

func promptPathListWithEnv(raw []string, env Env, envName string) []string {
	if raw != nil {
		return normalizeConfiguredPathList(raw)
	}
	return normalizeConfiguredPathList(parseStringCSV(env.defaultValue(envName, "")))
}
