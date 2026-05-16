package orchestration

import (
	"fmt"
	"strings"

	bridgeTasks "ghost-os/bridge/tasks"
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
	case taskKindOrchestration:
		return validateOrchestrationTaskDefinition(task)
	default:
		return fmt.Errorf("%w: unsupported task_kind %q", ErrInvalidTaskConfig, task.TaskKind)
	}
}

func normalizeTaskDefinition(task *ScheduledTask) {
	task.Name = strings.TrimSpace(task.Name)
	task.Message = strings.TrimSpace(task.Message)
	task.SessionID = strings.TrimSpace(task.SessionID)
	task.RuntimeOverrides = cloneTaskRuntimeOverrides(task.RuntimeOverrides)
	task.AgentMode = bridgeTasks.NormalizeAgentMode(task.AgentMode)
	task.Relay = bridgeTasks.CloneTaskRelayConfig(task.Relay)
	task.TaskKind = normalizeTaskKind(task.TaskKind)
	task.Action = strings.TrimSpace(task.Action)
	task.ActionParams = cloneTaskActionParams(task.ActionParams)
	task.Workflow = cloneTaskWorkflow(task.Workflow)
	task.Orchestration = cloneTaskOrchestration(task.Orchestration)
}

func validateAgentTaskDefinition(task *ScheduledTask) error {
	if strings.TrimSpace(task.Message) == "" {
		return fmt.Errorf("%w: message is required", ErrInvalidTaskConfig)
	}
	task.Name = ""
	task.Orchestration = nil
	if err := validateAgentTaskMode(task); err != nil {
		return err
	}
	if task.Workflow != nil {
		return fmt.Errorf("%w: agent_message does not allow workflow", ErrInvalidTaskConfig)
	}
	task.Action = ""
	task.ActionParams = nil
	return nil
}

func validateAgentTaskMode(task *ScheduledTask) error {
	switch task.AgentMode {
	case "", taskAgentModeSingle:
		task.AgentMode = taskAgentModeSingle
		task.Relay = nil
		return nil
	case taskAgentModeRelay:
		return validateTaskRelayConfig(task.Relay)
	default:
		return fmt.Errorf("%w: unsupported agent_mode %q", ErrInvalidTaskConfig, task.AgentMode)
	}
}

func validateTaskRelayConfig(relay *TaskRelayConfig) error {
	if relay == nil {
		return nil
	}
	policy := strings.TrimSpace(relay.StopPolicy)
	switch policy {
	case taskRelayStopPolicyAIDecides, taskRelayStopPolicyMaxRounds:
		if relay.MaxRounds <= 0 {
			return fmt.Errorf("%w: relay max_rounds must be > 0", ErrInvalidTaskConfig)
		}
	default:
		return fmt.Errorf("%w: unsupported relay stop_policy %q", ErrInvalidTaskConfig, relay.StopPolicy)
	}
	if relay.ExecutionTimeoutMS != nil && *relay.ExecutionTimeoutMS < 0 {
		return fmt.Errorf("%w: relay execution_timeout_ms must be >= 0", ErrInvalidTaskConfig)
	}
	return nil
}

func validateSystemTaskDefinition(task *ScheduledTask) error {
	task.Name = ""
	task.Message = ""
	task.Orchestration = nil
	if strings.TrimSpace(task.SessionID) != "" {
		return fmt.Errorf("%w: system_action does not allow session_id", ErrInvalidTaskConfig)
	}
	if task.Workflow != nil {
		return fmt.Errorf("%w: system_action does not allow workflow", ErrInvalidTaskConfig)
	}
	if task.Action == "" {
		return fmt.Errorf("%w: action is required for system_action", ErrInvalidTaskConfig)
	}
	return fmt.Errorf("%w: unsupported system action %q", ErrInvalidTaskConfig, task.Action)
}
