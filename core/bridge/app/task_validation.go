package app

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	taskKindAgentMessage = "agent_message"
	taskKindSystemAction = "system_action"
)

func normalizeTaskKind(kind string) string {
	switch strings.TrimSpace(kind) {
	case taskKindSystemAction:
		return taskKindSystemAction
	default:
		return taskKindAgentMessage
	}
}

func validateTaskDefinition(task *ScheduledTask) error {
	if task == nil {
		return fmt.Errorf("%w: task is nil", ErrInvalidTaskConfig)
	}
	task.TaskKind = normalizeTaskKind(task.TaskKind)
	task.Action = strings.TrimSpace(task.Action)
	task.ActionParams = cloneTaskActionParams(task.ActionParams)

	switch task.TaskKind {
	case taskKindAgentMessage:
		if strings.TrimSpace(task.Message) == "" {
			return fmt.Errorf("%w: message is required", ErrInvalidTaskConfig)
		}
		task.Action = ""
		task.ActionParams = nil
	case taskKindSystemAction:
		task.Message = ""
		if strings.TrimSpace(task.SessionID) != "" {
			return fmt.Errorf("%w: system_action does not allow session_id", ErrInvalidTaskConfig)
		}
		if task.Action == "" {
			return fmt.Errorf("%w: action is required for system_action", ErrInvalidTaskConfig)
		}
		switch task.Action {
		case busActionMemoryHygieneRun:
			params, err := decodeMemoryHygieneRunParams(task.ActionParams)
			if err != nil {
				return fmt.Errorf("%w: invalid %s params: %v", ErrInvalidTaskConfig, busActionMemoryHygieneRun, err)
			}
			task.ActionParams = memoryHygieneRunParamsToMap(params)
		default:
			return fmt.Errorf("%w: unsupported system action %q", ErrInvalidTaskConfig, task.Action)
		}
	default:
		return fmt.Errorf("%w: unsupported task_kind %q", ErrInvalidTaskConfig, task.TaskKind)
	}
	return nil
}

func cloneTaskActionParams(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = cloneJSONValue(value)
	}
	return out
}

func cloneJSONValue(input any) any {
	switch typed := input.(type) {
	case map[string]any:
		return cloneTaskActionParams(typed)
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = cloneJSONValue(typed[i])
		}
		return out
	default:
		return typed
	}
}

func decodeActionParamsMap[T any](input map[string]any) (T, error) {
	var out T
	if len(input) == 0 {
		return out, nil
	}
	data, err := json.Marshal(input)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, err
	}
	return out, nil
}
