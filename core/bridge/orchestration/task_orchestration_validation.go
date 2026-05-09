package orchestration

import (
	"fmt"

	groupdomain "ghost-os/bridge/orchestration/internal/domain/group"
)

type orchestrationExecutionPlan struct {
	nodes        map[string]OrchestrationNode
	entryGroupID string
	controlNext  map[string]string
	groupMember  map[string][]string
}

func validateOrchestrationTaskDefinition(task *ScheduledTask) error {
	if task.Orchestration == nil {
		return fmt.Errorf("%w: orchestration is required for orchestration task", ErrInvalidTaskConfig)
	}
	if task.Name == "" {
		return fmt.Errorf("%w: orchestration name is required", ErrInvalidTaskConfig)
	}
	if task.Message != "" || task.SessionID != "" || task.Action != "" {
		return fmt.Errorf("%w: orchestration task does not allow message, session_id, or action", ErrInvalidTaskConfig)
	}
	if task.RuntimeOverrides != nil || task.Workflow != nil || len(task.ActionParams) > 0 {
		return fmt.Errorf("%w: orchestration task only allows name, orchestration, and schedule fields", ErrInvalidTaskConfig)
	}
	if err := normalizeOrchestrationDefinitionRuntimeOverrides(task.Orchestration); err != nil {
		return err
	}
	_, err := buildOrchestrationExecutionPlan(task.Orchestration)
	return err
}

func normalizeOrchestrationDefinitionRuntimeOverrides(definition *OrchestrationDefinition) error {
	if definition == nil {
		return nil
	}
	for index := range definition.Nodes {
		node := &definition.Nodes[index]
		if node.Type != orchestrationNodeTypeAgent || node.Agent == nil {
			continue
		}
		overrides, err := normalizeOrchestrationAgentRuntimeOverrides(node.Agent.RuntimeOverrides)
		if err != nil {
			return fmt.Errorf("%w: orchestration agent node %q %v", ErrInvalidTaskConfig, node.ID, err)
		}
		node.Agent.RuntimeOverrides = overrides
	}
	return nil
}

func buildOrchestrationExecutionPlan(definition *OrchestrationDefinition) (orchestrationExecutionPlan, error) {
	plan, err := groupdomain.PlanBuilder{}.Build(definition)
	if err != nil {
		return orchestrationExecutionPlan{}, err
	}
	return orchestrationExecutionPlan{
		nodes:        plan.Nodes,
		entryGroupID: plan.EntryGroupID,
		controlNext:  plan.ControlNext,
		groupMember:  plan.GroupMembers,
	}, nil
}
