package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ghost-os/bridge/llm"

	"github.com/BurntSushi/toml"
)

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
	return cfg, nil
}

func loadBridgeFileConfig() (bridgeFileConfig, string, error) {
	cfg, path, _, err := loadBridgeFileConfigWithMigrationInfo()
	return cfg, path, err
}

// loadBridgeFileConfigWithMigrationInfo behaves like loadBridgeFileConfig but also reports
// whether the config file uses a legacy provider layout that can be migrated.
//
// IMPORTANT: This function is side-effect free; callers must explicitly invoke migration/write.
func loadBridgeFileConfigWithMigrationInfo() (bridgeFileConfig, string, bool, error) {
	configPath := configPathFromEnv()
	rawCfg, err := loadBridgeTomlConfig(configPath)
	switch {
	case err == nil:
		resolvedPath, resolveErr := resolveUserPath(configPath)
		if resolveErr != nil {
			return bridgeFileConfig{}, "", false, fmt.Errorf("resolve config path: %w", resolveErr)
		}
		needsMigration := hasLegacyProviderLayout(rawCfg)
		normalized := normalizeBridgeFileConfigForWrite(rawCfg)
		return normalized, resolvedPath, needsMigration, nil
	case errors.Is(err, os.ErrNotExist):
		resolvedPath, resolveErr := resolveUserPath(configPath)
		if resolveErr != nil {
			return bridgeFileConfig{}, "", false, fmt.Errorf("resolve config path: %w", resolveErr)
		}
		return bridgeFileConfig{}, resolvedPath, false, nil
	default:
		return bridgeFileConfig{}, configPath, false, fmt.Errorf("read config file %s: %w", configPath, err)
	}
}

// migrateBridgeFileConfigIfLegacy rewrites the configured TOML file to the current provider layout,
// but only when legacy provider fields are present.
//
// This migration is intentionally opt-in to avoid "read == write" surprises across server options
// and middleware that read config frequently.
func migrateBridgeFileConfigIfLegacy() (string, bool, error) {
	cfg, resolvedPath, needsMigration, err := loadBridgeFileConfigWithMigrationInfo()
	if err != nil {
		return resolvedPath, false, err
	}
	if !needsMigration {
		return resolvedPath, false, nil
	}
	if err := writeBridgeTomlConfig(configPathFromEnv(), cfg); err != nil {
		return resolvedPath, false, err
	}
	return resolvedPath, true, nil
}

func writeBridgeFileConfig(path string, cfg bridgeFileConfig) error {
	return writeBridgeTomlConfig(path, cfg)
}

func writeBridgeTomlConfig(path string, cfg bridgeFileConfig) error {
	resolvedPath, err := resolveUserPath(path)
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
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

func normalizeBridgeFileConfigForWrite(cfg bridgeFileConfig) bridgeFileConfig {
	out := cfg
	out.ActiveProvider = cloneOptionalStringPointer(out.ActiveProvider)
	out.Model = cloneOptionalStringPointer(out.Model)
	out.ChatPath = cloneOptionalStringPointer(out.ChatPath)
	out.ProjectRoot = cloneOptionalStringPointer(out.ProjectRoot)
	out.WorkerModel = cloneOptionalStringPointer(out.WorkerModel)
	out.PromptsPath = cloneOptionalStringPointer(out.PromptsPath)
	out.PromptsDir = cloneOptionalStringPointer(out.PromptsDir)
	out.PromptsCoreFiles = normalizeConfiguredPathList(out.PromptsCoreFiles)
	out.SessionsPath = cloneOptionalStringPointer(out.SessionsPath)
	out.RSSFeedsPath = cloneOptionalStringPointer(out.RSSFeedsPath)
	out.RSSInboxPath = cloneOptionalStringPointer(out.RSSInboxPath)
	out.RSSPollInterval = cloneOptionalStringPointer(out.RSSPollInterval)
	out.WebSearchTavilyAPIKey = cloneOptionalStringPointer(out.WebSearchTavilyAPIKey)
	out.AnthropicVersion = cloneOptionalStringPointer(out.AnthropicVersion)
	out.ToolSelectorMode = cloneOptionalStringPointer(out.ToolSelectorMode)
	out.ToolSelectorModel = cloneOptionalStringPointer(out.ToolSelectorModel)
	out.BindAddr = cloneOptionalStringPointer(out.BindAddr)
	out.APIToken = cloneOptionalStringPointer(out.APIToken)
	out.NativeBinaryPath = cloneOptionalStringPointer(out.NativeBinaryPath)
	out.NativeBinaryRoots = normalizeConfiguredPathList(out.NativeBinaryRoots)
	out.NativeBinaryCandidates = normalizeConfiguredPathList(out.NativeBinaryCandidates)
	out.NativeAllowedReadPaths = normalizeConfiguredPathList(out.NativeAllowedReadPaths)
	out.NativeAllowedWritePaths = normalizeConfiguredPathList(out.NativeAllowedWritePaths)
	out.ProviderHeaders, _ = normalizeProviderHeaders(out.ProviderHeaders)
	out.CORSOrigins = normalizeOrigins(out.CORSOrigins)
	out.ToolAllowlist = normalizeConfiguredToolNames(out.ToolAllowlist)
	out.ToolBlocklist = normalizeConfiguredToolNames(out.ToolBlocklist)

	providers := normalizeProviderConfigs(out.Providers, stringValue(out.Model))
	if len(providers) == 0 {
		providers = normalizeLegacyProviderConfigs(out.ModelProviders, stringValue(out.Model))
	}
	if len(providers) == 0 {
		legacyName := strings.TrimSpace(stringValue(out.Provider))
		legacyBaseURL := strings.TrimSpace(stringValue(out.BaseURL))
		legacyAPIKey := cloneOptionalStringPointer(out.APIKey)
		if legacyName != "" || legacyBaseURL != "" || legacyAPIKey != nil {
			if legacyName == "" {
				legacyName = string(defaultProvider)
			}
			providerType := inferProviderType(legacyName, legacyBaseURL, stringValue(out.Model))
			if providerType == "" {
				providerType = defaultProvider
			}
			if legacyBaseURL == "" {
				legacyBaseURL = defaultBaseURLForProvider(providerType)
			}
			providers = []providerConfig{{
				Name:    legacyName,
				Type:    providerType,
				BaseURL: legacyBaseURL,
				APIKey:  legacyAPIKey,
			}}
		}
	}

	out.Providers = providerConfigsToFileMap(providers)
	out.ActiveProvider = normalizedActiveProviderName(providers, out.ActiveProvider, out.ModelProvider)
	out.ModelProvider = nil
	out.ModelProviders = nil
	out.Provider = nil
	out.APIKey = nil
	out.BaseURL = nil
	return out
}

func hasLegacyProviderLayout(cfg bridgeFileConfig) bool {
	return cfg.ModelProvider != nil || len(cfg.ModelProviders) > 0 || cfg.Provider != nil || cfg.APIKey != nil || cfg.BaseURL != nil
}

func normalizedActiveProviderName(providers []providerConfig, preferred ...*string) *string {
	if len(providers) == 0 {
		return nil
	}
	for _, candidate := range preferred {
		name := strings.TrimSpace(stringValue(candidate))
		if providerIndexByName(providers, name) >= 0 {
			return stringPointer(name)
		}
	}
	return stringPointer(providers[0].Name)
}

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
	normalizedType := rawType.Normalized()
	if normalizedType == "" {
		normalizedType = inferProviderType(trimmedName, trimmedBaseURL, model)
	}
	if normalizedType == "" {
		normalizedType = defaultProvider
	}
	if trimmedName == "" && trimmedBaseURL == "" && cloneOptionalStringPointer(apiKey) == nil && len(normalizeProviderModels(models)) == 0 {
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
		APIKey:                     cloneOptionalStringPointer(apiKey),
		Models:                     normalizeProviderModels(models),
		ContextWindowTokens:        normalizePositiveInt(rawContextWindowTokens),
		ResponseReserveTokens:      normalizePositiveInt(rawResponseReserveTokens),
		ModelContextWindowTokens:   normalizeModelTokenOverrides(rawModelContextWindowTokens),
		ModelResponseReserveTokens: normalizeModelTokenOverrides(rawModelResponseReserveTokens),
	}, true
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
