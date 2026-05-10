package orchestration

import (
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
