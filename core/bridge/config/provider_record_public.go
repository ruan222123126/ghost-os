package config

func providerRecordsFromConfigs(raw []providerConfig) []ProviderRecord {
	if len(raw) == 0 {
		return nil
	}

	out := make([]ProviderRecord, 0, len(raw))
	for _, provider := range raw {
		out = append(out, ProviderRecord{
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

func providerConfigFromRecord(raw ProviderRecord) providerConfig {
	return providerConfig{
		Name:                       raw.Name,
		Type:                       raw.Type,
		BaseURL:                    raw.BaseURL,
		APIKey:                     cloneOptionalStringPointer(raw.APIKey),
		Models:                     append([]string(nil), raw.Models...),
		ContextWindowTokens:        raw.ContextWindowTokens,
		ResponseReserveTokens:      raw.ResponseReserveTokens,
		ModelContextWindowTokens:   cloneModelTokenOverrides(raw.ModelContextWindowTokens),
		ModelResponseReserveTokens: cloneModelTokenOverrides(raw.ModelResponseReserveTokens),
	}
}
