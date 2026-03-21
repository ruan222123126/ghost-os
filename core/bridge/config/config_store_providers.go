package config

import (
	"fmt"
	"strings"
)

func (s *store) AddProvider(cfg ProviderRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	normalized, err := validateProviderConfig(providerConfigFromRecord(cfg))
	if err != nil {
		return err
	}

	fileCfg, configPath, err := s.loadStoredFileConfigLocked()
	if err != nil {
		return err
	}
	providers := normalizeProviderConfigs(fileCfg.Providers, stringValue(fileCfg.Model))
	if providerIndexByName(providers, normalized.Name) >= 0 {
		return fmt.Errorf("%w: %s", errProviderExists, normalized.Name)
	}

	providers = append(providers, normalized)
	fileCfg.Providers = providerConfigsToFileMap(providers)
	if strings.TrimSpace(stringValue(fileCfg.ActiveProvider)) == "" {
		fileCfg.ActiveProvider = stringPointer(normalized.Name)
	}
	return s.persistLocked(configPath, fileCfg)
}

func (s *store) UpdateProvider(name string, cfg ProviderRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	target := strings.TrimSpace(name)
	if target == "" {
		return errProviderNameRequired
	}

	normalized, err := validateProviderConfig(providerConfigFromRecord(cfg))
	if err != nil {
		return err
	}

	fileCfg, configPath, err := s.loadStoredFileConfigLocked()
	if err != nil {
		return err
	}
	providers := normalizeProviderConfigs(fileCfg.Providers, stringValue(fileCfg.Model))

	index := providerIndexByName(providers, target)
	if index < 0 {
		return fmt.Errorf("%w: %s", errProviderNotFound, target)
	}
	if duplicate := providerIndexByName(providers, normalized.Name); duplicate >= 0 && duplicate != index {
		return fmt.Errorf("%w: %s", errProviderExists, normalized.Name)
	}

	current := providers[index]
	oldName := current.Name
	if normalized.APIKey == nil {
		normalized.APIKey = cloneOptionalStringPointer(current.APIKey)
	}
	providers[index] = normalized
	fileCfg.Providers = providerConfigsToFileMap(providers)
	if strings.EqualFold(strings.TrimSpace(stringValue(fileCfg.ActiveProvider)), oldName) {
		fileCfg.ActiveProvider = stringPointer(normalized.Name)
	}
	return s.persistLocked(configPath, fileCfg)
}

func (s *store) DeleteProvider(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	target := strings.TrimSpace(name)
	if target == "" {
		return errProviderNameRequired
	}

	fileCfg, configPath, err := s.loadStoredFileConfigLocked()
	if err != nil {
		return err
	}
	providers := normalizeProviderConfigs(fileCfg.Providers, stringValue(fileCfg.Model))

	index := providerIndexByName(providers, target)
	if index < 0 {
		return fmt.Errorf("%w: %s", errProviderNotFound, target)
	}

	deletedName := providers[index].Name
	providers = append(providers[:index], providers[index+1:]...)
	fileCfg.Providers = providerConfigsToFileMap(providers)
	if strings.EqualFold(strings.TrimSpace(stringValue(fileCfg.ActiveProvider)), deletedName) {
		fileCfg.ActiveProvider = normalizedActiveProviderName(providers)
	}
	return s.persistLocked(configPath, fileCfg)
}

func (s *store) SetActiveProvider(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	target := strings.TrimSpace(name)
	if target == "" {
		return errProviderNameRequired
	}

	fileCfg, configPath, err := s.loadStoredFileConfigLocked()
	if err != nil {
		return err
	}
	providers := normalizeProviderConfigs(fileCfg.Providers, stringValue(fileCfg.Model))

	index := providerIndexByName(providers, target)
	if index < 0 {
		return fmt.Errorf("%w: %s", errProviderNotFound, target)
	}
	fileCfg.ActiveProvider = stringPointer(providers[index].Name)
	return s.persistLocked(configPath, fileCfg)
}
