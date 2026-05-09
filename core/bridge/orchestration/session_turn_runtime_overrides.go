package orchestration

import (
	"fmt"
	"slices"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
)

type taskRuntimeOverrideStore struct {
	bridgeconfig.Store
	overrides *TaskRuntimeOverrides
}

type taskRuntimeProviderCatalog struct {
	activeProvider string
	providers      []bridgeconfig.ProviderRecord
}

func newTaskRuntimeOverrideStore(
	store bridgeconfig.Store,
	overrides *TaskRuntimeOverrides,
) bridgeconfig.Store {
	if store == nil || overrides == nil {
		return store
	}
	return taskRuntimeOverrideStore{
		Store:     store,
		overrides: cloneTaskRuntimeOverrides(overrides),
	}
}

func (s taskRuntimeOverrideStore) Config() (bridgeconfig.Config, error) {
	cfg, err := s.Store.Config()
	if err != nil {
		return bridgeconfig.Config{}, err
	}
	normalized, err := normalizeTaskRuntimeOverrides(s.overrides)
	if err != nil || normalized == nil {
		return cfg, err
	}
	catalog, err := loadTaskRuntimeProviderCatalog(s.Store)
	if err != nil {
		return bridgeconfig.Config{}, err
	}
	if err := validateTaskRuntimeOverridesAgainstCatalog(normalized, catalog, false); err != nil {
		return bridgeconfig.Config{}, err
	}
	return applyTaskRuntimeOverridesToConfig(cfg, normalized, catalog)
}

func loadTaskRuntimeProviderCatalog(store bridgeconfig.Store) (taskRuntimeProviderCatalog, error) {
	if store == nil {
		return taskRuntimeProviderCatalog{}, nil
	}
	providers, err := store.ListProviders()
	if err != nil {
		return taskRuntimeProviderCatalog{}, err
	}
	return taskRuntimeProviderCatalog{
		activeProvider: strings.TrimSpace(store.Snapshot().Provider),
		providers:      append([]bridgeconfig.ProviderRecord(nil), providers...),
	}, nil
}

func validateTaskRuntimeOverridesAgainstCatalog(
	overrides *TaskRuntimeOverrides,
	catalog taskRuntimeProviderCatalog,
	requireProviderModelPair bool,
) error {
	normalized, err := normalizeRuntimeOverridesForCatalogValidation(overrides, requireProviderModelPair)
	if err != nil || normalized == nil {
		return err
	}
	if strings.TrimSpace(normalized.ProviderName) == "" && strings.TrimSpace(normalized.Model) == "" {
		return nil
	}
	targetName := strings.TrimSpace(normalized.ProviderName)
	if targetName == "" {
		targetName = strings.TrimSpace(catalog.activeProvider)
	}
	record, ok := findTaskRuntimeProviderRecord(catalog.providers, targetName)
	if !ok {
		return fmt.Errorf("provider_name %q is not configured", targetName)
	}
	if !taskRuntimeModelExists(record.Models, normalized.Model) {
		return fmt.Errorf("model %q is not configured for provider %q", normalized.Model, record.Name)
	}
	return nil
}

func normalizeRuntimeOverridesForCatalogValidation(
	overrides *TaskRuntimeOverrides,
	requireProviderModelPair bool,
) (*TaskRuntimeOverrides, error) {
	options := taskRuntimeOverrideNormalizationOptions{requireProviderModelPair: requireProviderModelPair}
	return normalizeTaskRuntimeOverridesWithOptions(overrides, options)
}

func applyTaskRuntimeOverridesToConfig(
	cfg bridgeconfig.Config,
	overrides *TaskRuntimeOverrides,
	catalog taskRuntimeProviderCatalog,
) (bridgeconfig.Config, error) {
	targetName := strings.TrimSpace(overrides.ProviderName)
	if targetName != "" {
		record, ok := findTaskRuntimeProviderRecord(catalog.providers, targetName)
		if !ok {
			return bridgeconfig.Config{}, fmt.Errorf("provider_name %q is not configured", targetName)
		}
		cfg = applyTaskRuntimeProvider(cfg, record)
	}
	if model := strings.TrimSpace(overrides.Model); model != "" {
		cfg.Provider.Model = model
	}
	if taskRuntimeOverrideUsesToolScope(overrides) {
		cfg.ToolSelector.AllowlistOnly = true
		cfg.ToolSelector.Allowlist = append([]string(nil), overrides.ToolAllowlist...)
		cfg.ToolSelector.Blocklist = nil
	}
	if overrides.MaxTurns != nil {
		cfg.MaxTurns = *overrides.MaxTurns
	}
	return cfg, nil
}

func applyTaskRuntimeProvider(
	cfg bridgeconfig.Config,
	record bridgeconfig.ProviderRecord,
) bridgeconfig.Config {
	cfg.Provider.Type = record.Type
	cfg.Provider.APIKey = strings.TrimSpace(stringValue(record.APIKey))
	cfg.Provider.BaseURL = strings.TrimSpace(record.BaseURL)
	cfg.Provider.ContextWindowTokens = record.ContextWindowTokens
	cfg.Provider.ResponseReserveTokens = record.ResponseReserveTokens
	cfg.Provider.ModelContextWindowTokens = cloneModelTokenOverrides(record.ModelContextWindowTokens)
	cfg.Provider.ModelResponseReserveTokens = cloneModelTokenOverrides(record.ModelResponseReserveTokens)
	return cfg
}

func findTaskRuntimeProviderRecord(
	providers []bridgeconfig.ProviderRecord,
	name string,
) (bridgeconfig.ProviderRecord, bool) {
	target := strings.TrimSpace(name)
	for _, provider := range providers {
		if strings.EqualFold(strings.TrimSpace(provider.Name), target) {
			return provider, true
		}
	}
	return bridgeconfig.ProviderRecord{}, false
}

func taskRuntimeModelExists(models []string, model string) bool {
	target := strings.TrimSpace(model)
	if target == "" {
		return false
	}
	return slices.Contains(normalizeTaskRuntimeModels(models), target)
}

func normalizeTaskRuntimeModels(models []string) []string {
	if len(models) == 0 {
		return nil
	}
	out := make([]string, 0, len(models))
	seen := make(map[string]bool, len(models))
	for _, raw := range models {
		model := strings.TrimSpace(raw)
		if model == "" || seen[model] {
			continue
		}
		seen[model] = true
		out = append(out, model)
	}
	return out
}

func taskRuntimeOverrideUsesToolScope(overrides *TaskRuntimeOverrides) bool {
	if overrides == nil {
		return false
	}
	if overrides.ToolAllowlistOnly != nil && *overrides.ToolAllowlistOnly {
		return true
	}
	return overrides.ToolAllowlistOnly == nil && len(overrides.ToolAllowlist) > 0
}
