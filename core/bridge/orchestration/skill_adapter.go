package orchestration

import (
	bridgeskills "ghost-os/bridge/skills"

	bridgeconfig "ghost-os/bridge/config"
)

// configStoreAdapter adapts bridgeconfig.Store to skills.Store.
type configStoreAdapter struct {
	inner bridgeconfig.Store
}

func (a configStoreAdapter) Config() (bridgeskills.Config, error) {
	cfg, err := a.inner.Config()
	if err != nil {
		return bridgeskills.Config{}, err
	}
	return bridgeskills.Config{ProjectRoot: cfg.ProjectRoot}, nil
}

func NewSkillActionHandler(store bridgeconfig.Store, log bridgeskills.LogFunc) *bridgeskills.ActionHandler {
	return bridgeskills.NewActionHandler(configStoreAdapter{inner: store}, log)
}
