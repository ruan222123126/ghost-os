package tasks

import (
	"strings"

	bridgeconfig "ghost-os/bridge/config"
	bridgeTasks "ghost-os/bridge/tasks"
)

type runtimeOverrideStore struct {
	bridgeconfig.Store
	overrides *bridgeTasks.TaskRuntimeOverrides
}

func NewRuntimeOverrideStore(
	store bridgeconfig.Store,
	overrides *bridgeTasks.TaskRuntimeOverrides,
) bridgeconfig.Store {
	if store == nil || overrides == nil {
		return store
	}
	return runtimeOverrideStore{
		Store:     store,
		overrides: bridgeTasks.CloneTaskRuntimeOverrides(overrides),
	}
}

func (s runtimeOverrideStore) Config() (bridgeconfig.Config, error) {
	cfg, err := s.Store.Config()
	if err != nil {
		return bridgeconfig.Config{}, err
	}
	normalized, err := NormalizeTaskRuntimeOverrides(s.overrides)
	if err != nil || normalized == nil {
		return cfg, err
	}
	catalog, err := LoadRuntimeProviderCatalog(s.Store)
	if err != nil {
		return bridgeconfig.Config{}, err
	}
	if err := ValidateRuntimeOverridesAgainstCatalog(normalized, catalog, false); err != nil {
		return bridgeconfig.Config{}, err
	}
	return ApplyRuntimeOverridesToConfig(cfg, normalized, catalog)
}

func (s runtimeOverrideStore) Snapshot() bridgeconfig.Snapshot {
	snapshot := s.Store.Snapshot()
	normalized, err := NormalizeTaskRuntimeOverrides(s.overrides)
	if err != nil || normalized == nil {
		return snapshot
	}
	return applyRuntimeOverridesToSnapshot(snapshot, normalized, s.Store)
}

func applyRuntimeOverridesToSnapshot(
	snapshot bridgeconfig.Snapshot,
	overrides *bridgeTasks.TaskRuntimeOverrides,
	store bridgeconfig.Store,
) bridgeconfig.Snapshot {
	if overrides == nil {
		return snapshot
	}
	if providerName := strings.TrimSpace(overrides.ProviderName); providerName != "" {
		snapshot.Provider = providerName
		if provider, ok := findProviderRecord(store, providerName); ok {
			snapshot.ProviderType = string(provider.Type)
			snapshot.BaseURL = provider.BaseURL
			snapshot.APIKeySet = provider.APIKey != nil && strings.TrimSpace(*provider.APIKey) != ""
		}
	}
	if model := strings.TrimSpace(overrides.Model); model != "" {
		snapshot.Model = model
	}
	return snapshot
}

func findProviderRecord(store bridgeconfig.Store, name string) (bridgeconfig.ProviderRecord, bool) {
	if store == nil {
		return bridgeconfig.ProviderRecord{}, false
	}
	providers, err := store.ListProviders()
	if err != nil {
		return bridgeconfig.ProviderRecord{}, false
	}
	for _, provider := range providers {
		if strings.EqualFold(strings.TrimSpace(provider.Name), strings.TrimSpace(name)) {
			return provider, true
		}
	}
	return bridgeconfig.ProviderRecord{}, false
}
