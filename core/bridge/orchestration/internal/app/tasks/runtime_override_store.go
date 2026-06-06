package tasks

import (
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
