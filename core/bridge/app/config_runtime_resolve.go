package app

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
	return runtimeConfigFromFileConfigWithFallback(fileCfg, runtimeConfig{
		ProviderName:     getenvDefault("GHOST_PROVIDER", string(defaultProvider)),
		APIKey:           getenvDefault("GHOST_API_KEY", ""),
		BaseURL:          getenvDefault("GHOST_BASE_URL", ""),
		Model:            getenvDefault("GHOST_MODEL", ""),
		ChatPath:         getenvDefault("GHOST_CHAT_PATH", ""),
		NativePersistent: resolveNativePersistent(nil),
	})
}

// runtimeConfigFromFileConfigWithFallback folds file overrides onto an existing runtime snapshot.
func runtimeConfigFromFileConfigWithFallback(fileCfg bridgeFileConfig, fallback runtimeConfig) (runtimeConfig, error) {
	fileCfg = normalizeBridgeFileConfigForWrite(fileCfg)
	fallback = normalizeRuntimeConfig(fallback)
	providers := normalizeProviderConfigs(fileCfg.Providers, stringValue(fileCfg.Model))
	if len(providers) > 0 {
		activeName := strings.TrimSpace(stringValue(fileCfg.ActiveProvider))
		if activeName == "" {
			activeName = activeProviderLabel(fallback)
		}
		activeIndex := providerIndexByName(providers, activeName)
		if activeIndex < 0 {
			activeIndex = 0
		}
		active := providers[activeIndex]
		model := stringValue(fileCfg.Model)
		if model == "" {
			model = fallback.Model
		}
		apiKey := fallback.APIKey
		if active.APIKey != nil {
			apiKey = strings.TrimSpace(*active.APIKey)
		}
		baseURL := strings.TrimSpace(active.BaseURL)
		if baseURL == "" {
			baseURL = fallback.BaseURL
		}
		nativePersistent := fallback.NativePersistent
		if fileCfg.NativePersistent != nil {
			nativePersistent = *fileCfg.NativePersistent
		}
		chatPath := stringValue(fileCfg.ChatPath)
		if chatPath == "" {
			chatPath = fallback.ChatPath
		}
		return normalizeRuntimeConfig(runtimeConfig{
			ProviderName:     active.Name,
			Provider:         active.Type.Normalized(),
			APIKey:           apiKey,
			BaseURL:          baseURL,
			Model:            model,
			ChatPath:         chatPath,
			NativePersistent: nativePersistent,
		}), nil
	}

	providerName := strings.TrimSpace(fallback.ProviderName)
	if providerName == "" {
		providerName = string(defaultProvider)
	}
	model := stringValue(fileCfg.Model)
	if model == "" {
		model = fallback.Model
	}
	chatPath := stringValue(fileCfg.ChatPath)
	if chatPath == "" {
		chatPath = fallback.ChatPath
	}
	nativePersistent := fallback.NativePersistent
	if fileCfg.NativePersistent != nil {
		nativePersistent = *fileCfg.NativePersistent
	}
	providerType := inferProviderType(providerName, fallback.BaseURL, model)
	return normalizeRuntimeConfig(runtimeConfig{
		ProviderName:     providerName,
		Provider:         providerType,
		APIKey:           fallback.APIKey,
		BaseURL:          fallback.BaseURL,
		Model:            model,
		ChatPath:         chatPath,
		NativePersistent: nativePersistent,
	}), nil
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
