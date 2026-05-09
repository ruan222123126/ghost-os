package config

import (
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
)

func resolveActiveProvider(
	providers []providerConfig,
	activeName string,
	fallback runtimeConfig,
) providerConfig {
	name := strings.TrimSpace(activeName)
	if name == "" {
		name = activeProviderLabel(fallback)
	}
	index := providerIndexByName(providers, name)
	if index < 0 {
		return providers[0]
	}
	return providers[index]
}

func resolveRuntimeProviderName(fallback runtimeConfig) string {
	providerName := strings.TrimSpace(fallback.ProviderName)
	if providerName == "" {
		return string(defaultProvider)
	}
	return providerName
}

func resolveRuntimeModel(fileCfg bridgeFileConfig, fallback runtimeConfig) string {
	model := stringValue(fileCfg.Model)
	if model == "" {
		return fallback.Model
	}
	return model
}

func resolveRuntimeChatPath(fileCfg bridgeFileConfig, fallback runtimeConfig) string {
	chatPath := stringValue(fileCfg.ChatPath)
	if chatPath == "" {
		return fallback.ChatPath
	}
	return chatPath
}

func resolveRuntimeProjectRoot(fileCfg bridgeFileConfig, fallback runtimeConfig) string {
	projectRoot := stringValue(fileCfg.ProjectRoot)
	if projectRoot == "" {
		return fallback.ProjectRoot
	}
	return projectRoot
}

func resolveRuntimeMaxTurns(fileCfg bridgeFileConfig, fallback runtimeConfig) (int, error) {
	if fileCfg.MaxTurns != nil {
		if *fileCfg.MaxTurns <= 0 {
			return 0, fmt.Errorf("invalid max_turns: must be > 0, got %d", *fileCfg.MaxTurns)
		}
		return *fileCfg.MaxTurns, nil
	}
	return fallback.MaxTurns, nil
}

func resolveRuntimeTaskExecutionTimeoutMS(fileCfg bridgeFileConfig, fallback runtimeConfig) (int, error) {
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

func resolveRuntimeNativePersistent(fileCfg bridgeFileConfig, fallback runtimeConfig) bool {
	if fileCfg.NativePersistent != nil {
		return *fileCfg.NativePersistent
	}
	return fallback.NativePersistent
}

func resolveRuntimeAPIKey(active providerConfig, fallbackAPIKey string) string {
	if active.APIKey != nil {
		return strings.TrimSpace(*active.APIKey)
	}
	return fallbackAPIKey
}

func resolveRuntimeBaseURL(baseURL, fallbackBaseURL string) string {
	if strings.TrimSpace(baseURL) == "" {
		return fallbackBaseURL
	}
	return strings.TrimSpace(baseURL)
}

// normalizeProvider 统一 provider 大小写与空白字符。
func normalizeProvider(raw string) llm.Provider {
	return llm.Provider(strings.ToLower(strings.TrimSpace(raw)))
}

func inferProviderType(name, baseURL, model string) llm.Provider {
	if normalized := normalizeProvider(name).Normalized(); normalized != "" {
		return normalized
	}

	lowerBaseURL := strings.ToLower(strings.TrimSpace(baseURL))
	lowerModel := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.Contains(lowerModel, "claude"), strings.Contains(lowerBaseURL, "anthropic.com"):
		return llm.ProviderAnthropic
	case strings.Contains(lowerModel, "codex"):
		return llm.ProviderCodex
	case strings.Contains(lowerBaseURL, "openai.com"):
		return llm.ProviderOpenAI
	default:
		return llm.ProviderCustom
	}
}

func defaultBaseURLForProvider(provider llm.Provider) string {
	switch provider.Normalized() {
	case llm.ProviderAnthropic:
		return defaultAnthropicBaseURL
	default:
		return defaultBaseURL
	}
}

func activeProviderLabel(runtime runtimeConfig) string {
	if value := strings.TrimSpace(runtime.ProviderName); value != "" {
		return value
	}
	return string(runtime.Provider)
}

// normalizeRuntimeConfig 回填默认值并清理字符串字段。
func normalizeRuntimeConfig(runtime runtimeConfig) runtimeConfig {
	out := runtime
	out.ProviderName = strings.TrimSpace(out.ProviderName)
	if out.Provider == "" {
		out.Provider = inferProviderType(out.ProviderName, out.BaseURL, out.Model)
	}
	if out.Provider == "" {
		out.Provider = defaultProvider
	}
	if out.ProviderName == "" {
		out.ProviderName = string(out.Provider)
	}
	out.APIKey = strings.TrimSpace(out.APIKey)
	if strings.TrimSpace(out.BaseURL) == "" {
		out.BaseURL = defaultBaseURLForProvider(out.Provider)
	} else {
		out.BaseURL = strings.TrimSpace(out.BaseURL)
	}
	if strings.TrimSpace(out.Model) == "" {
		out.Model = defaultModel
	} else {
		out.Model = strings.TrimSpace(out.Model)
	}
	out.ChatPath = strings.TrimSpace(out.ChatPath)
	out.ResponseOptions = llm.CloneResponseOptions(out.ResponseOptions)
	out.ProjectRoot = strings.TrimSpace(out.ProjectRoot)
	if out.MaxTurns <= 0 {
		out.MaxTurns = defaultMaxTurns
	}
	if out.TaskExecutionTimeoutMS <= 0 {
		out.TaskExecutionTimeoutMS = defaultTaskExecutionTimeoutMS
	}
	out.WebSearchTavilyURL = strings.TrimSpace(out.WebSearchTavilyURL)
	out.WebSearchExaURL = strings.TrimSpace(out.WebSearchExaURL)
	out.WebSearchTavilyAPIKey = strings.TrimSpace(out.WebSearchTavilyAPIKey)
	out.WebSearchExaAPIKey = strings.TrimSpace(out.WebSearchExaAPIKey)
	return out
}

// validateRuntimeForExecution 校验当前 provider 的最小执行前置条件。
func validateRuntimeForExecution(runtime runtimeConfig) error {
	switch runtime.Provider {
	case llm.ProviderOpenAI, llm.ProviderAnthropic, llm.ProviderCodex:
		if runtime.APIKey == "" {
			return fmt.Errorf("GHOST_API_KEY is required for provider %q", runtime.Provider)
		}
	case llm.ProviderCustom:
		// custom provider 默认允许不传 API Key。
	default:
		return fmt.Errorf("invalid GHOST_PROVIDER=%q, expected one of: openai|anthropic|custom|codex", runtime.Provider)
	}
	return nil
}
