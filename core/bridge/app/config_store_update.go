package app

import (
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
)

// Update 仅覆盖请求中显式给出的字段，并在落库前执行归一化。
func (s *ConfigStore) Update(req configUpdateRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fileCfg, configPath, current, err := s.loadMutationStateLocked()
	if err != nil {
		return err
	}

	providers := normalizeProviderConfigs(fileCfg.Providers, stringValue(fileCfg.Model))
	if len(providers) == 0 && (req.Provider != nil || req.BaseURL != nil || req.APIKey != nil) {
		materializeActiveProvider(&fileCfg, current, req.Provider)
		providers = normalizeProviderConfigs(fileCfg.Providers, stringValue(fileCfg.Model))
	}

	if req.Provider != nil {
		providerName := strings.TrimSpace(*req.Provider)
		if providerName == "" {
			return errProviderNameRequired
		}
		index := providerIndexByName(providers, providerName)
		if index < 0 {
			return fmt.Errorf("%w: %s", errProviderNotFound, providerName)
		}
		fileCfg.ActiveProvider = stringPointer(providers[index].Name)
	}

	if req.APIKey != nil {
		if index := activeProviderIndex(providers, stringValue(fileCfg.ActiveProvider), current); index >= 0 {
			providers[index].APIKey = cloneOptionalStringPointer(req.APIKey)
			fileCfg.Providers = providerConfigsToFileMap(providers)
		}
	}

	if req.BaseURL != nil {
		if index := activeProviderIndex(providers, stringValue(fileCfg.ActiveProvider), current); index >= 0 {
			baseURL := strings.TrimSpace(*req.BaseURL)
			if baseURL == "" {
				baseURL = defaultBaseURLForProvider(providers[index].Type)
			}
			providers[index].BaseURL = baseURL
			fileCfg.Providers = providerConfigsToFileMap(providers)
		}
	}

	if req.Model != nil {
		fileCfg.Model = cloneOptionalStringPointer(req.Model)
	}
	if req.ChatPath != nil {
		fileCfg.ChatPath = cloneOptionalStringPointer(req.ChatPath)
	}

	return s.persistLocked(configPath, fileCfg)
}

func (s *ConfigStore) loadMutationStateLocked() (bridgeFileConfig, string, runtimeConfig, error) {
	fileCfg, configPath, err := loadBridgeFileConfig()
	if err != nil {
		return bridgeFileConfig{}, "", runtimeConfig{}, err
	}

	runtime := s.runtime
	if latest, latestErr := runtimeConfigFromFileConfigWithFallback(fileCfg, runtime); latestErr == nil {
		runtime = latest
	}
	return fileCfg, configPath, runtime, nil
}

func (s *ConfigStore) persistLocked(configPath string, fileCfg bridgeFileConfig) error {
	normalized := normalizeBridgeFileConfigForWrite(fileCfg)
	if err := writeBridgeFileConfig(configPath, normalized); err != nil {
		return err
	}
	runtime, err := runtimeConfigFromFileConfigWithFallback(normalized, s.runtime)
	if err != nil {
		return err
	}
	s.runtime = runtime
	return nil
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

func materializeActiveProvider(fileCfg *bridgeFileConfig, current runtimeConfig, requestedName *string) {
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
	if requestedName != nil && strings.TrimSpace(*requestedName) != "" && !strings.EqualFold(providerName, activeProviderLabel(current)) {
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
