package agentturn

import (
	"context"
	"errors"
	"strings"

	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
)

var ErrRunNotFound = errors.New("run not found")

func (s Service) Stop(
	ctx context.Context,
	params api.AgentStopParams,
	traceID string,
) (bus.ServiceResult, error) {
	sessionID := strings.TrimSpace(params.SessionID)
	stopTraceID := strings.TrimSpace(params.TraceID)
	if sessionID == "" && stopTraceID == "" {
		err := errors.New("session_id or trace_id is required")
		return bus.ServiceResult{}, bus.WrapError(bus.ServiceErrorInvalidInput, err)
	}
	if s.Stopper == nil {
		err := errors.New("run registry is not available")
		return bus.ServiceResult{}, bus.WrapError(bus.ServiceErrorUnavailable, err)
	}
	handle, err := s.cancel(ctx, sessionID, stopTraceID)
	if err != nil {
		return s.handleStopError(traceID, err)
	}
	s.log(traceID, bus.ActionAgentStop, "success", nil)
	return bus.ResultSuccess(api.AgentStopResponse{
		Status:    "stopped",
		Message:   "agent run cancelled successfully",
		SessionID: resolveStoppedSessionID(sessionID, handle),
	}), nil
}

func (s Service) cancel(ctx context.Context, sessionID string, traceID string) (StopHandle, error) {
	if sessionID != "" {
		return s.Stopper.CancelAndWaitBySessionID(ctx, sessionID)
	}
	return s.Stopper.CancelAndWaitByTraceID(ctx, traceID)
}

func (s Service) handleStopError(traceID string, err error) (bus.ServiceResult, error) {
	if errors.Is(err, ErrRunNotFound) {
		s.log(traceID, bus.ActionAgentStop, "not_running", nil)
		return bus.ResultSuccess(api.AgentStopResponse{
			Status:  "not_running",
			Message: "no active run found",
		}), nil
	}
	s.log(traceID, bus.ActionAgentStop, "error", err)
	return bus.ServiceResult{}, bus.WrapError(bus.ServiceErrorInternal, err)
}

func resolveStoppedSessionID(sessionID string, handle StopHandle) string {
	if trimmedSessionID := strings.TrimSpace(sessionID); trimmedSessionID != "" {
		return trimmedSessionID
	}
	return strings.TrimSpace(handle.SessionID)
}
