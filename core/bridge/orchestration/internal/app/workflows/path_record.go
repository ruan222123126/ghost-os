package workflows

import (
	"fmt"
	"strings"

	workflowdomain "ghost-os/bridge/orchestration/internal/domain/workflow"
	sharedresult "ghost-os/bridge/orchestration/internal/shared/result"
	bridgeTasks "ghost-os/bridge/tasks"
)

func stepStatus(step stepResult) string {
	if step.err != nil {
		return bridgeTasks.RunStatusError
	}
	status := strings.TrimSpace(step.outcome.Status)
	if status == "" {
		return bridgeTasks.RunStatusSuccess
	}
	return status
}

func stepInput(node workflowdomain.Node, step stepResult) any {
	if step.executed && step.outcome.InputSnapshot != nil {
		return sharedresult.JSONSnapshot(step.outcome.InputSnapshot)
	}
	return nodeInputSnapshot(node)
}

func stepPreview(step stepResult) string {
	preview := strings.TrimSpace(step.outcome.Preview)
	if preview != "" || strings.TrimSpace(step.nextNodeID) == "" {
		return preview
	}
	return "next: " + strings.TrimSpace(step.nextNodeID)
}

func stepError(node workflowdomain.Node, step stepResult) string {
	if step.err == nil {
		return ""
	}
	return fmt.Sprintf("workflow node %s (%s) failed: %v", node.ID, node.Type, step.err)
}

func nodeInputSnapshot(node workflowdomain.Node) any {
	return sharedresult.JSONSnapshot(node)
}

func nodeOutputSnapshot(nextNodeID string, outcome NodeOutcome) any {
	output := map[string]any{}
	if strings.TrimSpace(nextNodeID) != "" {
		output["next_node_id"] = strings.TrimSpace(nextNodeID)
	}
	addOutcomeFields(output, outcome)
	if len(output) == 0 {
		return nil
	}
	return output
}

func addOutcomeFields(output map[string]any, outcome NodeOutcome) {
	if strings.TrimSpace(outcome.SessionID) != "" {
		output["session_id_output"] = strings.TrimSpace(outcome.SessionID)
	}
	if strings.TrimSpace(outcome.Preview) != "" {
		output["response_preview"] = strings.TrimSpace(outcome.Preview)
	}
	if outcome.OutputValue != nil {
		output["output"] = sharedresult.JSONSnapshot(outcome.OutputValue)
	} else if strings.TrimSpace(outcome.OutputText) != "" {
		output["output"] = strings.TrimSpace(outcome.OutputText)
	}
	if outcome.Err != nil {
		output["error"] = outcome.Err.Error()
	}
}
