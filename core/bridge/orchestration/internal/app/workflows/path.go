package workflows

import (
	"context"
	"fmt"
	"strings"

	workflowdomain "ghost-os/bridge/orchestration/internal/domain/workflow"
	sharedresult "ghost-os/bridge/orchestration/internal/shared/result"
	bridgeTasks "ghost-os/bridge/tasks"
)

type pathResult struct {
	status         string
	sessionID      string
	lastNode       workflowdomain.Node
	lastPreview    string
	awaitingNodeID string
	awaitingPrompt string
	err            error
}

func executePath(
	ctx context.Context,
	cmd ExecuteCommand,
	startNodeID string,
	state *runState,
	recorder *sharedresult.NodeResultRecorder,
	branchID string,
) pathResult {
	currentNodeID := strings.TrimSpace(startNodeID)
	result := pathCursor{}
	for {
		if err := ctx.Err(); err != nil {
			return pathResult{status: bridgeTasks.RunStatusError, err: err}
		}
		node, ok := cmd.Plan.Node(currentNodeID)
		if !ok {
			return missingNodePathResult(currentNodeID)
		}
		stepOutcome := executePathNode(ctx, cmd, node, state, recorder, branchID)
		if stepOutcome.done {
			return stepOutcome.result.withCursor(result)
		}
		result.remember(node, stepOutcome.step, state)
		currentNodeID = stepOutcome.step.nextNodeID
	}
}

type pathCursor struct {
	lastSessionID string
	lastNode      workflowdomain.Node
	lastPreview   string
}

func (c *pathCursor) remember(node workflowdomain.Node, step stepResult, state *runState) {
	outcome := step.outcome
	if sessionID := strings.TrimSpace(outcome.SessionID); sessionID != "" {
		c.lastSessionID = sessionID
	}
	if !step.executed || outcome.Status == bridgeTasks.RunStatusAwaitingHuman {
		return
	}
	state.recordNode(node, outcome)
	c.lastNode = node
	c.lastPreview = outcome.Preview
}

func (r pathResult) withCursor(cursor pathCursor) pathResult {
	if r.status != bridgeTasks.RunStatusSuccess {
		return r
	}
	r.sessionID = cursor.lastSessionID
	r.lastNode = cursor.lastNode
	r.lastPreview = cursor.lastPreview
	return r
}

func missingNodePathResult(nodeID string) pathResult {
	return pathResult{
		status: bridgeTasks.RunStatusError,
		err:    fmt.Errorf("workflow node %q is missing", nodeID),
	}
}

type pathNodeResult struct {
	step   stepResult
	result pathResult
	done   bool
}
