package config

import (
	"errors"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
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
	sections, err := resolveConfigSections(fileCfg, env, runtime)
	if err != nil {
		return Config{}, err
	}
	cfg := composeConfig(fileCfg, env, runtime, sections)
	return finalizeLoadedConfig(cfg)
}

type configSections struct {
	Provider         ProviderConfig
	RSS              RSSConfig
	Worker           WorkerConfig
	Task             TaskConfig
	ToolSelector     ToolSelectorConfig
	ToolSearch       ToolSearchConfig
	PromptsDir       string
	ProMaxIterations int
	MaxTurns         int
}

func resolveConfigSections(fileCfg bridgeFileConfig, env envSnapshot, runtime runtimeConfig) (configSections, error) {
	headers, promptsDir, err := loadConfigEnvDetails(fileCfg, env)
	if err != nil {
		return configSections{}, err
	}
	provider, err := buildProviderConfig(runtime, fileCfg, env, headers)
	if err != nil {
		return configSections{}, err
	}
	features, err := resolveRuntimeFeatureSections(fileCfg, env)
	if err != nil {
		return configSections{}, err
	}
	proMaxIterations, maxTurns, err := resolveIterationLimits(fileCfg, env)
	if err != nil {
		return configSections{}, err
	}
	return configSections{
		Provider:         provider,
		RSS:              features.RSS,
		Worker:           features.Worker,
		Task:             features.Task,
		ToolSelector:     features.ToolSelector,
		ToolSearch:       features.ToolSearch,
		PromptsDir:       promptsDir,
		ProMaxIterations: proMaxIterations,
		MaxTurns:         maxTurns,
	}, nil
}

type runtimeFeatureSections struct {
	RSS          RSSConfig
	Worker       WorkerConfig
	Task         TaskConfig
	ToolSelector ToolSelectorConfig
	ToolSearch   ToolSearchConfig
}

func resolveRuntimeFeatureSections(fileCfg bridgeFileConfig, env envSnapshot) (runtimeFeatureSections, error) {
	rss, err := buildRSSConfig(fileCfg, env)
	if err != nil {
		return runtimeFeatureSections{}, err
	}
	worker, err := buildWorkerConfig(fileCfg, env)
	if err != nil {
		return runtimeFeatureSections{}, err
	}
	task, err := buildTaskConfig(fileCfg, env)
	if err != nil {
		return runtimeFeatureSections{}, err
	}
	toolSelector, err := buildToolSelectorConfig(fileCfg, env)
	if err != nil {
		return runtimeFeatureSections{}, err
	}
	toolSearch, err := buildToolSearchConfig(fileCfg, env)
	if err != nil {
		return runtimeFeatureSections{}, err
	}
	return runtimeFeatureSections{
		RSS:          rss,
		Worker:       worker,
		Task:         task,
		ToolSelector: toolSelector,
		ToolSearch:   toolSearch,
	}, nil
}

func resolveIterationLimits(fileCfg bridgeFileConfig, env envSnapshot) (int, int, error) {
	proMaxIterations, err := intOrEnvWithEnv(
		fileCfg.ProMaxIterations,
		"pro_max_iterations",
		env,
		"GHOST_PRO_MAX_ITERATIONS",
		defaultProMaxIterations,
	)
	if err != nil {
		return 0, 0, err
	}
	maxTurns, err := intOrEnvWithEnv(fileCfg.MaxTurns, "max_turns", env, "GHOST_MAX_TURNS", defaultMaxTurns)
	if err != nil {
		return 0, 0, err
	}
	return proMaxIterations, maxTurns, nil
}

func composeConfig(fileCfg bridgeFileConfig, env envSnapshot, runtime runtimeConfig, sections configSections) Config {
	return Config{
		Provider:                   sections.Provider,
		RSS:                        sections.RSS,
		Worker:                     sections.Worker,
		GraphQL:                    runtime.GraphQL,
		ToolSelector:               sections.ToolSelector,
		ToolSearch:                 sections.ToolSearch,
		NativePersistent:           runtime.NativePersistent,
		NativeBinaryPath:           resolveNativeBinaryPath(fileCfg, env),
		NativeBinaryRoots:          resolveNativeBinaryRoots(fileCfg, env),
		NativeBinaryCandidates:     resolveNativeBinaryCandidates(fileCfg, env),
		NativeAllowedReadPaths:     resolveNativeAllowedReadPaths(fileCfg, env),
		NativeAllowedWritePaths:    resolveNativeAllowedWritePaths(fileCfg, env),
		ProjectRoot:                runtime.ProjectRoot,
		Task:                       sections.Task,
		ChatPath:                   runtime.ChatPath,
		ResponseOptions:            llm.CloneResponseOptions(runtime.ResponseOptions),
		CodexStatelessRetryEnabled: runtime.CodexStatelessRetryEnabled,
		PromptsPath:                valueOrEnvWithEnv(fileCfg.PromptsPath, env, "GHOST_PROMPTS_PATH", defaultPromptsPath),
		PromptsDir:                 sections.PromptsDir,
		PromptsCoreFiles:           promptsCoreFiles(fileCfg, env),
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
		SessionsPath:               resolveSessionsPath(fileCfg, env),
		WebSearchTavilyURL:         runtime.WebSearchTavilyURL,
		WebSearchExaURL:            runtime.WebSearchExaURL,
		WebSearchTavilyAPIKey:      runtime.WebSearchTavilyAPIKey,
		WebSearchExaAPIKey:         runtime.WebSearchExaAPIKey,
		SessionHumanLogFullEnabled: runtime.SessionHumanLogFullEnabled,
		AssistantMarkdownEnabled:   runtime.AssistantMarkdownEnabled,
		MemoryModeEnabled:          runtime.MemoryModeEnabled,
		WebRooterEnabled:           runtime.WebRooterEnabled,
		WebRooterBaseURL:           runtime.WebRooterBaseURL,
		WebRooterAPIToken:          runtime.WebRooterAPIToken,
		WebRooterTimeoutMS:         runtime.WebRooterTimeoutMS,
		ProMaxIterations:           sections.ProMaxIterations,
		MaxTurns:                   sections.MaxTurns,
	}
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
	if strings.TrimSpace(cfg.WebRooterBaseURL) == "" {
		return Config{}, errors.New("web_rooter_base_url must not be empty")
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
