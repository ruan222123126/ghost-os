package orchestration

import bridgeskills "ghost-os/bridge/skills"

// Skill action methods on bridgeService delegate to skillHandler.
// All business logic lives in bridge/skills.ActionHandler.

func (s *bridgeService) executeSkillListAction(traceID string) (any, int, error) {
	return s.skillHandler.ExecuteListAction(traceID)
}

func (s *bridgeService) executeSkillUpdateAction(
	params bridgeskills.SkillIDParams,
	req bridgeskills.SkillUpdateRequest,
	traceID string,
) (any, int, error) {
	return s.skillHandler.ExecuteUpdateAction(params, req, traceID)
}

func (s *bridgeService) executeSkillDeleteAction(params bridgeskills.SkillIDParams, traceID string) (any, int, error) {
	return s.skillHandler.ExecuteDeleteAction(params, traceID)
}
