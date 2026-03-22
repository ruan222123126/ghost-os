package orchestration

import (
	"fmt"

	bridgeTasks "ghost-os/bridge/tasks"
)

func executeWorkflowTask(task ScheduledTask) bridgeTasks.ExecutionResult {
	startNodeID, endNodeID, err := buildWorkflowExecutionPlan(task)
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
	}
	return bridgeTasks.ExecutionResult{
		Status:          taskRunStatusSuccess,
		ResponsePreview: fmt.Sprintf("workflow completed: %s -> %s", startNodeID, endNodeID),
	}
}

func buildWorkflowExecutionPlan(task ScheduledTask) (string, string, error) {
	normalized := task
	if err := validateTaskDefinition(&normalized); err != nil {
		return "", "", err
	}
	edge := normalized.Workflow.Edges[0]
	return edge.FromNodeID, edge.ToNodeID, nil
}
