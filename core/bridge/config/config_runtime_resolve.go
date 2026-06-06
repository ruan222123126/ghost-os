package config

import "ghost-os/bridge/llm"

func runtimeConfigFromEnv() (runtimeConfig, error) {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return runtimeConfig{}, err
	}
	return resolveRuntimeConfig(fileCfg, currentEnv())
}

func resolveRuntimeConfig(fileCfg bridgeFileConfig, env envSnapshot) (runtimeConfig, error) {
	fallback, err := runtimeFallbackFromEnv(env)
	if err != nil {
		return runtimeConfig{}, err
	}
	return resolveRuntimeConfigWithFallback(fileCfg, fallback)
}

func runtimeFallbackFromEnv(env envSnapshot) (runtimeConfig, error) {
	settings, err := resolveRuntimeFallbackSettings(env)
	if err != nil {
		return runtimeConfig{}, err
	}
	return normalizeRuntimeConfig(runtimeConfig{
		ProviderName:                   env.defaultValue("GHOST_PROVIDER", string(defaultProvider)),
		APIKey:                         env.defaultValue("GHOST_API_KEY", ""),
		BaseURL:                        env.defaultValue("GHOST_BASE_URL", ""),
		Model:                          env.defaultValue("GHOST_MODEL", ""),
		ChatPath:                       env.defaultValue("GHOST_CHAT_PATH", ""),
		ResponseOptions:                settings.ResponseOptions,
		CodexStatelessRetryEnabled:     settings.CodexStatelessRetryEnabled,
		NativePersistent:               settings.NativePersistent,
		ProjectRoot:                    env.defaultValue("GHOST_PROJECT_ROOT", ""),
		MaxTurns:                       settings.MaxTurns,
		TaskExecutionTimeoutMS:         settings.TaskExecutionTimeoutMS,
		RelayDefaultStopPolicy:         settings.RelayDefaultStopPolicy,
		RelayDefaultMaxRounds:          settings.RelayDefaultMaxRounds,
		RelayDefaultExecutionTimeoutMS: settings.RelayDefaultExecutionTimeoutMS,
		LLMCompletionRetryCount:        settings.LLMCompletionRetryCount,
		LLMCompletionRetryIntervalMS:   settings.LLMCompletionRetryIntervalMS,
		ModelSelectionEnabled:          settings.ModelSelectionEnabled,
		WebSearchTavilyURL:             settings.WebSearch.TavilyURL,
		WebSearchExaURL:                settings.WebSearch.ExaURL,
		WebSearchTavilyAPIKey:          settings.WebSearch.TavilyAPIKey,
		WebSearchExaAPIKey:             settings.WebSearch.ExaAPIKey,
		SessionHumanLogFullEnabled:     settings.SessionHumanLogFullEnabled,
		SessionSystemPromptVisible:     settings.SessionSystemPromptVisible,
		AssistantMarkdownEnabled:       settings.AssistantMarkdownEnabled,
		ToolCallCompactOutputEnabled:   settings.ToolCallCompactOutputEnabled,
		MemoryModeEnabled:              settings.MemoryModeEnabled,
		MicrocompactEnabled:            settings.MicrocompactEnabled,
		SessionTitleMode:               settings.SessionTitleMode,
	}), nil
}

type runtimeFallbackSettings struct {
	WebSearch                      webSearchSettings
	ResponseOptions                llm.ResponseOptions
	CodexStatelessRetryEnabled     bool
	NativePersistent               bool
	MaxTurns                       int
	TaskExecutionTimeoutMS         int
	RelayDefaultStopPolicy         string
	RelayDefaultMaxRounds          int
	RelayDefaultExecutionTimeoutMS int
	LLMCompletionRetryCount        int
	LLMCompletionRetryIntervalMS   int
	ModelSelectionEnabled          bool
	SessionHumanLogFullEnabled     bool
	SessionSystemPromptVisible     bool
	AssistantMarkdownEnabled       bool
	ToolCallCompactOutputEnabled   bool
	MemoryModeEnabled              bool
	MicrocompactEnabled            bool
	SessionTitleMode               string
}

func resolveRuntimeFallbackSettings(env envSnapshot) (runtimeFallbackSettings, error) {
	if err := validateNoRemovedGraphQLEnv(env); err != nil {
		return runtimeFallbackSettings{}, err
	}
	if err := validateNoRemovedProEnv(env); err != nil {
		return runtimeFallbackSettings{}, err
	}
	responseOptions, err := responseOptionsFromEnv(env)
	if err != nil {
		return runtimeFallbackSettings{}, err
	}
	codexRetryEnabled, nativePersistent, sessionHumanLogFullEnabled, err := resolveRuntimeFallbackFlags(env)
	if err != nil {
		return runtimeFallbackSettings{}, err
	}
	maxTurns, err := parsePositiveIntValue(env.value("GHOST_MAX_TURNS"), "GHOST_MAX_TURNS", defaultMaxTurns)
	if err != nil {
		return runtimeFallbackSettings{}, err
	}
	taskExecutionTimeoutMS, err := parsePositiveIntValue(
		env.value("GHOST_TASK_EXECUTION_TIMEOUT_MS"),
		"GHOST_TASK_EXECUTION_TIMEOUT_MS",
		defaultTaskExecutionTimeoutMS,
	)
	if err != nil {
		return runtimeFallbackSettings{}, err
	}
	relayDefaults, err := defaultRelaySettings()
	if err != nil {
		return runtimeFallbackSettings{}, err
	}
	retryCount, err := parseNonNegativeIntValue(
		env.value("GHOST_LLM_COMPLETION_RETRY_COUNT"),
		"GHOST_LLM_COMPLETION_RETRY_COUNT",
		defaultLLMCompletionRetryCount,
	)
	if err != nil {
		return runtimeFallbackSettings{}, err
	}
	retryIntervalMS, err := parseNonNegativeIntValue(
		env.value("GHOST_LLM_COMPLETION_RETRY_INTERVAL_MS"),
		"GHOST_LLM_COMPLETION_RETRY_INTERVAL_MS",
		defaultLLMCompletionRetryIntervalMS,
	)
	if err != nil {
		return runtimeFallbackSettings{}, err
	}
	return runtimeFallbackSettings{
		WebSearch:                      webSearchSettingsFromEnv(env),
		ResponseOptions:                responseOptions,
		CodexStatelessRetryEnabled:     codexRetryEnabled,
		NativePersistent:               nativePersistent,
		MaxTurns:                       maxTurns,
		TaskExecutionTimeoutMS:         taskExecutionTimeoutMS,
		RelayDefaultStopPolicy:         relayDefaults.stopPolicy,
		RelayDefaultMaxRounds:          relayDefaults.maxRounds,
		RelayDefaultExecutionTimeoutMS: relayDefaults.executionTimeoutMS,
		LLMCompletionRetryCount:        retryCount,
		LLMCompletionRetryIntervalMS:   retryIntervalMS,
		ModelSelectionEnabled:          defaultModelSelectionEnabled,
		SessionHumanLogFullEnabled:     sessionHumanLogFullEnabled,
		SessionSystemPromptVisible:     defaultSessionSystemPromptVisible,
		AssistantMarkdownEnabled:       defaultAssistantMarkdownEnabled,
		ToolCallCompactOutputEnabled:   defaultToolCallCompactOutputEnabled,
		MemoryModeEnabled:              defaultMemoryModeEnabled,
		MicrocompactEnabled:            defaultMicrocompactEnabled,
		SessionTitleMode:               defaultSessionTitleMode,
	}, nil
}

func resolveRuntimeFallbackFlags(env envSnapshot) (bool, bool, bool, error) {
	codexRetryEnabled, err := parseBoolValue(
		env.value("GHOST_CODEX_STATELESS_RETRY_ENABLED"),
		"GHOST_CODEX_STATELESS_RETRY_ENABLED",
		false,
	)
	if err != nil {
		return false, false, false, err
	}
	if err := validateRuntimeToolAllowlistOnlyEnv(env); err != nil {
		return false, false, false, err
	}
	nativePersistent, err := resolveNativePersistent(nil, env)
	if err != nil {
		return false, false, false, err
	}
	sessionHumanLogFullEnabled, err := parseBoolValue(
		env.value("GHOST_SESSION_HUMAN_LOG_FULL_ENABLED"),
		"GHOST_SESSION_HUMAN_LOG_FULL_ENABLED",
		false,
	)
	if err != nil {
		return false, false, false, err
	}
	return codexRetryEnabled, nativePersistent, sessionHumanLogFullEnabled, nil
}

func validateRuntimeToolAllowlistOnlyEnv(env envSnapshot) error {
	_, err := parseBoolValue(env.value("GHOST_TOOL_ALLOWLIST_ONLY"), "GHOST_TOOL_ALLOWLIST_ONLY", false)
	return err
}

// resolveRuntimeConfigWithFallback folds file overrides onto an existing
// runtime snapshot. Store patch flows use this explicit exception path.
func resolveRuntimeConfigWithFallback(fileCfg bridgeFileConfig, fallback runtimeConfig) (runtimeConfig, error) {
	normalizedFileCfg, err := normalizeBridgeFileConfigForWrite(fileCfg)
	if err != nil {
		return runtimeConfig{}, err
	}
	fallback = normalizeRuntimeConfig(fallback)
	settings, err := resolveRuntimeFileSettings(normalizedFileCfg, fallback)
	if err != nil {
		return runtimeConfig{}, err
	}
	buildInput := runtimeConfigBuildInput{
		FileCfg:  normalizedFileCfg,
		Fallback: fallback,
		Settings: settings,
	}
	if len(settings.Providers) > 0 {
		return runtimeConfigWithProviders(buildInput, settings.Providers), nil
	}
	return runtimeConfigWithoutProviders(buildInput), nil
}

type runtimeFileSettings struct {
	WebSearch                      webSearchSettings
	ResponseOptions                llm.ResponseOptions
	Providers                      []providerConfig
	CodexStatelessRetryEnabled     bool
	MaxTurns                       int
	TaskExecutionTimeoutMS         int
	RelayDefaultStopPolicy         string
	RelayDefaultMaxRounds          int
	RelayDefaultExecutionTimeoutMS int
	LLMCompletionRetryCount        int
	LLMCompletionRetryIntervalMS   int
	ModelSelectionEnabled          bool
	SessionHumanLogFullEnabled     bool
	SessionSystemPromptVisible     bool
	AssistantMarkdownEnabled       bool
	ToolCallCompactOutputEnabled   bool
	MemoryModeEnabled              bool
	MicrocompactEnabled            bool
	SessionTitleMode               string
}

type runtimeConfigBuildInput struct {
	FileCfg  bridgeFileConfig
	Fallback runtimeConfig
	Settings runtimeFileSettings
}

func resolveRuntimeFileSettings(fileCfg bridgeFileConfig, fallback runtimeConfig) (runtimeFileSettings, error) {
	responseOptions, err := fileResponseOptions(fileCfg, fallback.ResponseOptions)
	if err != nil {
		return runtimeFileSettings{}, err
	}
	maxTurns, err := resolveRuntimeMaxTurns(fileCfg, fallback)
	if err != nil {
		return runtimeFileSettings{}, err
	}
	taskExecutionTimeoutMS, err := resolveRuntimeTaskExecutionTimeoutMS(fileCfg, fallback)
	if err != nil {
		return runtimeFileSettings{}, err
	}
	relayDefaults, err := resolveRuntimeRelayDefaults(fileCfg, fallback)
	if err != nil {
		return runtimeFileSettings{}, err
	}
	retryCount, err := resolveRuntimeLLMCompletionRetryCount(fileCfg, fallback)
	if err != nil {
		return runtimeFileSettings{}, err
	}
	retryIntervalMS, err := resolveRuntimeLLMCompletionRetryIntervalMS(fileCfg, fallback)
	if err != nil {
		return runtimeFileSettings{}, err
	}
	sessionTitleMode, err := resolveRuntimeSessionTitleMode(fileCfg, fallback)
	if err != nil {
		return runtimeFileSettings{}, err
	}
	return runtimeFileSettings{
		WebSearch: fileWebSearchSettings(fileCfg, webSearchSettings{
			TavilyURL:    fallback.WebSearchTavilyURL,
			ExaURL:       fallback.WebSearchExaURL,
			TavilyAPIKey: fallback.WebSearchTavilyAPIKey,
			ExaAPIKey:    fallback.WebSearchExaAPIKey,
		}),
		ResponseOptions:                responseOptions,
		Providers:                      normalizeProviderConfigs(fileCfg.Providers, stringValue(fileCfg.Model)),
		CodexStatelessRetryEnabled:     resolveRuntimeCodexRetryEnabled(fileCfg, fallback),
		MaxTurns:                       maxTurns,
		TaskExecutionTimeoutMS:         taskExecutionTimeoutMS,
		RelayDefaultStopPolicy:         relayDefaults.stopPolicy,
		RelayDefaultMaxRounds:          relayDefaults.maxRounds,
		RelayDefaultExecutionTimeoutMS: relayDefaults.executionTimeoutMS,
		LLMCompletionRetryCount:        retryCount,
		LLMCompletionRetryIntervalMS:   retryIntervalMS,
		ModelSelectionEnabled:          resolveRuntimeModelSelectionEnabled(fileCfg, fallback),
		SessionHumanLogFullEnabled:     resolveRuntimeSessionHumanLogFullEnabled(fileCfg, fallback),
		SessionSystemPromptVisible:     resolveRuntimeSessionSystemPromptVisible(fileCfg, fallback),
		AssistantMarkdownEnabled:       resolveRuntimeAssistantMarkdownEnabled(fileCfg, fallback),
		ToolCallCompactOutputEnabled:   resolveRuntimeToolCallCompactOutputEnabled(fileCfg, fallback),
		MemoryModeEnabled:              resolveRuntimeMemoryModeEnabled(fileCfg, fallback),
		MicrocompactEnabled:            resolveRuntimeMicrocompactEnabled(fileCfg, fallback),
		SessionTitleMode:               sessionTitleMode,
	}, nil
}
