package mode

import (
	"errors"
	"strconv"
	"strings"
)

const (
	Pro  = "pro"
	Prox = "prox"

	StatusRunning    = "running"
	StatusCompleted  = "completed"
	StatusIncomplete = "incomplete"
	StatusCancelled  = "cancelled"
	StatusError      = "error"

	StopCompleted = "pro_complete"
	StopMaxLimit  = "max_iterations"
	StopCancelled = "cancelled"
	StopError     = "error"
)

type ProRequest struct {
	Mode          string
	OriginalTask  string
	MaxIterations int
	Unlimited     bool
}

func ParseProRequest(message string, defaultMaxIterations int) (ProRequest, bool, error) {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return ProRequest{}, false, nil
	}

	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return ProRequest{}, false, nil
	}

	mode := strings.ToLower(strings.TrimSpace(fields[0]))
	if mode != Pro && mode != Prox {
		return ProRequest{}, false, nil
	}

	request, err := parseProFields(mode, fields[1:], defaultMaxIterations)
	if err != nil {
		return ProRequest{}, true, err
	}
	return request, true, nil
}

func parseProFields(mode string, fields []string, defaultMaxIterations int) (ProRequest, error) {
	maxIterations, taskFields, err := splitProFields(fields)
	if err != nil {
		return ProRequest{}, err
	}
	task := strings.TrimSpace(strings.Join(taskFields, " "))
	if task == "" {
		return ProRequest{}, errors.New("pro/prox task is required")
	}

	request := ProRequest{Mode: mode, OriginalTask: task}
	if mode == Pro {
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

func splitProFields(fields []string) (int, []string, error) {
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

func ResultStatus(stoppedBy string) string {
	switch strings.TrimSpace(stoppedBy) {
	case StopCompleted:
		return StatusCompleted
	case StopMaxLimit:
		return StatusIncomplete
	case StopCancelled:
		return StatusCancelled
	default:
		return StatusError
	}
}
