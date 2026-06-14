package tasks

import (
	"fmt"
	"strings"

	appworkflows "ghost-os/bridge/orchestration/internal/app/workflows"
	groupdomain "ghost-os/bridge/orchestration/internal/domain/group"
	bridgeTasks "ghost-os/bridge/tasks"
)

func ValidateDefinition(task *bridgeTasks.ScheduledTask) error {
	if task == nil {
		return fmt.Errorf("%w: task is nil", bridgeTasks.ErrInvalidTaskConfig)
	}
	NormalizeDefinition(task)

	switch task.TaskKind {
	case bridgeTasks.KindAgentMessage:
		return validateAgentDefinition(task)
	case bridgeTasks.KindWorkflow:
		return appworkflows.ValidateTaskDefinition(task)
	case bridgeTasks.KindOrchestration:
		return validateOrchestrationDefinition(task)
	default:
		return fmt.Errorf("%w: unsupported task_kind %q", bridgeTasks.ErrInvalidTaskConfig, task.TaskKind)
	}
}

func NormalizeDefinition(task *bridgeTasks.ScheduledTask) {
	task.Name = strings.TrimSpace(task.Name)
	task.Message = strings.TrimSpace(task.Message)
	task.SessionID = strings.TrimSpace(task.SessionID)
	task.RuntimeOverrides = bridgeTasks.CloneTaskRuntimeOverrides(task.RuntimeOverrides)
	task.AgentMode = bridgeTasks.NormalizeAgentMode(task.AgentMode)
	task.Relay = bridgeTasks.CloneTaskRelayConfig(task.Relay)
	task.TaskKind = bridgeTasks.NormalizeKind(task.TaskKind)
	task.Workflow = bridgeTasks.CloneWorkflowDefinition(task.Workflow)
	task.Orchestration = bridgeTasks.CloneOrchestrationDefinition(task.Orchestration)
}

func validateAgentDefinition(task *bridgeTasks.ScheduledTask) error {
	if strings.TrimSpace(task.Message) == "" {
		return fmt.Errorf("%w: message is required", bridgeTasks.ErrInvalidTaskConfig)
	}
	task.Name = ""
	task.Orchestration = nil
	if err := validateAgentMode(task); err != nil {
		return err
	}
	if task.Workflow != nil {
		return fmt.Errorf("%w: agent_message does not allow workflow", bridgeTasks.ErrInvalidTaskConfig)
	}
	return nil
}

func validateAgentMode(task *bridgeTasks.ScheduledTask) error {
	switch task.AgentMode {
	case "", bridgeTasks.AgentModeSingle:
		task.AgentMode = bridgeTasks.AgentModeSingle
		task.Relay = nil
		return nil
	case bridgeTasks.AgentModeRelay:
		return ValidateRelayConfig(task.Relay)
	default:
		return fmt.Errorf("%w: unsupported agent_mode %q", bridgeTasks.ErrInvalidTaskConfig, task.AgentMode)
	}
}

func ValidateRelayConfig(relay *bridgeTasks.TaskRelayConfig) error {
	if relay == nil {
		return nil
	}
	policy := strings.TrimSpace(relay.StopPolicy)
	switch policy {
	case bridgeTasks.RelayStopPolicyAIDecides, bridgeTasks.RelayStopPolicyMaxRounds:
		if relay.MaxRounds <= 0 {
			return fmt.Errorf("%w: relay max_rounds must be > 0", bridgeTasks.ErrInvalidTaskConfig)
		}
	default:
		return fmt.Errorf("%w: unsupported relay stop_policy %q", bridgeTasks.ErrInvalidTaskConfig, relay.StopPolicy)
	}
	if relay.ExecutionTimeoutMS != nil && *relay.ExecutionTimeoutMS < 0 {
		return fmt.Errorf("%w: relay execution_timeout_ms must be >= 0", bridgeTasks.ErrInvalidTaskConfig)
	}
	return nil
}

func validateOrchestrationDefinition(task *bridgeTasks.ScheduledTask) error {
	if task.Orchestration == nil {
		return fmt.Errorf("%w: orchestration is required for orchestration task", bridgeTasks.ErrInvalidTaskConfig)
	}
	if task.Name == "" {
		return fmt.Errorf("%w: orchestration name is required", bridgeTasks.ErrInvalidTaskConfig)
	}
	if task.Message != "" || task.SessionID != "" {
		return fmt.Errorf("%w: orchestration task does not allow message or session_id", bridgeTasks.ErrInvalidTaskConfig)
	}
	if task.RuntimeOverrides != nil || task.Workflow != nil {
		return fmt.Errorf("%w: orchestration task only allows name, orchestration, and schedule fields", bridgeTasks.ErrInvalidTaskConfig)
	}
	if err := normalizeOrchestrationDefinitionRuntimeOverrides(task.Orchestration); err != nil {
		return err
	}
	_, err := groupdomain.PlanBuilder{}.Build(task.Orchestration)
	return err
}

func normalizeOrchestrationDefinitionRuntimeOverrides(definition *bridgeTasks.OrchestrationDefinition) error {
	if definition == nil {
		return nil
	}
	for index := range definition.Nodes {
		node := &definition.Nodes[index]
		if node.Type != bridgeTasks.OrchestrationNodeTypeAgent || node.Agent == nil {
			continue
		}
		overrides, err := NormalizeOrchestrationAgentRuntimeOverrides(node.Agent.RuntimeOverrides)
		if err != nil {
			return fmt.Errorf("%w: orchestration agent node %q %v", bridgeTasks.ErrInvalidTaskConfig, node.ID, err)
		}
		node.Agent.RuntimeOverrides = overrides
	}
	return nil
}

func EnsureWorkflowAllowedForKind(taskKind string, workflow *bridgeTasks.WorkflowDefinition) error {
	normalized := bridgeTasks.NormalizeKind(taskKind)
	if workflow == nil || normalized == bridgeTasks.KindWorkflow {
		return nil
	}
	if !bridgeTasks.IsSupportedKind(normalized) {
		return InvalidConfig(fmt.Sprintf("unsupported task_kind %q", normalized))
	}
	return InvalidConfig(normalized + " does not allow workflow")
}

func EnsureOrchestrationAllowedForKind(taskKind string, definition *bridgeTasks.OrchestrationDefinition) error {
	normalized := bridgeTasks.NormalizeKind(taskKind)
	if definition == nil || normalized == bridgeTasks.KindOrchestration {
		return nil
	}
	if !bridgeTasks.IsSupportedKind(normalized) {
		return InvalidConfig(fmt.Sprintf("unsupported task_kind %q", normalized))
	}
	return InvalidConfig(normalized + " does not allow orchestration")
}
