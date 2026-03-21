package config

import (
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
)

func applyProviderUpdatePatch(
	fileCfg *bridgeFileConfig,
	current runtimeConfig,
	req UpdateRequest,
) error {
	providers := normalizeProviderConfigs(fileCfg.Providers, stringValue(fileCfg.Model))
	if err := applyActiveProviderUpdate(fileCfg, providers, req.Provider); err != nil {
		return err
	}

	index := activeProviderIndex(providers, stringValue(fileCfg.ActiveProvider), current)
	if index < 0 {
		return nil
	}
	updated, changed := applyActiveProviderConfigUpdate(providers[index], req)
	if !changed {
		return nil
	}
	providers[index] = updated
	fileCfg.Providers = providerConfigsToFileMap(providers)
	return nil
}

func applyActiveProviderUpdate(
	fileCfg *bridgeFileConfig,
	providers []providerConfig,
	providerName *string,
) error {
	if providerName == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*providerName)
	if trimmed == "" {
		return errProviderNameRequired
	}
	index := providerIndexByName(providers, trimmed)
	if index < 0 {
		return fmt.Errorf("%w: %s", errProviderNotFound, trimmed)
	}
	fileCfg.ActiveProvider = stringPointer(providers[index].Name)
	return nil
}

func applyActiveProviderConfigUpdate(provider providerConfig, req UpdateRequest) (providerConfig, bool) {
	updated := provider
	changed := false
	if req.APIKey != nil {
		updated.APIKey = cloneOptionalStringPointer(req.APIKey)
		changed = true
	}
	if req.BaseURL != nil {
		updated.BaseURL = resolveUpdatedProviderBaseURL(provider.Type, *req.BaseURL)
		changed = true
	}
	return updated, changed
}

func resolveUpdatedProviderBaseURL(providerType llm.Provider, baseURL string) string {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return defaultBaseURLForProvider(providerType)
	}
	return trimmed
}

func activeProviderIndex(providers []providerConfig, activeName string, current runtimeConfig) int {
	if index := providerIndexByName(providers, activeName); index >= 0 {
		return index
	}
	if index := providerIndexByName(providers, activeProviderLabel(current)); index >= 0 {
		return index
	}
	if len(providers) == 0 {
		return -1
	}
	return 0
}

func prepareActiveProviderUpdateBase(
	fileCfg *bridgeFileConfig,
	current runtimeConfig,
	requestedName *string,
) {
	if fileCfg == nil || len(normalizeProviderConfigs(fileCfg.Providers, stringValue(fileCfg.Model))) > 0 {
		return
	}

	providerName := activeProviderLabel(current)
	if requestedName != nil && strings.TrimSpace(*requestedName) != "" {
		providerName = strings.TrimSpace(*requestedName)
	}
	providerType := inferProviderType(providerName, current.BaseURL, current.Model)
	if providerType == "" {
		providerType = defaultProvider
	}

	baseURL := strings.TrimSpace(current.BaseURL)
	if requestedName != nil &&
		strings.TrimSpace(*requestedName) != "" &&
		!strings.EqualFold(providerName, activeProviderLabel(current)) {
		baseURL = ""
	}
	if baseURL == "" {
		baseURL = defaultBaseURLForProvider(providerType)
	}

	fileCfg.Providers = providerConfigsToFileMap([]providerConfig{{
		Name:    providerName,
		Type:    providerType,
		BaseURL: baseURL,
		APIKey:  optionalStringPointer(current.APIKey),
	}})
	fileCfg.ActiveProvider = stringPointer(providerName)
}

func validateProviderConfig(cfg providerConfig) (providerConfig, error) {
	normalizedType := cfg.Type.Normalized()
	if normalizedType == "" {
		return providerConfig{}, errProviderTypeInvalid
	}

	normalized := providerConfig{
		Name:                       strings.TrimSpace(cfg.Name),
		Type:                       normalizedType,
		BaseURL:                    strings.TrimSpace(cfg.BaseURL),
		APIKey:                     cloneOptionalStringPointer(cfg.APIKey),
		Models:                     normalizeProviderModels(cfg.Models),
		ContextWindowTokens:        normalizePositiveInt(cfg.ContextWindowTokens),
		ResponseReserveTokens:      normalizePositiveInt(cfg.ResponseReserveTokens),
		ModelContextWindowTokens:   normalizeModelTokenOverrides(cfg.ModelContextWindowTokens),
		ModelResponseReserveTokens: normalizeModelTokenOverrides(cfg.ModelResponseReserveTokens),
	}
	if normalized.Name == "" {
		return providerConfig{}, errProviderNameRequired
	}
	if normalized.BaseURL == "" {
		if normalized.Type == llm.ProviderCustom {
			return providerConfig{}, errProviderBaseURLRequired
		}
		normalized.BaseURL = defaultBaseURLForProvider(normalized.Type)
	}
	return normalized, nil
}
