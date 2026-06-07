package providers

import (
	"sort"
	"strings"

	"ghost-os/bridge/llm"
)

func NormalizeFileMap(raw map[string]FileRecord, model string) []Record {
	if len(raw) == 0 {
		return nil
	}

	names := make([]string, 0, len(raw))
	for name := range raw {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]Record, 0, len(names))
	for _, name := range names {
		record := raw[name]
		provider, ok := normalizeFileRecord(normalizeFileRecordInput{
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
		if ok {
			out = append(out, provider)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func ToFileMap(records []Record) map[string]FileRecord {
	if len(records) == 0 {
		return nil
	}

	out := make(map[string]FileRecord, len(records))
	for _, provider := range records {
		name := strings.TrimSpace(provider.Name)
		if name == "" {
			continue
		}
		out[name] = FileRecord{
			Type:                       provider.Type.Normalized(),
			BaseURL:                    strings.TrimSpace(provider.BaseURL),
			APIKey:                     cloneOptionalStringPointer(provider.APIKey),
			Models:                     NormalizeModels(provider.Models),
			ContextWindowTokens:        NormalizePositiveInt(provider.ContextWindowTokens),
			ResponseReserveTokens:      NormalizePositiveInt(provider.ResponseReserveTokens),
			ModelContextWindowTokens:   NormalizeModelTokenOverrides(provider.ModelContextWindowTokens),
			ModelResponseReserveTokens: NormalizeModelTokenOverrides(provider.ModelResponseReserveTokens),
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func ValidateRecord(cfg Record) (Record, error) {
	normalizedType := cfg.Type.Normalized()
	if normalizedType == "" {
		return Record{}, ErrTypeInvalid
	}

	normalized := Record{
		Name:                       strings.TrimSpace(cfg.Name),
		Type:                       normalizedType,
		BaseURL:                    strings.TrimSpace(cfg.BaseURL),
		APIKey:                     cloneOptionalStringPointer(cfg.APIKey),
		Models:                     NormalizeModels(cfg.Models),
		ContextWindowTokens:        NormalizePositiveInt(cfg.ContextWindowTokens),
		ResponseReserveTokens:      NormalizePositiveInt(cfg.ResponseReserveTokens),
		ModelContextWindowTokens:   NormalizeModelTokenOverrides(cfg.ModelContextWindowTokens),
		ModelResponseReserveTokens: NormalizeModelTokenOverrides(cfg.ModelResponseReserveTokens),
	}
	if normalized.Name == "" {
		return Record{}, ErrNameRequired
	}
	if normalized.BaseURL == "" {
		if normalized.Type == llm.ProviderCustom {
			return Record{}, ErrBaseURLRequired
		}
		normalized.BaseURL = DefaultBaseURL(normalized.Type)
	}
	return normalized, nil
}

func InferType(name, baseURL, model string) llm.Provider {
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

func DefaultBaseURL(provider llm.Provider) string {
	switch provider.Normalized() {
	case llm.ProviderAnthropic:
		return defaultAnthropicBaseURL
	default:
		return defaultBaseURL
	}
}

func IndexByName(records []Record, name string) int {
	target := strings.TrimSpace(name)
	if target == "" {
		return -1
	}
	for index, provider := range records {
		if strings.EqualFold(strings.TrimSpace(provider.Name), target) {
			return index
		}
	}
	return -1
}

func NormalizeModels(models []string) []string {
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

func NormalizePositiveInt(value int) int {
	if value > 0 {
		return value
	}
	return 0
}

func NormalizeModelTokenOverrides(raw map[string]int) map[string]int {
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

func CloneModelTokenOverrides(raw map[string]int) map[string]int {
	if len(raw) == 0 {
		return nil
	}

	out := make(map[string]int, len(raw))
	for key, value := range raw {
		out[key] = value
	}
	return out
}

func NormalizedActiveProviderName(records []Record, preferred ...*string) *string {
	if len(records) == 0 {
		return nil
	}
	for _, candidate := range preferred {
		name := strings.TrimSpace(stringValue(candidate))
		if IndexByName(records, name) >= 0 {
			return stringPointer(name)
		}
	}
	return stringPointer(records[0].Name)
}

type normalizeFileRecordInput struct {
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

func normalizeFileRecord(input normalizeFileRecordInput) (Record, bool) {
	trimmedName := strings.TrimSpace(input.Name)
	trimmedBaseURL := strings.TrimSpace(input.BaseURL)
	normalizedType := normalizedProviderType(input.RawType, trimmedName, trimmedBaseURL, input.Model)
	normalizedAPIKey := cloneOptionalStringPointer(input.APIKey)
	normalizedModels := NormalizeModels(input.Models)
	if trimmedName == "" && trimmedBaseURL == "" && normalizedAPIKey == nil && len(normalizedModels) == 0 {
		return Record{}, false
	}
	if trimmedName == "" {
		return Record{}, false
	}
	if trimmedBaseURL == "" {
		trimmedBaseURL = DefaultBaseURL(normalizedType)
	}

	return Record{
		Name:                       trimmedName,
		Type:                       normalizedType,
		BaseURL:                    trimmedBaseURL,
		APIKey:                     normalizedAPIKey,
		Models:                     normalizedModels,
		ContextWindowTokens:        NormalizePositiveInt(input.RawContextWindowTokens),
		ResponseReserveTokens:      NormalizePositiveInt(input.RawResponseReserveTokens),
		ModelContextWindowTokens:   NormalizeModelTokenOverrides(input.RawModelContextWindowTokens),
		ModelResponseReserveTokens: NormalizeModelTokenOverrides(input.RawModelResponseReserveTokens),
	}, true
}

func normalizedProviderType(rawType llm.Provider, name, baseURL, model string) llm.Provider {
	normalizedType := rawType.Normalized()
	if normalizedType == "" {
		normalizedType = InferType(name, baseURL, model)
	}
	if normalizedType == "" {
		return defaultProvider
	}
	return normalizedType
}

func normalizeProvider(raw string) llm.Provider {
	return llm.Provider(strings.ToLower(strings.TrimSpace(raw)))
}
