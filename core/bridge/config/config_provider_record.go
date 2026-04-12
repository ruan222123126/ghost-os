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
		provider, ok := normalizeProviderRecord(normalizeProviderRecordInput{
			Name:                          name,
			RawType:                       record.Type,
			BaseURL:                       record.BaseURL,
			APIKey:                        record.APIKey,
			Models:                        record.Models,
			RawContextWindowTokens:        record.ContextWindowTokens,
			RawResponseReserveTokens:      record.ResponseReserveTokens,
			RawModelContextWindowTokens:   record.ModelContextWindowTokens,
			RawModelResponseReserveTokens: record.ModelResponseReserveTokens,
			Model:                         model,
		})
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

type normalizeProviderRecordInput struct {
	Name                          string
	RawType                       llm.Provider
	BaseURL                       string
	APIKey                        *string
	Models                        []string
	RawContextWindowTokens        int
	RawResponseReserveTokens      int
	RawModelContextWindowTokens   map[string]int
	RawModelResponseReserveTokens map[string]int
	Model                         string
}

func normalizeProviderRecord(input normalizeProviderRecordInput) (providerConfig, bool) {
	trimmedName := strings.TrimSpace(input.Name)
	trimmedBaseURL := strings.TrimSpace(input.BaseURL)
	normalizedType := normalizedProviderType(input.RawType, trimmedName, trimmedBaseURL, input.Model)
	normalizedAPIKey := cloneOptionalStringPointer(input.APIKey)
	normalizedModels := normalizeProviderModels(input.Models)
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
		ContextWindowTokens:        normalizePositiveInt(input.RawContextWindowTokens),
		ResponseReserveTokens:      normalizePositiveInt(input.RawResponseReserveTokens),
		ModelContextWindowTokens:   normalizeModelTokenOverrides(input.RawModelContextWindowTokens),
		ModelResponseReserveTokens: normalizeModelTokenOverrides(input.RawModelResponseReserveTokens),
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
