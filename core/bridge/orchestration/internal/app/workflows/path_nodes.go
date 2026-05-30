package workflows

import (
	"context"
	"fmt"
	"strings"
	"time"

	workflowdomain "ghost-os/bridge/orchestration/internal/domain/workflow"
	sharedresult "ghost-os/bridge/orchestration/internal/shared/result"
	bridgeTasks "ghost-os/bridge/tasks"
)

func executePathNode(
	ctx context.Context,
	cmd ExecuteCommand,
	node workflowdomain.Node,
	state *runState,
	recorder *sharedresult.NodeResultRecorder,
	branchID string,
) pathNodeResult {
	startedAt := time.Now().UTC()
	if node.Type == workflowdomain.NodeTypeEnd {
		recordEndNode(node, branchID, startedAt, recorder)
		return pathNodeResult{done: true, result: pathResult{status: bridgeTasks.RunStatusSuccess}}
	}
	step := executeStep(ctx, cmd, node, state, branchID)
	recordStep(node, step, branchID, startedAt, recorder)
	return stepToPathNodeResult(node, step)
}

func recordEndNode(
	node workflowdomain.Node,
	branchID string,
	startedAt time.Time,
	recorder *sharedresult.NodeResultRecorder,
) {
	recorder.Record(sharedresult.NodeRecord{
		NodeID:    node.ID,
		NodeType:  node.Type,
		Status:    bridgeTasks.RunStatusSuccess,
		StartedAt: startedAt,
		BranchID:  branchID,
		Input:     nodeInputSnapshot(node),
		Output:    map[string]any{"reached_end": true},
		Preview:   "end node reached",
	})
}

func recordStep(
	node workflowdomain.Node,
	step stepResult,
	branchID string,
	startedAt time.Time,
	recorder *sharedresult.NodeResultRecorder,
) {
	recorder.Record(sharedresult.NodeRecord{
		NodeID:    node.ID,
		NodeType:  node.Type,
		Status:    stepStatus(step),
		StartedAt: startedAt,
		BranchID:  branchID,
		Input:     stepInput(node, step),
		Output:    nodeOutputSnapshot(step.nextNodeID, step.outcome),
		Preview:   stepPreview(step),
		Error:     stepError(node, step),
	})
}

func stepToPathNodeResult(node workflowdomain.Node, step stepResult) pathNodeResult {
	if step.err != nil {
		return pathNodeResult{done: true, result: pathResult{status: bridgeTasks.RunStatusError, err: fmt.Errorf("%s", stepError(node, step))}}
	}
	if step.outcome.Status == bridgeTasks.RunStatusAwaitingHuman {
		return awaitingPathNodeResult(node, step)
	}
	return pathNodeResult{step: step}
}

func awaitingPathNodeResult(node workflowdomain.Node, step stepResult) pathNodeResult {
	return pathNodeResult{
		step: step,
		done: true,
		result: pathResult{
			status:         bridgeTasks.RunStatusAwaitingHuman,
			sessionID:      strings.TrimSpace(step.outcome.SessionID),
			awaitingNodeID: node.ID,
			awaitingPrompt: step.outcome.Preview,
		},
	}
}
