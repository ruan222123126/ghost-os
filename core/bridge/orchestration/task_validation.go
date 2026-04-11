package orchestration

import (
	"fmt"
	"strings"

	bridgerss "ghost-os/bridge/rss"
)

func validateTaskDefinition(task *ScheduledTask) error {
	if task == nil {
		return fmt.Errorf("%w: task is nil", ErrInvalidTaskConfig)
	}
	normalizeTaskDefinition(task)

	switch task.TaskKind {
	case taskKindAgentMessage:
		return validateAgentTaskDefinition(task)
	case taskKindSystemAction:
		return validateSystemTaskDefinition(task)
	case taskKindWorkflow:
		return validateWorkflowTaskDefinition(task)
	default:
		return fmt.Errorf("%w: unsupported task_kind %q", ErrInvalidTaskConfig, task.TaskKind)
	}
}

func normalizeTaskDefinition(task *ScheduledTask) {
	task.Message = strings.TrimSpace(task.Message)
	task.SessionID = strings.TrimSpace(task.SessionID)
	task.RuntimeOverrides = cloneTaskRuntimeOverrides(task.RuntimeOverrides)
	task.TaskKind = normalizeTaskKind(task.TaskKind)
	task.Action = strings.TrimSpace(task.Action)
	task.ActionParams = cloneTaskActionParams(task.ActionParams)
	task.Workflow = cloneTaskWorkflow(task.Workflow)
}

func validateAgentTaskDefinition(task *ScheduledTask) error {
	if strings.TrimSpace(task.Message) == "" {
		return fmt.Errorf("%w: message is required", ErrInvalidTaskConfig)
	}
	if task.Workflow != nil {
		return fmt.Errorf("%w: agent_message does not allow workflow", ErrInvalidTaskConfig)
	}
	task.Action = ""
	task.ActionParams = nil
	return nil
}

func validateSystemTaskDefinition(task *ScheduledTask) error {
	task.Message = ""
	if strings.TrimSpace(task.SessionID) != "" {
		return fmt.Errorf("%w: system_action does not allow session_id", ErrInvalidTaskConfig)
	}
	if task.Workflow != nil {
		return fmt.Errorf("%w: system_action does not allow workflow", ErrInvalidTaskConfig)
	}
	if task.Action == "" {
		return fmt.Errorf("%w: action is required for system_action", ErrInvalidTaskConfig)
	}
	if err := bridgerss.ValidateSystemTaskParams(task); err != nil {
		return err
	}
	return nil
}
