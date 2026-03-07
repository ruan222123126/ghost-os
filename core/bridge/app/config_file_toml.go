package app

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
)

type configPaths struct {
	Toml string
	YAML string
}

func resolveConfigPaths() (configPaths, error) {
	rawPath := strings.TrimSpace(os.Getenv("GHOST_CONFIG_PATH"))
	if rawPath == "" {
		tomlPath, err := resolveUserPath(defaultConfigPath)
		if err != nil {
			return configPaths{}, fmt.Errorf("resolve config path: %w", err)
		}
		yamlPath, err := resolveUserPath(defaultLegacyConfigPath)
		if err != nil {
			return configPaths{}, fmt.Errorf("resolve legacy config path: %w", err)
		}
		return configPaths{Toml: tomlPath, YAML: yamlPath}, nil
	}

	resolvedPath, err := resolveUserPath(rawPath)
	if err != nil {
		return configPaths{}, fmt.Errorf("resolve config path: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(resolvedPath))
	base := strings.TrimSuffix(resolvedPath, filepath.Ext(resolvedPath))
	switch ext {
	case ".yaml", ".yml":
		return configPaths{Toml: base + ".toml", YAML: resolvedPath}, nil
	case ".toml":
		return configPaths{Toml: resolvedPath, YAML: base + ".yaml"}, nil
	default:
		return configPaths{Toml: resolvedPath, YAML: resolvedPath + ".yaml"}, nil
	}
}

func loadBridgeTomlConfig(path string) (bridgeFileConfig, error) {
	resolvedPath, err := resolveUserPath(path)
	if err != nil {
		return bridgeFileConfig{}, fmt.Errorf("resolve config path: %w", err)
	}

	raw, err := os.ReadFile(resolvedPath)
	if err != nil {
		return bridgeFileConfig{}, err
	}
	if strings.TrimSpace(string(raw)) == "" {
		return bridgeFileConfig{}, nil
	}

	var cfg bridgeFileConfig
	if _, err := toml.Decode(string(raw), &cfg); err != nil {
		return bridgeFileConfig{}, fmt.Errorf("parse config file %s: %w", resolvedPath, err)
	}
	return normalizeBridgeFileConfigForWrite(cfg), nil
}

func loadBridgeFileConfig() (bridgeFileConfig, string, error) {
	paths, err := resolveConfigPaths()
	if err != nil {
		return bridgeFileConfig{}, "", err
	}

	cfg, err := loadBridgeTomlConfig(paths.Toml)
	switch {
	case err == nil:
		return cfg, paths.Toml, nil
	case !errors.Is(err, os.ErrNotExist):
		return bridgeFileConfig{}, paths.Toml, fmt.Errorf("read config file %s: %w", paths.Toml, err)
	}

	legacyCfg, err := loadLegacyBridgeYAMLConfig(paths.YAML)
	switch {
	case err == nil:
		if reflect.DeepEqual(legacyCfg, bridgeFileConfig{}) {
			return legacyCfg, paths.Toml, nil
		}
		migrated, migrateErr := migrateYamlToToml(paths.Toml, legacyCfg)
		if migrateErr != nil {
			return bridgeFileConfig{}, paths.Toml, migrateErr
		}
		return migrated, paths.Toml, nil
	case errors.Is(err, os.ErrNotExist):
		return bridgeFileConfig{}, paths.Toml, nil
	default:
		return bridgeFileConfig{}, paths.Toml, fmt.Errorf("read config file %s: %w", paths.YAML, err)
	}
}

func writeBridgeFileConfig(path string, cfg bridgeFileConfig) error {
	return writeBridgeTomlConfig(path, cfg)
}

func writeBridgeTomlConfig(path string, cfg bridgeFileConfig) error {
	resolvedPath, err := resolveWriteConfigPath(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(resolvedPath), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	normalized := normalizeBridgeFileConfigForWrite(cfg)
	var buffer bytes.Buffer
	if err := toml.NewEncoder(&buffer).Encode(normalized); err != nil {
		return fmt.Errorf("encode config file: %w", err)
	}
	if err := os.WriteFile(resolvedPath, buffer.Bytes(), 0o600); err != nil {
		return fmt.Errorf("write config file %s: %w", resolvedPath, err)
	}
	return nil
}

func migrateYamlToToml(tomlPath string, legacyCfg bridgeFileConfig) (bridgeFileConfig, error) {
	migrated := normalizeBridgeFileConfigForWrite(legacyCfg)
	if err := writeBridgeTomlConfig(tomlPath, migrated); err != nil {
		return bridgeFileConfig{}, fmt.Errorf("migrate yaml config to toml: %w", err)
	}
	return migrated, nil
}

func resolveWriteConfigPath(path string) (string, error) {
	resolvedPath, err := resolveUserPath(path)
	if err != nil {
		return "", fmt.Errorf("resolve config path: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(resolvedPath))
	if ext == ".yaml" || ext == ".yml" {
		return strings.TrimSuffix(resolvedPath, filepath.Ext(resolvedPath)) + ".toml", nil
	}
	return resolvedPath, nil
}

func normalizeBridgeFileConfigForWrite(cfg bridgeFileConfig) bridgeFileConfig {
	out := cfg
	out.ModelProvider = cloneOptionalStringPointer(out.ModelProvider)
	out.Model = cloneOptionalStringPointer(out.Model)
	out.ChatPath = cloneOptionalStringPointer(out.ChatPath)
	out.WorkerModel = cloneOptionalStringPointer(out.WorkerModel)
	out.PromptsPath = cloneOptionalStringPointer(out.PromptsPath)
	out.SessionsPath = cloneOptionalStringPointer(out.SessionsPath)
	out.MemoryWarmPath = cloneOptionalStringPointer(out.MemoryWarmPath)
	out.MemoryColdPath = cloneOptionalStringPointer(out.MemoryColdPath)
	out.MemoryGraphPath = cloneOptionalStringPointer(out.MemoryGraphPath)
	out.MemoryDecisionPath = cloneOptionalStringPointer(out.MemoryDecisionPath)
	out.MemoryWarmTTL = cloneOptionalStringPointer(out.MemoryWarmTTL)
	out.MemoryEvolutionInterval = cloneOptionalStringPointer(out.MemoryEvolutionInterval)
	out.MemoryGraphNamespace = cloneOptionalStringPointer(out.MemoryGraphNamespace)
	out.MemoryDecisionRecipeInterval = cloneOptionalStringPointer(out.MemoryDecisionRecipeInterval)
	out.AnthropicVersion = cloneOptionalStringPointer(out.AnthropicVersion)
	out.ToolSelectorMode = cloneOptionalStringPointer(out.ToolSelectorMode)
	out.ToolSelectorModel = cloneOptionalStringPointer(out.ToolSelectorModel)
	out.BindAddr = cloneOptionalStringPointer(out.BindAddr)
	out.APIToken = cloneOptionalStringPointer(out.APIToken)
	out.ProviderHeaders, _ = normalizeProviderHeaders(out.ProviderHeaders)
	out.CORSOrigins = normalizeOrigins(out.CORSOrigins)
	out.ModelProviders = normalizeProviderConfigs(out.ModelProviders)

	if len(out.ModelProviders) == 0 {
		legacyName := strings.TrimSpace(stringValue(out.Provider))
		legacyBaseURL := strings.TrimSpace(stringValue(out.BaseURL))
		legacyAPIKey := cloneOptionalStringPointer(out.APIKey)
		if legacyName != "" || legacyBaseURL != "" || legacyAPIKey != nil {
			if legacyName == "" {
				legacyName = string(defaultProvider)
			}
			providerType := inferProviderType(legacyName, legacyBaseURL, stringValue(out.Model))
			if legacyBaseURL == "" {
				legacyBaseURL = defaultBaseURLForProvider(providerType)
			}
			out.ModelProviders = []providerConfig{{
				Name:    legacyName,
				BaseURL: legacyBaseURL,
				APIKey:  legacyAPIKey,
			}}
			if out.ModelProvider == nil {
				out.ModelProvider = stringPointer(legacyName)
			}
		}
	}

	if len(out.ModelProviders) == 0 {
		out.ModelProvider = nil
	} else {
		activeName := strings.TrimSpace(stringValue(out.ModelProvider))
		activeIndex := providerIndexByName(out.ModelProviders, activeName)
		if activeIndex < 0 {
			out.ModelProvider = stringPointer(out.ModelProviders[0].Name)
		} else {
			out.ModelProvider = stringPointer(out.ModelProviders[activeIndex].Name)
		}
	}

	out.Provider = nil
	out.APIKey = nil
	out.BaseURL = nil
	return out
}

func normalizeProviderConfigs(providers []providerConfig) []providerConfig {
	if len(providers) == 0 {
		return nil
	}

	out := make([]providerConfig, 0, len(providers))
	for _, provider := range providers {
		normalized := providerConfig{
			Name:    strings.TrimSpace(provider.Name),
			BaseURL: strings.TrimSpace(provider.BaseURL),
			APIKey:  cloneOptionalStringPointer(provider.APIKey),
			Models:  normalizeProviderModels(provider.Models),
		}
		if normalized.Name == "" && normalized.BaseURL == "" && normalized.APIKey == nil && len(normalized.Models) == 0 {
			continue
		}
		out = append(out, normalized)
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
		cloned := providerConfig{
			Name:    provider.Name,
			BaseURL: provider.BaseURL,
			APIKey:  cloneOptionalStringPointer(provider.APIKey),
			Models:  append([]string(nil), provider.Models...),
		}
		out = append(out, cloned)
	}
	return out
}
