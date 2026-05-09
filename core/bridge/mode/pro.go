package mode

import (
	"errors"
	"strconv"
	"strings"
)

const (
	Pro = "pro"

	removedProxMode = "prox"

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
	if mode == removedProxMode {
		return ProRequest{}, true, errors.New("prox mode has been removed; use pro <max_iterations> <task>")
	}
	if mode != Pro {
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
		return ProRequest{}, errors.New("pro task is required")
	}

	if maxIterations <= 0 {
		maxIterations = defaultMaxIterations
	}
	if maxIterations <= 0 {
		return ProRequest{}, errors.New("pro max iterations must be > 0")
	}
	return ProRequest{Mode: mode, OriginalTask: task, MaxIterations: maxIterations}, nil
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
		return 0, nil, errors.New("pro max iterations must be > 0")
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
