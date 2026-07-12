package agentturn

import (
	"context"
	"errors"

	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
	bridgeTasks "ghost-os/bridge/tasks"
)

func (s Service) Execute(
	ctx context.Context,
	params api.AgentParams,
	runtimeOverrides *bridgeTasks.TaskRuntimeOverrides,
	traceID string,
) (bus.ServiceResult, error) {
	prepared, err := s.Prepare(params, runtimeOverrides, traceID)
	if err != nil {
		return bus.ServiceResult{}, err
	}
	return s.executeStandard(ctx, prepared, traceID)
}

func (s Service) Prepare(
	params api.AgentParams,
	runtimeOverrides *bridgeTasks.TaskRuntimeOverrides,
	traceID string,
) (PreparedRequest, error) {
	prepared, err := PrepareWithRuntimeOverrides(s.Guards, params, runtimeOverrides)
	if err != nil {
		if !errors.Is(err, ErrMessageRequired) {
			s.log(traceID, bus.ActionAgentSend, "error", err)
		}
		return PreparedRequest{}, err
	}
	return prepared, nil
}

func (s Service) executeStandard(
	ctx context.Context,
	prepared PreparedRequest,
	traceID string,
) (bus.ServiceResult, error) {
	if s.Runner == nil || s.Finalizer == nil {
		err := errors.New("agent turn runner is not configured")
		return bus.ServiceResult{}, bus.WrapError(bus.ServiceErrorInternal, err)
	}
	s.log(traceID, bus.ActionAgentSend, "running", nil)
	response, sessionID, err := s.Runner.RunTurn(ctx, prepared, traceID)
	if err != nil {
		return s.handleStandardError(traceID, sessionID, WrapErrorWithSessionID(err, sessionID))
	}
	return s.completeStandardTurn(traceID, response, sessionID)
}
