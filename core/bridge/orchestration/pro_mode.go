package orchestration

import (
	"context"
	"errors"
	"net/http"

	bridgemode "ghost-os/bridge/mode"
	"ghost-os/bridge/session"
)

const (
	proModePro              = bridgemode.Pro
	proModeStatusRunning    = bridgemode.StatusRunning
	proModeStatusCompleted  = bridgemode.StatusCompleted
	proModeStatusIncomplete = bridgemode.StatusIncomplete
	proModeStatusCancelled  = bridgemode.StatusCancelled
	proModeStatusError      = bridgemode.StatusError
	proModeStopCompleted    = bridgemode.StopCompleted
	proModeStopMaxLimit     = bridgemode.StopMaxLimit
	proModeStopCancelled    = bridgemode.StopCancelled
	proModeStopError        = bridgemode.StopError
)

type proModeRequest = bridgemode.ProRequest

type proModeResult struct {
	Message        string
	StoppedBy      string
	FinalChangeLog string
	Records        []session.IterationRecord
}

func parseProModeRequest(message string, defaultMaxIterations int) (proModeRequest, bool, error) {
	return bridgemode.ParseProRequest(message, defaultMaxIterations)
}

func (s *bridgeService) executeProModeAction(ctx context.Context, prepared preparedAgentTurnRequest, traceID string) (agentResponse, int, error) {
	payload, err := newProModeRunner(s).Execute(ctx, prepared, traceID)
	if err != nil {
		statusCode := mapProModeError(err)
		if errors.Is(err, session.ErrSessionNotFound) {
			return agentResponse{}, statusCode, err
		}
		normalizedKind, normalizedErr := normalizeAgentExecutionError(err)
		if normalizedKind != ServiceErrorInternal {
			return agentResponse{}, legacyStatusFromServiceErrorKind(normalizedKind), normalizedErr
		}
		return agentResponse{}, statusCode, err
	}
	return payload, http.StatusOK, nil
}

func mapProModeError(err error) int {
	switch {
	case errors.Is(err, session.ErrInvalidSessionID):
		return http.StatusBadRequest
	case errors.Is(err, session.ErrSessionNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrSessionInflight), errors.Is(err, context.Canceled):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func proModeResultStatus(stoppedBy string) string {
	return bridgemode.ResultStatus(stoppedBy)
}
