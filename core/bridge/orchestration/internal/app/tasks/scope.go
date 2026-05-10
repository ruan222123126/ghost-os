package tasks

import (
	"errors"
	"fmt"
	"strings"

	"ghost-os/bridge/orchestration/internal/contracts/api"
	bridgeTasks "ghost-os/bridge/tasks"
)

const (
	ScopeUser          = "user"
	ScopeSystem        = "system"
	ScopeOrchestration = "orchestration"
)

func NormalizeID(id string) (string, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return "", fmt.Errorf("%w: task id is required", bridgeTasks.ErrInvalidTaskID)
	}
	return trimmed, nil
}

func InvalidConfig(message string) error {
	return fmt.Errorf("%w: %s", bridgeTasks.ErrInvalidTaskConfig, message)
}

func WrapConfigError(err error) error {
	if err == nil || errors.Is(err, bridgeTasks.ErrInvalidTaskConfig) {
		return err
	}
	return fmt.Errorf("%w: %v", bridgeTasks.ErrInvalidTaskConfig, err)
}

func EnsureMatchesScope(task bridgeTasks.ScheduledTask, scope string) error {
	if IncludeInScope(task, scope) {
		return nil
	}
	return bridgeTasks.ErrTaskNotFound
}

func IncludeInScope(task bridgeTasks.ScheduledTask, scope string) bool {
	kind := bridgeTasks.NormalizeKind(task.TaskKind)
	switch strings.TrimSpace(scope) {
	case "", ScopeUser:
		return kind == bridgeTasks.KindAgentMessage || kind == bridgeTasks.KindWorkflow
	case ScopeSystem:
		return kind == bridgeTasks.KindSystemAction
	case ScopeOrchestration:
		return kind == bridgeTasks.KindOrchestration
	default:
		return false
	}
}

func BuildPayload(task bridgeTasks.ScheduledTask) api.TaskPayload {
	payload := CloneScheduledTask(task)
	payload.TaskKind = bridgeTasks.NormalizeKind(task.TaskKind)
	return payload
}

func BuildRunLogPayload(run bridgeTasks.RunLog) api.TaskRunLogPayload {
	return run
}

func CloneScheduledTask(task bridgeTasks.ScheduledTask) bridgeTasks.ScheduledTask {
	cloned := task
	cloned.RuntimeOverrides = bridgeTasks.CloneTaskRuntimeOverrides(task.RuntimeOverrides)
	cloned.Relay = bridgeTasks.CloneTaskRelayConfig(task.Relay)
	cloned.ActionParams = bridgeTasks.CloneActionParams(task.ActionParams)
	cloned.Workflow = bridgeTasks.CloneWorkflowDefinition(task.Workflow)
	cloned.Orchestration = bridgeTasks.CloneOrchestrationDefinition(task.Orchestration)
	return cloned
}
