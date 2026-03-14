package config

import (
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
)

func runtimeConfigFromEnv() (runtimeConfig, error) {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return runtimeConfig{}, err
	}
	return runtimeConfigFromFileConfig(fileCfg)
}

func runtimeConfigFromFileConfig(fileCfg bridgeFileConfig) (runtimeConfig, error) {
	webSearch := envWebSearchSettings()
	return runtimeConfigFromFileConfigWithFallback(fileCfg, runtimeConfig{
		ProviderName:          getenvDefault("GHOST_PROVIDER", string(defaultProvider)),
		APIKey:                getenvDefault("GHOST_API_KEY", ""),
		BaseURL:               getenvDefault("GHOST_BASE_URL", ""),
		Model:                 getenvDefault("GHOST_MODEL", ""),
		ChatPath:              getenvDefault("GHOST_CHAT_PATH", ""),
		NativePersistent:      resolveNativePersistent(nil),
		ProjectRoot:           getenvDefault("GHOST_PROJECT_ROOT", ""),
		WebSearchTavilyAPIKey: webSearch.TavilyAPIKey,
		WebSearchExaAPIKey:    webSearch.ExaAPIKey,
	})
}

// runtimeConfigFromFileConfigWithFallback folds file overrides onto an existing runtime snapshot.
func runtimeConfigFromFileConfigWithFallback(fileCfg bridgeFileConfig, fallback runtimeConfig) (runtimeConfig, error) {
	fileCfg = normalizeBridgeFileConfigForWrite(fileCfg)
	fallback = normalizeRuntimeConfig(fallback)
	webSearch := fileWebSearchSettings(fileCfg, webSearchSettings{
		TavilyAPIKey: fallback.WebSearchTavilyAPIKey,
		ExaAPIKey:    fallback.WebSearchExaAPIKey,
	})
	allowlistOnly := boolOrEnv(fileCfg.ToolAllowlistOnly, "GHOST_TOOL_ALLOWLIST_ONLY", false)
	providers := normalizeProviderConfigs(fileCfg.Providers, stringValue(fileCfg.Model))
	if len(providers) > 0 {
		return runtimeConfigWithProviders(fileCfg, fallback, providers, allowlistOnly, webSearch), nil
	}
	return runtimeConfigWithoutProviders(fileCfg, fallback, allowlistOnly, webSearch), nil
}

func runtimeConfigWithProviders(
	fileCfg bridgeFileConfig,
	fallback runtimeConfig,
	providers []providerConfig,
	allowlistOnly bool,
	webSearch webSearchSettings,
) runtimeConfig {
	active := resolveActiveProvider(providers, stringValue(fileCfg.ActiveProvider), fallback)
	return normalizeRuntimeConfig(runtimeConfig{
		ProviderName:               active.Name,
		Provider:                   active.Type.Normalized(),
		APIKey:                     resolveRuntimeAPIKey(active, fallback.APIKey),
		BaseURL:                    resolveRuntimeBaseURL(active.BaseURL, fallback.BaseURL),
		Model:                      resolveRuntimeModel(fileCfg, fallback),
		ChatPath:                   resolveRuntimeChatPath(fileCfg, fallback),
		NativePersistent:           resolveRuntimeNativePersistent(fileCfg, fallback),
		ProjectRoot:                resolveRuntimeProjectRoot(fileCfg, fallback),
		ModelSelectionEnabled:      !allowlistOnly,
		ContextWindowTokens:        active.ContextWindowTokens,
		ResponseReserveTokens:      active.ResponseReserveTokens,
		ModelContextWindowTokens:   cloneModelTokenOverrides(active.ModelContextWindowTokens),
		ModelResponseReserveTokens: cloneModelTokenOverrides(active.ModelResponseReserveTokens),
		WebSearchTavilyAPIKey:      webSearch.TavilyAPIKey,
		WebSearchExaAPIKey:         webSearch.ExaAPIKey,
	})
}

func runtimeConfigWithoutProviders(
	fileCfg bridgeFileConfig,
	fallback runtimeConfig,
	allowlistOnly bool,
	webSearch webSearchSettings,
) runtimeConfig {
	providerName := resolveRuntimeProviderName(fallback)
	return normalizeRuntimeConfig(runtimeConfig{
		ProviderName:               providerName,
		Provider:                   inferProviderType(providerName, fallback.BaseURL, resolveRuntimeModel(fileCfg, fallback)),
		APIKey:                     fallback.APIKey,
		BaseURL:                    fallback.BaseURL,
		Model:                      resolveRuntimeModel(fileCfg, fallback),
		ChatPath:                   resolveRuntimeChatPath(fileCfg, fallback),
		NativePersistent:           resolveRuntimeNativePersistent(fileCfg, fallback),
		ProjectRoot:                resolveRuntimeProjectRoot(fileCfg, fallback),
		ModelSelectionEnabled:      !allowlistOnly,
		ContextWindowTokens:        fallback.ContextWindowTokens,
		ResponseReserveTokens:      fallback.ResponseReserveTokens,
		ModelContextWindowTokens:   cloneModelTokenOverrides(fallback.ModelContextWindowTokens),
		ModelResponseReserveTokens: cloneModelTokenOverrides(fallback.ModelResponseReserveTokens),
		WebSearchTavilyAPIKey:      webSearch.TavilyAPIKey,
		WebSearchExaAPIKey:         webSearch.ExaAPIKey,
	})
}

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
	out.ProjectRoot = strings.TrimSpace(out.ProjectRoot)
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

// getenvDefault 在环境变量为空时回落默认值。
