package skills

import bridgeskills "ghost-os/bridge/skills"

type Handler interface {
	ExecuteListAction(traceID string) (any, int, error)
	ExecuteUpdateAction(
		params bridgeskills.SkillIDParams,
		req bridgeskills.SkillUpdateRequest,
		traceID string,
	) (any, int, error)
	ExecuteDeleteAction(params bridgeskills.SkillIDParams, traceID string) (any, int, error)
}

type Service struct {
	Handler Handler
}

func (s Service) List(traceID string) (any, int, error) {
	return s.Handler.ExecuteListAction(traceID)
}

func (s Service) Update(
	params bridgeskills.SkillIDParams,
	req bridgeskills.SkillUpdateRequest,
	traceID string,
) (any, int, error) {
	return s.Handler.ExecuteUpdateAction(params, req, traceID)
}

func (s Service) Delete(params bridgeskills.SkillIDParams, traceID string) (any, int, error) {
	return s.Handler.ExecuteDeleteAction(params, traceID)
}
