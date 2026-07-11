package runtime

import (
	"fmt"
	"strings"

	"ghost-os/bridge/config/internal/providers"
	"ghost-os/bridge/config/internal/storage"
	"ghost-os/bridge/llm"
)

func Resolve(fileCfg storage.FileConfig, env storage.EnvSnapshot) (Snapshot, error) {
	fallback, err := FallbackFromEnv(env)
	if err != nil {
		return Snapshot{}, err
	}
	return ResolveWithFallback(fileCfg, fallback)
}

func FallbackFromEnv(env storage.EnvSnapshot) (Snapshot, error) {
	settings, err := resolveFallbackSettings(env)
	if err != nil {
		return Snapshot{}, err
	}
	return Normalize(Snapshot{
		ProviderName:                   env.DefaultValue("GHOST_PROVIDER", string(DefaultProvider)),
		APIKey:                         env.DefaultValue("GHOST_API_KEY", ""),
		BaseURL:                        env.DefaultValue("GHOST_BASE_URL", ""),
		Model:                          env.DefaultValue("GHOST_MODEL", ""),
		ChatPath:                       env.DefaultValue("GHOST_CHAT_PATH", ""),
		ResponseOptions:                settings.ResponseOptions,
		CodexStatelessRetryEnabled:     settings.CodexStatelessRetryEnabled,
		NativePersistent:               settings.NativePersistent,
		ProjectRoot:                    env.DefaultValue("GHOST_PROJECT_ROOT", ""),
		MaxTurns:                       settings.MaxTurns,
		TaskExecutionTimeoutMS:         settings.TaskExecutionTimeoutMS,
		RelayDefaultStopPolicy:         settings.RelayDefaultStopPolicy,
		RelayDefaultMaxRounds:          settings.RelayDefaultMaxRounds,
		RelayDefaultExecutionTimeoutMS: settings.RelayDefaultExecutionTimeoutMS,
		ExternalCodexPermissionMode:    settings.ExternalCodexPermissionMode,
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

func ResolveWithFallback(fileCfg storage.FileConfig, fallback Snapshot) (Snapshot, error) {
	normalizedFileCfg, err := storage.NormalizeForWrite(fileCfg)
	if err != nil {
		return Snapshot{}, err
	}
	fallback = Normalize(fallback)
	settings, err := resolveFileSettings(normalizedFileCfg, fallback)
	if err != nil {
		return Snapshot{}, err
	}
	input := buildInput{
		FileCfg:  normalizedFileCfg,
		Fallback: fallback,
		Settings: settings,
	}
	if len(settings.Providers) > 0 {
		return buildWithProviders(input, settings.Providers), nil
	}
	return buildWithoutProviders(input), nil
}

type fallbackSettings struct {
	WebSearch                      webSearchSettings
	ResponseOptions                llm.ResponseOptions
	CodexStatelessRetryEnabled     bool
	NativePersistent               bool
	MaxTurns                       int
	TaskExecutionTimeoutMS         int
	RelayDefaultStopPolicy         string
	RelayDefaultMaxRounds          int
	RelayDefaultExecutionTimeoutMS int
	ExternalCodexPermissionMode    string
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

type fileSettings struct {
	WebSearch                      webSearchSettings
	ResponseOptions                llm.ResponseOptions
	Providers                      []providers.Record
	CodexStatelessRetryEnabled     bool
	MaxTurns                       int
	TaskExecutionTimeoutMS         int
	RelayDefaultStopPolicy         string
	RelayDefaultMaxRounds          int
	RelayDefaultExecutionTimeoutMS int
	ExternalCodexPermissionMode    string
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

type buildInput struct {
	FileCfg  storage.FileConfig
	Fallback Snapshot
	Settings fileSettings
}

func resolveFallbackSettings(env storage.EnvSnapshot) (fallbackSettings, error) {
	if err := validateNoRemovedGraphQLEnv(env); err != nil {
		return fallbackSettings{}, err
	}
	responseOptions, err := responseOptionsFromEnv(env)
	if err != nil {
		return fallbackSettings{}, err
	}
	codexRetryEnabled, nativePersistent, sessionHumanLogFullEnabled, err := resolveFallbackFlags(env)
	if err != nil {
		return fallbackSettings{}, err
	}
	maxTurns, err := storage.ParsePositiveIntValue(env.Value("GHOST_MAX_TURNS"), "GHOST_MAX_TURNS", DefaultMaxTurns)
	if err != nil {
		return fallbackSettings{}, err
	}
	taskExecutionTimeoutMS, err := storage.ParsePositiveIntValue(
		env.Value("GHOST_TASK_EXECUTION_TIMEOUT_MS"),
		"GHOST_TASK_EXECUTION_TIMEOUT_MS",
		DefaultTaskExecutionTimeoutMS,
	)
	if err != nil {
		return fallbackSettings{}, err
	}
	relayDefaults, err := defaultRelaySettings()
	if err != nil {
		return fallbackSettings{}, err
	}
	retryCount, err := storage.ParseNonNegativeIntValue(
		env.Value("GHOST_LLM_COMPLETION_RETRY_COUNT"),
		"GHOST_LLM_COMPLETION_RETRY_COUNT",
		DefaultLLMCompletionRetryCount,
	)
	if err != nil {
		return fallbackSettings{}, err
	}
	retryIntervalMS, err := storage.ParseNonNegativeIntValue(
		env.Value("GHOST_LLM_COMPLETION_RETRY_INTERVAL_MS"),
		"GHOST_LLM_COMPLETION_RETRY_INTERVAL_MS",
		DefaultLLMCompletionRetryIntervalMS,
	)
	if err != nil {
		return fallbackSettings{}, err
	}
	return fallbackSettings{
		WebSearch:                      webSearchSettingsFromEnv(env),
		ResponseOptions:                responseOptions,
		CodexStatelessRetryEnabled:     codexRetryEnabled,
		NativePersistent:               nativePersistent,
		MaxTurns:                       maxTurns,
		TaskExecutionTimeoutMS:         taskExecutionTimeoutMS,
		RelayDefaultStopPolicy:         relayDefaults.stopPolicy,
		RelayDefaultMaxRounds:          relayDefaults.maxRounds,
		RelayDefaultExecutionTimeoutMS: relayDefaults.executionTimeoutMS,
		ExternalCodexPermissionMode:    DefaultExternalCodexPermissionMode,
		LLMCompletionRetryCount:        retryCount,
		LLMCompletionRetryIntervalMS:   retryIntervalMS,
		ModelSelectionEnabled:          DefaultModelSelectionEnabled,
		SessionHumanLogFullEnabled:     sessionHumanLogFullEnabled,
		SessionSystemPromptVisible:     DefaultSessionSystemPromptVisible,
		AssistantMarkdownEnabled:       DefaultAssistantMarkdownEnabled,
		ToolCallCompactOutputEnabled:   DefaultToolCallCompactOutputEnabled,
		MemoryModeEnabled:              DefaultMemoryModeEnabled,
		MicrocompactEnabled:            DefaultMicrocompactEnabled,
		SessionTitleMode:               DefaultSessionTitleMode,
	}, nil
}

func resolveFallbackFlags(env storage.EnvSnapshot) (bool, bool, bool, error) {
	codexRetryEnabled, err := storage.ParseBoolValue(
		env.Value("GHOST_CODEX_STATELESS_RETRY_ENABLED"),
		"GHOST_CODEX_STATELESS_RETRY_ENABLED",
		false,
	)
	if err != nil {
		return false, false, false, err
	}
	if err := validateRuntimeToolAllowlistOnlyEnv(env); err != nil {
		return false, false, false, err
	}
	nativePersistent, err := storage.ResolveNativePersistent(nil, env)
	if err != nil {
		return false, false, false, err
	}
	sessionHumanLogFullEnabled, err := storage.ParseBoolValue(
		env.Value("GHOST_SESSION_HUMAN_LOG_FULL_ENABLED"),
		"GHOST_SESSION_HUMAN_LOG_FULL_ENABLED",
		false,
	)
	if err != nil {
		return false, false, false, err
	}
	return codexRetryEnabled, nativePersistent, sessionHumanLogFullEnabled, nil
}

func validateRuntimeToolAllowlistOnlyEnv(env storage.EnvSnapshot) error {
	_, err := storage.ParseBoolValue(env.Value("GHOST_TOOL_ALLOWLIST_ONLY"), "GHOST_TOOL_ALLOWLIST_ONLY", false)
	return err
}

func resolveFileSettings(fileCfg storage.FileConfig, fallback Snapshot) (fileSettings, error) {
	responseOptions, err := fileResponseOptions(fileCfg, fallback.ResponseOptions)
	if err != nil {
		return fileSettings{}, err
	}
	maxTurns, err := resolveMaxTurns(fileCfg, fallback)
	if err != nil {
		return fileSettings{}, err
	}
	taskExecutionTimeoutMS, err := resolveTaskExecutionTimeoutMS(fileCfg, fallback)
	if err != nil {
		return fileSettings{}, err
	}
	relayDefaults, err := resolveRelayDefaults(fileCfg, fallback)
	if err != nil {
		return fileSettings{}, err
	}
	retryCount, err := resolveLLMCompletionRetryCount(fileCfg, fallback)
	if err != nil {
		return fileSettings{}, err
	}
	retryIntervalMS, err := resolveLLMCompletionRetryIntervalMS(fileCfg, fallback)
	if err != nil {
		return fileSettings{}, err
	}
	sessionTitleMode, err := resolveSessionTitleMode(fileCfg, fallback)
	if err != nil {
		return fileSettings{}, err
	}
	externalCodexPermissionMode, err := resolveExternalCodexPermissionMode(fileCfg, fallback)
	if err != nil {
		return fileSettings{}, err
	}
	return fileSettings{
		WebSearch: fileWebSearchSettings(fileCfg, webSearchSettings{
			TavilyURL:    fallback.WebSearchTavilyURL,
			ExaURL:       fallback.WebSearchExaURL,
			TavilyAPIKey: fallback.WebSearchTavilyAPIKey,
			ExaAPIKey:    fallback.WebSearchExaAPIKey,
		}),
		ResponseOptions:                responseOptions,
		Providers:                      storage.NormalizeProviderConfigs(fileCfg.Providers, storage.StringValue(fileCfg.Model)),
		CodexStatelessRetryEnabled:     resolveCodexRetryEnabled(fileCfg, fallback),
		MaxTurns:                       maxTurns,
		TaskExecutionTimeoutMS:         taskExecutionTimeoutMS,
		RelayDefaultStopPolicy:         relayDefaults.stopPolicy,
		RelayDefaultMaxRounds:          relayDefaults.maxRounds,
		RelayDefaultExecutionTimeoutMS: relayDefaults.executionTimeoutMS,
		ExternalCodexPermissionMode:    externalCodexPermissionMode,
		LLMCompletionRetryCount:        retryCount,
		LLMCompletionRetryIntervalMS:   retryIntervalMS,
		ModelSelectionEnabled:          resolveModelSelectionEnabled(fileCfg, fallback),
		SessionHumanLogFullEnabled:     resolveSessionHumanLogFullEnabled(fileCfg, fallback),
		SessionSystemPromptVisible:     resolveSessionSystemPromptVisible(fileCfg, fallback),
		AssistantMarkdownEnabled:       resolveAssistantMarkdownEnabled(fileCfg, fallback),
		ToolCallCompactOutputEnabled:   resolveToolCallCompactOutputEnabled(fileCfg, fallback),
		MemoryModeEnabled:              resolveMemoryModeEnabled(fileCfg, fallback),
		MicrocompactEnabled:            resolveMicrocompactEnabled(fileCfg, fallback),
		SessionTitleMode:               sessionTitleMode,
	}, nil
}

func validateNoRemovedGraphQLEnv(env storage.EnvSnapshot) error {
	names := configuredRemovedGraphQLEnv(env)
	if len(names) == 0 {
		return nil
	}
	return fmt.Errorf("unsupported GraphQL env vars: %s", strings.Join(names, ", "))
}

func configuredRemovedGraphQLEnv(env storage.EnvSnapshot) []string {
	names := []string{
		"GHOST_GRAPHQL_TOOL_RUNTIME_ENABLED",
		"GHOST_GRAPHQL_TEXT_SANITIZE_ENABLED",
		"GHOST_GRAPHQL_ENABLED",
		"GHOST_GRAPHQL_ENDPOINT",
		"GHOST_GRAPHQL_API_KEY",
		"GHOST_GRAPHQL_SCHEMA_PATH",
		"GHOST_GRAPHQL_TIMEOUT_MS",
		"GHOST_GRAPHQL_MAX_RESPONSE_BYTES",
		"GHOST_GRAPHQL_HEADERS",
	}
	out := make([]string, 0, len(names))
	for _, name := range names {
		if env.Value(name) != "" {
			out = append(out, name)
		}
	}
	return out
}
