package orchestration

import (
	"context"
	"fmt"
	"strings"
	"time"

	bridgeTasks "ghost-os/bridge/tasks"
)

type workflowPathResult struct {
	status         string
	sessionID      string
	lastNode       WorkflowNode
	lastPreview    string
	awaitingNodeID string
	awaitingPrompt string
	err            error
}

type workflowParallelBranchResult struct {
	index       int
	startNodeID string
	path        workflowPathResult
}

func (r workflowTaskRunner) initRunState() (WorkflowNode, workflowRunState, error) {
	startNode, ok := r.plan.node(r.plan.startID)
	if !ok {
		return WorkflowNode{}, workflowRunState{}, fmt.Errorf("workflow start node %q is missing", r.plan.startID)
	}
	state := newWorkflowRunState(len(r.plan.nodes))
	if err := state.variables.seedInputs(startNode.Start); err != nil {
		return WorkflowNode{}, workflowRunState{}, err
	}
	return startNode, state, nil
}

func (r workflowTaskRunner) newBranchState(startNode WorkflowNode) (workflowRunState, error) {
	state := newWorkflowRunState(len(r.plan.nodes))
	if err := state.variables.seedInputs(startNode.Start); err != nil {
		return workflowRunState{}, err
	}
	return state, nil
}

func (r workflowTaskRunner) executePath(
	ctx context.Context,
	deps agentRuntimeDependencies,
	startNodeID string,
	state *workflowRunState,
	recorder *workflowNodeResultRecorder,
	branchID string,
) workflowPathResult {
	currentNodeID := strings.TrimSpace(startNodeID)
	lastSessionID := ""
	lastNode := WorkflowNode{}
	lastPreview := ""
	for {
		if err := ctx.Err(); err != nil {
			return workflowPathResult{status: taskRunStatusError, err: err}
		}
		node, ok := r.plan.node(currentNodeID)
		if !ok {
			return workflowPathResult{status: taskRunStatusError, err: fmt.Errorf("workflow node %q is missing", currentNodeID)}
		}
		startedAt := time.Now().UTC()
		nodeInput := workflowNodeInputSnapshot(node)
		if node.Type == workflowNodeTypeEnd {
			recorder.record(workflowNodeRecord{
				NodeID:    node.ID,
				NodeType:  node.Type,
				Status:    taskRunStatusSuccess,
				StartedAt: startedAt,
				BranchID:  branchID,
				Input:     nodeInput,
				Output:    workflowEndNodeOutputSnapshot(),
				Preview:   "end node reached",
			})
			return workflowPathResult{
				status:      taskRunStatusSuccess,
				sessionID:   lastSessionID,
				lastNode:    lastNode,
				lastPreview: lastPreview,
			}
		}
		step := r.executeStep(ctx, deps, node, state)
		if step.executed {
			nodeInput = workflowNodeInputFromOutcome(node, step.outcome)
		}
		if step.err != nil {
			nodeErr := fmt.Sprintf("workflow node %s (%s) failed: %v", node.ID, node.Type, step.err)
			recorder.record(workflowNodeRecord{
				NodeID:    node.ID,
				NodeType:  node.Type,
				Status:    taskRunStatusError,
				StartedAt: startedAt,
				BranchID:  branchID,
				Input:     nodeInput,
				Output:    workflowNodeOutputSnapshot(step.nextNodeID, step.outcome),
				Preview:   strings.TrimSpace(step.outcome.preview),
				Error:     nodeErr,
			})
			return workflowPathResult{
				status: taskRunStatusError,
				err:    fmt.Errorf("%s", nodeErr),
			}
		}
		if sessionID := strings.TrimSpace(step.outcome.sessionID); sessionID != "" {
			lastSessionID = sessionID
		}
		stepStatus := strings.TrimSpace(step.outcome.status)
		if stepStatus == "" {
			stepStatus = taskRunStatusSuccess
		}
		stepPreview := strings.TrimSpace(step.outcome.preview)
		if stepPreview == "" && strings.TrimSpace(step.nextNodeID) != "" {
			stepPreview = "next: " + strings.TrimSpace(step.nextNodeID)
		}
		recorder.record(workflowNodeRecord{
			NodeID:    node.ID,
			NodeType:  node.Type,
			Status:    stepStatus,
			StartedAt: startedAt,
			BranchID:  branchID,
			Input:     nodeInput,
			Output:    workflowNodeOutputSnapshot(step.nextNodeID, step.outcome),
			Preview:   stepPreview,
		})
		if step.outcome.status == taskRunStatusAwaitingHuman {
			return workflowPathResult{
				status:         taskRunStatusAwaitingHuman,
				sessionID:      lastSessionID,
				awaitingNodeID: node.ID,
				awaitingPrompt: step.outcome.preview,
			}
		}
		if step.executed {
			state.recordNode(node.ID, step.outcome)
			lastNode = node
			lastPreview = step.outcome.preview
		}
		currentNodeID = step.nextNodeID
	}
}

func (r workflowTaskRunner) executeParallelBranches(
	ctx context.Context,
	deps agentRuntimeDependencies,
	startNode WorkflowNode,
	branchNodeIDs []string,
	recorder *workflowNodeResultRecorder,
) bridgeTasks.ExecutionResult {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	resultCh := make(chan workflowParallelBranchResult, len(branchNodeIDs))
	results := make([]workflowParallelBranchResult, len(branchNodeIDs))
	for index, nodeID := range branchNodeIDs {
		go r.runParallelBranch(runCtx, deps, startNode, nodeID, index, resultCh, recorder)
	}
	terminalIndex := -1
	for range branchNodeIDs {
		result := <-resultCh
		results[result.index] = result
		if terminalIndex >= 0 || result.path.status == taskRunStatusSuccess {
			continue
		}
		terminalIndex = result.index
		cancel()
	}
	if terminalIndex >= 0 {
		return r.pathResultToExecutionResult(results[terminalIndex].path)
	}
	return r.buildParallelSuccessResult(results)
}

func (r workflowTaskRunner) runParallelBranch(
	ctx context.Context,
	deps agentRuntimeDependencies,
	startNode WorkflowNode,
	branchNodeID string,
	index int,
	resultCh chan<- workflowParallelBranchResult,
	recorder *workflowNodeResultRecorder,
) {
	state, err := r.newBranchState(startNode)
	path := workflowPathResult{}
	if err != nil {
		path = workflowPathResult{status: taskRunStatusError, err: err}
	} else {
		path = r.executePath(ctx, deps, branchNodeID, &state, recorder, strings.TrimSpace(branchNodeID))
	}
	resultCh <- workflowParallelBranchResult{
		index:       index,
		startNodeID: strings.TrimSpace(branchNodeID),
		path:        path,
	}
}

func (r workflowTaskRunner) pathResultToExecutionResult(result workflowPathResult) bridgeTasks.ExecutionResult {
	switch result.status {
	case taskRunStatusSuccess:
		return buildWorkflowSuccessResult(result.lastNode, result.lastPreview, result.sessionID, r.plan)
	case taskRunStatusAwaitingHuman:
		return bridgeTasks.ExecutionResult{
			Status:          taskRunStatusAwaitingHuman,
			SessionIDOutput: result.sessionID,
			ResponsePreview: fmt.Sprintf("workflow awaiting human at %s: %s", result.awaitingNodeID, result.awaitingPrompt),
		}
	default:
		errText := "workflow execution failed"
		if result.err != nil {
			errText = result.err.Error()
		}
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: errText}
	}
}

func (r workflowTaskRunner) buildParallelSuccessResult(results []workflowParallelBranchResult) bridgeTasks.ExecutionResult {
	parts := make([]string, 0, len(results))
	sessionID := ""
	for _, result := range results {
		parts = append(parts, r.formatParallelBranchPreview(result))
		if id := strings.TrimSpace(result.path.sessionID); id != "" {
			sessionID = id
		}
	}
	return bridgeTasks.ExecutionResult{
		Status:          taskRunStatusSuccess,
		SessionIDOutput: sessionID,
		ResponsePreview: fmt.Sprintf("workflow completed in parallel: %s", strings.Join(parts, " | ")),
	}
}

func (r workflowTaskRunner) formatParallelBranchPreview(result workflowParallelBranchResult) string {
	lastNodeID := strings.TrimSpace(result.path.lastNode.ID)
	if lastNodeID == "" {
		return fmt.Sprintf("%s -> %s", result.startNodeID, r.plan.endID)
	}
	lastPreview := strings.TrimSpace(result.path.lastPreview)
	if lastPreview == "" {
		return lastNodeID
	}
	return fmt.Sprintf("%s: %s", lastNodeID, lastPreview)
}
