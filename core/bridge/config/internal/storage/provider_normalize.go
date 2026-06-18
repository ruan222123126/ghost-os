package storage

import "ghost-os/bridge/config/internal/providers"

func NormalizeProviderConfigs(raw map[string]ProviderFileConfig, model string) []ProviderConfig {
	return providers.NormalizeFileMap(providerFileMapToDomain(raw), model)
}

func ProviderConfigsToFileMap(records []ProviderConfig) map[string]ProviderFileConfig {
	return providerFileMapFromDomain(providers.ToFileMap(records))
}

func CloneModelTokenOverrides(raw map[string]int) map[string]int {
	return providers.CloneModelTokenOverrides(raw)
}

func NormalizedActiveProviderName(records []ProviderConfig, preferred ...*string) *string {
	return providers.NormalizedActiveProviderName(records, preferred...)
}

func providerFileMapToDomain(raw map[string]ProviderFileConfig) map[string]providers.FileRecord {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]providers.FileRecord, len(raw))
	for name, record := range raw {
		out[name] = providers.FileRecord{
			Type:                       record.Type,
			BaseURL:                    record.BaseURL,
			APIKey:                     CloneOptionalStringPointer(record.APIKey),
			Models:                     append([]string(nil), record.Models...),
			ContextWindowTokens:        record.ContextWindowTokens,
			ResponseReserveTokens:      record.ResponseReserveTokens,
			ModelContextWindowTokens:   CloneModelTokenOverrides(record.ModelContextWindowTokens),
			ModelResponseReserveTokens: CloneModelTokenOverrides(record.ModelResponseReserveTokens),
		}
	}
	return out
}

func providerFileMapFromDomain(raw map[string]providers.FileRecord) map[string]ProviderFileConfig {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]ProviderFileConfig, len(raw))
	for name, record := range raw {
		out[name] = ProviderFileConfig{
			Type:                       record.Type,
			BaseURL:                    record.BaseURL,
			APIKey:                     CloneOptionalStringPointer(record.APIKey),
			Models:                     append([]string(nil), record.Models...),
			ContextWindowTokens:        record.ContextWindowTokens,
			ResponseReserveTokens:      record.ResponseReserveTokens,
			ModelContextWindowTokens:   CloneModelTokenOverrides(record.ModelContextWindowTokens),
			ModelResponseReserveTokens: CloneModelTokenOverrides(record.ModelResponseReserveTokens),
		}
	}
	return out
}
