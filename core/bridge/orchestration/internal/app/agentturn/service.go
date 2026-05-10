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
	if result, handled, err := s.executeSpecial(ctx, prepared, traceID); handled {
		return result, err
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

func (s Service) executeSpecial(
	ctx context.Context,
	prepared PreparedRequest,
	traceID string,
) (bus.ServiceResult, bool, error) {
	if prepared.Mode == ModePlan {
		result, err := s.executePlan(ctx, prepared, traceID)
		return result, true, err
	}
	return bus.ServiceResult{}, false, nil
}

func (s Service) executePlan(
	ctx context.Context,
	prepared PreparedRequest,
	traceID string,
) (bus.ServiceResult, error) {
	if s.Special == nil {
		return specialRunnerMissing()
	}
	if prepared.RuntimeOverrides != nil {
		err := errors.New("runtime_overrides are not supported in plan mode")
		s.log(traceID, bus.ActionAgentSend, "error", err)
		return bus.ServiceResult{}, bus.WrapError(bus.ServiceErrorInvalidInput, err)
	}
	return s.executeSpecialTurn(ctx, prepared, traceID, s.Special.RunPlan)
}

func (s Service) executePro(
	ctx context.Context,
	prepared PreparedRequest,
	traceID string,
) (bus.ServiceResult, error) {
	if s.Special == nil {
		return specialRunnerMissing()
	}
	if prepared.RuntimeOverrides != nil {
		err := errors.New("runtime_overrides are not supported in pro mode")
		s.log(traceID, bus.ActionAgentSend, "error", err)
		return bus.ServiceResult{}, bus.WrapError(bus.ServiceErrorInvalidInput, err)
	}
	return s.executeSpecialTurn(ctx, prepared, traceID, s.Special.RunPro)
}

func specialRunnerMissing() (bus.ServiceResult, error) {
	err := errors.New("special mode runner is not configured")
	return bus.ServiceResult{}, bus.WrapError(bus.ServiceErrorInternal, err)
}

func (s Service) executeSpecialTurn(
	ctx context.Context,
	prepared PreparedRequest,
	traceID string,
	run func(context.Context, PreparedRequest, string) (api.AgentResponse, int, error),
) (bus.ServiceResult, error) {
	s.log(traceID, bus.ActionAgentSend, "running", nil)
	payload, code, err := run(ctx, prepared, traceID)
	if err != nil {
		s.log(traceID, bus.ActionAgentSend, "error", err)
		return bus.ServiceResult{}, bus.WrapError(bus.ErrorKindFromStatus(code), err)
	}
	s.publishSpecial(traceID, payload)
	return resultFromStatus(payload, code), nil
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
		return s.handleStandardError(traceID, sessionID, err)
	}
	return s.completeStandardTurn(traceID, response, sessionID)
}

func resultFromStatus(payload any, statusCode int) bus.ServiceResult {
	switch bus.OutcomeFromStatus(statusCode) {
	case bus.ServiceOutcomeCreated:
		return bus.ResultCreated(payload)
	case bus.ServiceOutcomeAccepted:
		return bus.ResultAccepted(payload)
	default:
		return bus.ResultSuccess(payload)
	}
}
