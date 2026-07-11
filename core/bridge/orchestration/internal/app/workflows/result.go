package workflows

import (
	"fmt"
	"strings"

	workflowdomain "ghost-os/bridge/orchestration/internal/domain/workflow"
	bridgeTasks "ghost-os/bridge/tasks"
)

func pathResultToExecutionResult(result pathResult, plan workflowdomain.Plan) bridgeTasks.ExecutionResult {
	switch result.status {
	case bridgeTasks.RunStatusSuccess:
		return buildSuccessResult(result.lastNode, result.lastPreview, result.sessionID, plan)
	case bridgeTasks.RunStatusAwaitingHuman:
		return bridgeTasks.ExecutionResult{
			Status:          bridgeTasks.RunStatusAwaitingHuman,
			SessionIDOutput: result.sessionID,
			ResponsePreview: fmt.Sprintf("workflow awaiting human at %s: %s", result.awaitingNodeID, result.awaitingPrompt),
		}
	default:
		return bridgeTasks.ExecutionResult{Status: bridgeTasks.RunStatusError, Error: resultErrorText(result)}
	}
}

func buildSuccessResult(
	lastNode workflowdomain.Node,
	lastPreview string,
	sessionID string,
	plan workflowdomain.Plan,
) bridgeTasks.ExecutionResult {
	if strings.TrimSpace(lastNode.ID) == "" {
		return bridgeTasks.ExecutionResult{
			Status:          bridgeTasks.RunStatusSuccess,
			SessionIDOutput: sessionID,
			ResponsePreview: fmt.Sprintf("workflow completed: %s -> %s", plan.StartID(), plan.EndID()),
		}
	}
	return bridgeTasks.ExecutionResult{
		Status:          bridgeTasks.RunStatusSuccess,
		SessionIDOutput: sessionID,
		ResponsePreview: fmt.Sprintf("workflow completed at %s: %s", lastNode.ID, lastPreview),
	}
}

func resultErrorText(result pathResult) string {
	if result.err != nil {
		return result.err.Error()
	}
	return "workflow execution failed"
}
