package config

import (
	"sort"
	"strings"

	"ghost-os/bridge/llm"
)

func normalizeProviderConfigs(raw map[string]providerFileConfig, model string) []providerConfig {
	if len(raw) == 0 {
		return nil
	}

	names := make([]string, 0, len(raw))
	for name := range raw {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]providerConfig, 0, len(names))
	for _, name := range names {
		record := raw[name]
		provider, ok := normalizeProviderRecord(name, record.Type, record.BaseURL, record.APIKey, record.Models, record.ContextWindowTokens, record.ResponseReserveTokens, record.ModelContextWindowTokens, record.ModelResponseReserveTokens, model)
		if !ok {
			continue
		}
		out = append(out, provider)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeLegacyProviderConfigs(raw []legacyProviderConfig, model string) []providerConfig {
	if len(raw) == 0 {
		return nil
	}

	out := make([]providerConfig, 0, len(raw))
	for _, provider := range raw {
		normalized, ok := normalizeProviderRecord(provider.Name, provider.Type, provider.BaseURL, provider.APIKey, provider.Models, 0, 0, nil, nil, model)
		if !ok {
			continue
		}
		out = append(out, normalized)
	}
	if len(out) == 0 {
		return nil
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func normalizeProviderRecord(
	name string,
	rawType llm.Provider,
	baseURL string,
	apiKey *string,
	models []string,
	rawContextWindowTokens int,
	rawResponseReserveTokens int,
	rawModelContextWindowTokens map[string]int,
	rawModelResponseReserveTokens map[string]int,
	model string,
) (providerConfig, bool) {
	trimmedName := strings.TrimSpace(name)
	trimmedBaseURL := strings.TrimSpace(baseURL)
	normalizedType := normalizedProviderType(rawType, trimmedName, trimmedBaseURL, model)
	normalizedAPIKey := cloneOptionalStringPointer(apiKey)
	normalizedModels := normalizeProviderModels(models)
	if trimmedName == "" && trimmedBaseURL == "" && normalizedAPIKey == nil && len(normalizedModels) == 0 {
		return providerConfig{}, false
	}
	if trimmedName == "" {
		return providerConfig{}, false
	}
	if trimmedBaseURL == "" {
		trimmedBaseURL = defaultBaseURLForProvider(normalizedType)
	}

	return providerConfig{
		Name:                       trimmedName,
		Type:                       normalizedType,
		BaseURL:                    trimmedBaseURL,
		APIKey:                     normalizedAPIKey,
		Models:                     normalizedModels,
		ContextWindowTokens:        normalizePositiveInt(rawContextWindowTokens),
		ResponseReserveTokens:      normalizePositiveInt(rawResponseReserveTokens),
		ModelContextWindowTokens:   normalizeModelTokenOverrides(rawModelContextWindowTokens),
		ModelResponseReserveTokens: normalizeModelTokenOverrides(rawModelResponseReserveTokens),
	}, true
}

func normalizedProviderType(rawType llm.Provider, name, baseURL, model string) llm.Provider {
	normalizedType := rawType.Normalized()
	if normalizedType == "" {
		normalizedType = inferProviderType(name, baseURL, model)
	}
	if normalizedType == "" {
		return defaultProvider
	}
	return normalizedType
}

func providerConfigsToFileMap(providers []providerConfig) map[string]providerFileConfig {
	if len(providers) == 0 {
		return nil
	}

	out := make(map[string]providerFileConfig, len(providers))
	for _, provider := range providers {
		name := strings.TrimSpace(provider.Name)
		if name == "" {
			continue
		}
		out[name] = providerFileConfig{
			Type:                       provider.Type.Normalized(),
			BaseURL:                    strings.TrimSpace(provider.BaseURL),
			APIKey:                     cloneOptionalStringPointer(provider.APIKey),
			Models:                     normalizeProviderModels(provider.Models),
			ContextWindowTokens:        normalizePositiveInt(provider.ContextWindowTokens),
			ResponseReserveTokens:      normalizePositiveInt(provider.ResponseReserveTokens),
			ModelContextWindowTokens:   normalizeModelTokenOverrides(provider.ModelContextWindowTokens),
			ModelResponseReserveTokens: normalizeModelTokenOverrides(provider.ModelResponseReserveTokens),
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeProviderModels(models []string) []string {
	if len(models) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(models))
	out := make([]string, 0, len(models))
	for _, model := range models {
		trimmed := strings.TrimSpace(model)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func providerIndexByName(providers []providerConfig, name string) int {
	target := strings.TrimSpace(name)
	if target == "" {
		return -1
	}
	for index, provider := range providers {
		if strings.EqualFold(strings.TrimSpace(provider.Name), target) {
			return index
		}
	}
	return -1
}

func cloneProviderConfigs(providers []providerConfig) []providerConfig {
	if len(providers) == 0 {
		return nil
	}

	out := make([]providerConfig, 0, len(providers))
	for _, provider := range providers {
		out = append(out, providerConfig{
			Name:                       provider.Name,
			Type:                       provider.Type,
			BaseURL:                    provider.BaseURL,
			APIKey:                     cloneOptionalStringPointer(provider.APIKey),
			Models:                     append([]string(nil), provider.Models...),
			ContextWindowTokens:        provider.ContextWindowTokens,
			ResponseReserveTokens:      provider.ResponseReserveTokens,
			ModelContextWindowTokens:   cloneModelTokenOverrides(provider.ModelContextWindowTokens),
			ModelResponseReserveTokens: cloneModelTokenOverrides(provider.ModelResponseReserveTokens),
		})
	}
	return out
}

func normalizePositiveInt(value int) int {
	if value > 0 {
		return value
	}
	return 0
}

func normalizeModelTokenOverrides(raw map[string]int) map[string]int {
	if len(raw) == 0 {
		return nil
	}

	out := make(map[string]int, len(raw))
	for key, value := range raw {
		trimmed := strings.ToLower(strings.TrimSpace(key))
		if trimmed == "" || value <= 0 {
			continue
		}
		out[trimmed] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cloneModelTokenOverrides(raw map[string]int) map[string]int {
	if len(raw) == 0 {
		return nil
	}

	out := make(map[string]int, len(raw))
	for key, value := range raw {
		out[key] = value
	}
	return out
}
