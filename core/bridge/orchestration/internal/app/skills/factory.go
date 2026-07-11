package skills

import (
	bridgeconfig "ghost-os/bridge/config"
	bridgeskills "ghost-os/bridge/skills"
)

type configStoreAdapter struct {
	inner bridgeconfig.Store
}

func (a configStoreAdapter) Config() (bridgeskills.Config, error) {
	cfg, err := a.inner.Config()
	if err != nil {
		return bridgeskills.Config{}, err
	}
	return bridgeskills.Config{
		ProjectRoot:    cfg.ProjectRoot,
		SkillBlocklist: append([]string(nil), cfg.SkillBlocklist...),
	}, nil
}

func (a configStoreAdapter) SetSkillEnabled(skillID string, enabled bool) error {
	return a.inner.SetSkillEnabled(skillID, enabled)
}

func NewActionHandler(store bridgeconfig.Store, log bridgeskills.LogFunc) *bridgeskills.ActionHandler {
	return bridgeskills.NewActionHandler(configStoreAdapter{inner: store}, log)
}
