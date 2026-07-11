package tasks

import (
	"fmt"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/orchestration/internal/domain/runtimeopts"
	bridgeTasks "ghost-os/bridge/tasks"
	bridgetools "ghost-os/bridge/tools"
)

type RuntimeProviderCatalog struct {
	ActiveProvider string
	Providers      []bridgeconfig.ProviderRecord
}

type runtimeOverrideNormalizationOptions struct {
	requireProviderModelPair bool
	dropMaxTurns             bool
}

func NormalizeTaskRuntimeOverrides(
	input *bridgeTasks.TaskRuntimeOverrides,
) (*bridgeTasks.TaskRuntimeOverrides, error) {
	return normalizeTaskRuntimeOverridesWithOptions(input, runtimeOverrideNormalizationOptions{})
}

func NormalizeWorkflowAgentRuntimeOverrides(
	input *bridgeTasks.TaskRuntimeOverrides,
) (*bridgeTasks.TaskRuntimeOverrides, error) {
	return normalizeTaskRuntimeOverridesWithOptions(input, runtimeOverrideNormalizationOptions{
		requireProviderModelPair: true,
	})
}

func NormalizeOrchestrationAgentRuntimeOverrides(
	input *bridgeTasks.TaskRuntimeOverrides,
) (*bridgeTasks.TaskRuntimeOverrides, error) {
	return normalizeTaskRuntimeOverridesWithOptions(input, runtimeOverrideNormalizationOptions{
		requireProviderModelPair: true,
		dropMaxTurns:             true,
	})
}

func NormalizeTaskRuntimeOverridesForCatalogValidation(
	input *bridgeTasks.TaskRuntimeOverrides,
	requireProviderModelPair bool,
) (*bridgeTasks.TaskRuntimeOverrides, error) {
	return normalizeTaskRuntimeOverridesWithOptions(input, runtimeOverrideNormalizationOptions{
		requireProviderModelPair: requireProviderModelPair,
	})
}

func LoadRuntimeProviderCatalog(store bridgeconfig.Store) (RuntimeProviderCatalog, error) {
	if store == nil {
		return RuntimeProviderCatalog{}, nil
	}
	providers, err := store.ListProviders()
	if err != nil {
		return RuntimeProviderCatalog{}, err
	}
	return RuntimeProviderCatalog{
		ActiveProvider: strings.TrimSpace(store.Snapshot().Provider),
		Providers:      append([]bridgeconfig.ProviderRecord(nil), providers...),
	}, nil
}

func ValidateRuntimeOverridesAgainstCatalog(
	overrides *bridgeTasks.TaskRuntimeOverrides,
	catalog RuntimeProviderCatalog,
	requireProviderModelPair bool,
) error {
	normalized, err := NormalizeTaskRuntimeOverridesForCatalogValidation(overrides, requireProviderModelPair)
	if err != nil || normalized == nil {
		return err
	}
	return runtimeopts.ValidateOverridesAgainstCatalog(
		normalized,
		runtimeProviderCatalogForDomain(catalog),
		requireProviderModelPair,
	)
}

func ApplyRuntimeOverridesToConfig(
	cfg bridgeconfig.Config,
	overrides *bridgeTasks.TaskRuntimeOverrides,
	catalog RuntimeProviderCatalog,
) (bridgeconfig.Config, error) {
	targetName := strings.TrimSpace(overrides.ProviderName)
	if targetName != "" {
		record, ok := findRuntimeProviderRecord(catalog.Providers, targetName)
		if !ok {
			return bridgeconfig.Config{}, fmt.Errorf("provider_name %q is not configured", targetName)
		}
		cfg = applyRuntimeProvider(cfg, record)
	}
	if model := strings.TrimSpace(overrides.Model); model != "" {
		cfg.Provider.Model = model
	}
	if RuntimeOverrideUsesToolScope(overrides) {
		cfg.ToolSelector.AllowlistOnly = true
		cfg.ToolSelector.Allowlist = append([]string(nil), overrides.ToolAllowlist...)
		cfg.ToolSelector.Blocklist = nil
	}
	if overrides.MaxTurns != nil {
		cfg.MaxTurns = *overrides.MaxTurns
	}
	return cfg, nil
}

func applyRuntimeProvider(
	cfg bridgeconfig.Config,
	record bridgeconfig.ProviderRecord,
) bridgeconfig.Config {
	cfg.Provider.Type = record.Type
	cfg.Provider.APIKey = providerRecordAPIKey(record)
	cfg.Provider.BaseURL = strings.TrimSpace(record.BaseURL)
	cfg.Provider.ContextWindowTokens = record.ContextWindowTokens
	cfg.Provider.ResponseReserveTokens = record.ResponseReserveTokens
	cfg.Provider.ModelContextWindowTokens = cloneModelTokenOverrides(record.ModelContextWindowTokens)
	cfg.Provider.ModelResponseReserveTokens = cloneModelTokenOverrides(record.ModelResponseReserveTokens)
	return cfg
}

func providerRecordAPIKey(record bridgeconfig.ProviderRecord) string {
	if record.APIKey == nil {
		return ""
	}
	return strings.TrimSpace(*record.APIKey)
}

func findRuntimeProviderRecord(
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

func RuntimeOverrideUsesToolScope(overrides *bridgeTasks.TaskRuntimeOverrides) bool {
	return runtimeopts.UsesToolScope(overrides)
}

func cloneModelTokenOverrides(raw map[string]int) map[string]int {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]int, len(raw))
	for key, value := range raw {
		out[strings.TrimSpace(key)] = value
	}
	return out
}

func normalizeTaskRuntimeOverridesWithOptions(
	input *bridgeTasks.TaskRuntimeOverrides,
	options runtimeOverrideNormalizationOptions,
) (*bridgeTasks.TaskRuntimeOverrides, error) {
	return runtimeopts.NormalizeTaskOverrides(input, runtimeopts.OverrideNormalizationOptions{
		RequireProviderModelPair: options.requireProviderModelPair,
		DropMaxTurns:             options.dropMaxTurns,
		ValidToolNames:           runtimeToolNames(),
	})
}

func runtimeProviderCatalogForDomain(catalog RuntimeProviderCatalog) runtimeopts.ProviderCatalog {
	providers := make([]runtimeopts.ProviderRecord, 0, len(catalog.Providers))
	for _, provider := range catalog.Providers {
		providers = append(providers, runtimeopts.ProviderRecord{
			Name:   strings.TrimSpace(provider.Name),
			Models: append([]string(nil), provider.Models...),
		})
	}
	return runtimeopts.ProviderCatalog{
		ActiveProvider: strings.TrimSpace(catalog.ActiveProvider),
		Providers:      providers,
	}
}

func runtimeToolNames() []string {
	metadata := bridgetools.GetToolMetadata()
	names := make([]string, 0, len(metadata))
	for _, item := range metadata {
		if name := strings.TrimSpace(item.Name); name != "" {
			names = append(names, name)
		}
	}
	return names
}
