package runtime

import (
	"fmt"
	"strings"

	"ghost-os/bridge/config/internal/providers"
	"ghost-os/bridge/config/internal/storage"
	"ghost-os/bridge/llm"
)

func resolveProviderName(fallback Snapshot) string {
	providerName := strings.TrimSpace(fallback.ProviderName)
	if providerName == "" {
		return string(DefaultProvider)
	}
	return providerName
}

func resolveModel(fileCfg storage.FileConfig, fallback Snapshot) string {
	model := storage.StringValue(fileCfg.Model)
	if model == "" {
		return fallback.Model
	}
	return model
}

func resolveChatPath(fileCfg storage.FileConfig, fallback Snapshot) string {
	chatPath := storage.StringValue(fileCfg.ChatPath)
	if chatPath == "" {
		return fallback.ChatPath
	}
	return chatPath
}

func resolveProjectRoot(fileCfg storage.FileConfig, fallback Snapshot) string {
	projectRoot := storage.StringValue(fileCfg.ProjectRoot)
	if projectRoot == "" {
		return fallback.ProjectRoot
	}
	return projectRoot
}

func resolveMaxTurns(fileCfg storage.FileConfig, fallback Snapshot) (int, error) {
	if fileCfg.MaxTurns != nil {
		if *fileCfg.MaxTurns <= 0 {
			return 0, fmt.Errorf("invalid max_turns: must be > 0, got %d", *fileCfg.MaxTurns)
		}
		return *fileCfg.MaxTurns, nil
	}
	return fallback.MaxTurns, nil
}

func resolveTaskExecutionTimeoutMS(fileCfg storage.FileConfig, fallback Snapshot) (int, error) {
	if fileCfg.TaskExecutionTimeoutMS != nil {
		if *fileCfg.TaskExecutionTimeoutMS <= 0 {
			return 0, fmt.Errorf(
				"invalid task_execution_timeout_ms: must be > 0, got %d",
				*fileCfg.TaskExecutionTimeoutMS,
			)
		}
		return *fileCfg.TaskExecutionTimeoutMS, nil
	}
	return fallback.TaskExecutionTimeoutMS, nil
}

func resolveNativePersistent(fileCfg storage.FileConfig, fallback Snapshot) bool {
	if fileCfg.NativePersistent != nil {
		return *fileCfg.NativePersistent
	}
	return fallback.NativePersistent
}

func resolveAPIKey(active providers.Record, fallbackAPIKey string) string {
	if active.APIKey != nil {
		return strings.TrimSpace(*active.APIKey)
	}
	return fallbackAPIKey
}

func resolveBaseURL(baseURL, fallbackBaseURL string) string {
	if strings.TrimSpace(baseURL) == "" {
		return fallbackBaseURL
	}
	return strings.TrimSpace(baseURL)
}

func ActiveProviderLabel(runtime Snapshot) string {
	return providers.ActiveLabel(providerRuntimeSnapshot(runtime))
}

func InferProviderType(name, baseURL, model string) llm.Provider {
	return providers.InferType(name, baseURL, model)
}

func DefaultBaseURLForProvider(provider llm.Provider) string {
	return providers.DefaultBaseURL(provider)
}

func Normalize(raw Snapshot) Snapshot {
	out := raw
	out.ProviderName = strings.TrimSpace(out.ProviderName)
	if out.Provider == "" {
		out.Provider = providers.InferType(out.ProviderName, out.BaseURL, out.Model)
	}
	if out.Provider == "" {
		out.Provider = DefaultProvider
	}
	if out.ProviderName == "" {
		out.ProviderName = string(out.Provider)
	}
	out.APIKey = strings.TrimSpace(out.APIKey)
	if strings.TrimSpace(out.BaseURL) == "" {
		out.BaseURL = providers.DefaultBaseURL(out.Provider)
	} else {
		out.BaseURL = strings.TrimSpace(out.BaseURL)
	}
	if strings.TrimSpace(out.Model) == "" {
		out.Model = DefaultModel
	} else {
		out.Model = strings.TrimSpace(out.Model)
	}
	out.ChatPath = strings.TrimSpace(out.ChatPath)
	out.ResponseOptions = llm.CloneResponseOptions(out.ResponseOptions)
	out.ProjectRoot = strings.TrimSpace(out.ProjectRoot)
	if out.MaxTurns <= 0 {
		out.MaxTurns = DefaultMaxTurns
	}
	if out.TaskExecutionTimeoutMS <= 0 {
		out.TaskExecutionTimeoutMS = DefaultTaskExecutionTimeoutMS
	}
	if strings.TrimSpace(out.RelayDefaultStopPolicy) == "" {
		out.RelayDefaultStopPolicy = DefaultRelayStopPolicy
	} else {
		out.RelayDefaultStopPolicy = strings.TrimSpace(out.RelayDefaultStopPolicy)
	}
	if out.RelayDefaultMaxRounds <= 0 {
		out.RelayDefaultMaxRounds = DefaultRelayMaxRounds
	}
	if out.RelayDefaultExecutionTimeoutMS < 0 {
		out.RelayDefaultExecutionTimeoutMS = DefaultRelayExecutionTimeoutMS
	}
	out.WebSearchTavilyURL = strings.TrimSpace(out.WebSearchTavilyURL)
	out.WebSearchExaURL = strings.TrimSpace(out.WebSearchExaURL)
	out.WebSearchTavilyAPIKey = strings.TrimSpace(out.WebSearchTavilyAPIKey)
	out.WebSearchExaAPIKey = strings.TrimSpace(out.WebSearchExaAPIKey)
	sessionTitleMode, err := NormalizeSessionTitleMode(out.SessionTitleMode)
	if err == nil {
		out.SessionTitleMode = sessionTitleMode
	}
	return out
}

func ValidateForExecution(runtime Snapshot) error {
	switch runtime.Provider {
	case llm.ProviderOpenAI, llm.ProviderAnthropic, llm.ProviderCodex:
		if runtime.APIKey == "" {
			return fmt.Errorf("GHOST_API_KEY is required for provider %q", runtime.Provider)
		}
	case llm.ProviderCustom:
	default:
		return fmt.Errorf("invalid GHOST_PROVIDER=%q, expected one of: openai|anthropic|custom|codex", runtime.Provider)
	}
	return nil
}

func providerRuntimeSnapshot(runtime Snapshot) providers.RuntimeSnapshot {
	return providers.RuntimeSnapshot{
		ProviderName: runtime.ProviderName,
		Provider:     runtime.Provider,
		APIKey:       runtime.APIKey,
		BaseURL:      runtime.BaseURL,
		Model:        runtime.Model,
	}
}
