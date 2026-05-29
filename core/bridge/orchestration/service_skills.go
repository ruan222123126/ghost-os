package orchestration

import (
	bridgeconfig "ghost-os/bridge/config"
	appskills "ghost-os/bridge/orchestration/internal/app/skills"
	bridgeskills "ghost-os/bridge/skills"
)

// Skill action methods on bridgeService delegate to skillHandler.
// All business logic lives in bridge/skills.ActionHandler.

func (s *bridgeService) executeSkillListAction(traceID string) (any, int, error) {
	return s.skillService().List(traceID)
}

func (s *bridgeService) executeSkillUpdateAction(
	params bridgeskills.SkillIDParams,
	req bridgeskills.SkillUpdateRequest,
	traceID string,
) (any, int, error) {
	return s.skillService().Update(params, req, traceID)
}

func (s *bridgeService) executeSkillDeleteAction(params bridgeskills.SkillIDParams, traceID string) (any, int, error) {
	return s.skillService().Delete(params, traceID)
}

func (s *bridgeService) skillService() appskills.Service {
	return appskills.Service{Handler: s.skillHandler}
}

// configStoreAdapter adapts bridgeconfig.Store to skills.Store.
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

func NewSkillActionHandler(store bridgeconfig.Store, log bridgeskills.LogFunc) *bridgeskills.ActionHandler {
	return bridgeskills.NewActionHandler(configStoreAdapter{inner: store}, log)
}
