package orchestration

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

const (
	proModePro              = "pro"
	proModeProx             = "prox"
	proModeStatusRunning    = "running"
	proModeStatusCompleted  = "completed"
	proModeStatusIncomplete = "incomplete"
	proModeStatusCancelled  = "cancelled"
	proModeStatusError      = "error"
	proModeStopCompleted    = "pro_complete"
	proModeStopMaxLimit     = "max_iterations"
	proModeStopCancelled    = "cancelled"
	proModeStopError        = "error"
)

type proModeRequest struct {
	Mode          string
	OriginalTask  string
	MaxIterations int
	Unlimited     bool
}

type proModeResult struct {
	Message        string
	StoppedBy      string
	FinalChangeLog string
	Records        []session.IterationRecord
}

type proModeCatalog struct {
	base   tools.ToolCatalog
	extra  map[string]tools.Tool
	hidden map[string]bool
}

func parseProModeRequest(message string, defaultMaxIterations int) (proModeRequest, bool, error) {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return proModeRequest{}, false, nil
	}

	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return proModeRequest{}, false, nil
	}

	mode := strings.ToLower(strings.TrimSpace(fields[0]))
	if mode != proModePro && mode != proModeProx {
		return proModeRequest{}, false, nil
	}

	request, err := parseProModeFields(mode, fields[1:], defaultMaxIterations)
	if err != nil {
		return proModeRequest{}, true, err
	}
	return request, true, nil
}

func parseProModeFields(mode string, fields []string, defaultMaxIterations int) (proModeRequest, error) {
	maxIterations, taskFields, err := splitProModeFields(fields)
	if err != nil {
		return proModeRequest{}, err
	}
	task := strings.TrimSpace(strings.Join(taskFields, " "))
	if task == "" {
		return proModeRequest{}, errors.New("pro/prox task is required")
	}

	request := proModeRequest{Mode: mode, OriginalTask: task}
	if mode == proModePro {
		if maxIterations <= 0 {
			maxIterations = defaultMaxIterations
		}
		request.MaxIterations = maxIterations
		return request, nil
	}

	request.MaxIterations = maxIterations
	request.Unlimited = maxIterations == 0
	return request, nil
}

func splitProModeFields(fields []string) (int, []string, error) {
	if len(fields) == 0 {
		return 0, nil, nil
	}
	parsed, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, fields, nil
	}
	if parsed <= 0 {
		return 0, nil, errors.New("pro/prox max iterations must be > 0")
	}
	return parsed, fields[1:], nil
}

func (s *bridgeService) executeProModeAction(ctx context.Context, prepared preparedAgentTurnRequest, traceID string) (agentResponse, int, error) {
	payload, err := newProModeRunner(s).Execute(ctx, prepared, traceID)
	if err != nil {
		statusCode := mapProModeError(err)
		if errors.Is(err, session.ErrSessionNotFound) {
			return agentResponse{}, statusCode, err
		}
		normalizedErr, normalizedCode := normalizeAgentExecutionError(err)
		if normalizedCode != http.StatusInternalServerError {
			return agentResponse{}, normalizedCode, normalizedErr
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
	switch strings.TrimSpace(stoppedBy) {
	case proModeStopCompleted:
		return proModeStatusCompleted
	case proModeStopMaxLimit:
		return proModeStatusIncomplete
	case proModeStopCancelled:
		return proModeStatusCancelled
	default:
		return proModeStatusError
	}
}
