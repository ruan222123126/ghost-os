package config

import (
	"errors"
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
	if req.Model != nil && !current.ModelSelectionEnabled {
		return errModelSelectionDisabled
	}
	providers := normalizeProviderConfigs(fileCfg.Providers, stringValue(fileCfg.Model))
	providers = materializeRuntimeProviderIfNeeded(&fileCfg, current, providers, req)
	if err := applyProviderRuntimeUpdate(&fileCfg, current, providers, req); err != nil {
		return err
	}
	materializeRuntimeGraphQLIfNeeded(&fileCfg, current, req)
	materializeRuntimeWebSearchIfNeeded(&fileCfg, current, req)
	applyConfigRuntimeFields(&fileCfg, req)
	return s.persistLocked(configPath, fileCfg)
}

func materializeRuntimeProviderIfNeeded(
	fileCfg *bridgeFileConfig,
	current runtimeConfig,
	providers []providerConfig,
	req configUpdateRequest,
) []providerConfig {
	if len(providers) > 0 || !requiresRuntimeProviderMaterialization(req) {
		return providers
	}
	materializeActiveProvider(fileCfg, current, req.Provider)
	return normalizeProviderConfigs(fileCfg.Providers, stringValue(fileCfg.Model))
}

func requiresRuntimeProviderMaterialization(req configUpdateRequest) bool {
	return req.Provider != nil || req.BaseURL != nil || req.APIKey != nil
}

func materializeRuntimeWebSearchIfNeeded(
	fileCfg *bridgeFileConfig,
	current runtimeConfig,
	req configUpdateRequest,
) {
	if fileCfg == nil || !requiresRuntimeWebSearchMaterialization(req) {
		return
	}
	if fileCfg.WebSearchTavilyAPIKey == nil {
		fileCfg.WebSearchTavilyAPIKey = optionalStringPointer(current.WebSearchTavilyAPIKey)
	}
	if fileCfg.WebSearchExaAPIKey == nil {
		fileCfg.WebSearchExaAPIKey = optionalStringPointer(current.WebSearchExaAPIKey)
	}
}

func requiresRuntimeWebSearchMaterialization(req configUpdateRequest) bool {
	return req.WebSearchTavilyAPIKey != nil ||
		req.WebSearchExaAPIKey != nil
}

func applyProviderRuntimeUpdate(
	fileCfg *bridgeFileConfig,
	current runtimeConfig,
	providers []providerConfig,
	req configUpdateRequest,
) error {
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

func applyActiveProviderConfigUpdate(provider providerConfig, req configUpdateRequest) (providerConfig, bool) {
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

func applyConfigRuntimeFields(fileCfg *bridgeFileConfig, req configUpdateRequest) {
	if req.Model != nil {
		fileCfg.Model = cloneOptionalStringPointer(req.Model)
	}
	if req.ChatPath != nil {
		fileCfg.ChatPath = cloneOptionalStringPointer(req.ChatPath)
	}
	applyGraphQLRuntimeFields(fileCfg, req)
	if req.WebSearchTavilyAPIKey != nil {
		fileCfg.WebSearchTavilyAPIKey = cloneOptionalStringPointer(req.WebSearchTavilyAPIKey)
	}
	if req.WebSearchExaAPIKey != nil {
		fileCfg.WebSearchExaAPIKey = cloneOptionalStringPointer(req.WebSearchExaAPIKey)
	}
}

// SetProjectRoot 更新并持久化 project_root，并刷新运行态快照。
func (s *ConfigStore) SetProjectRoot(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return errors.New("project_root is required")
	}

	fileCfg, configPath, _, err := s.loadMutationStateLocked()
	if err != nil {
		return err
	}
	fileCfg.ProjectRoot = stringPointer(trimmed)
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
	runtime, err := runtimeConfigFromFileConfigWithFallback(normalized, s.runtime)
	if err != nil {
		return err
	}
	if err := writeBridgeFileConfig(configPath, normalized); err != nil {
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
