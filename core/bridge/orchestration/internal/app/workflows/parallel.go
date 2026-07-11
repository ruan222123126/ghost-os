package workflows

import (
	"context"
	"fmt"
	"strings"

	workflowdomain "ghost-os/bridge/orchestration/internal/domain/workflow"
	sharedresult "ghost-os/bridge/orchestration/internal/shared/result"
	bridgeTasks "ghost-os/bridge/tasks"
)

type parallelBranchResult struct {
	index       int
	startNodeID string
	path        pathResult
}

func executeParallelBranches(
	ctx context.Context,
	cmd ExecuteCommand,
	startNode workflowdomain.Node,
	branchNodeIDs []string,
	recorder *sharedresult.NodeResultRecorder,
) bridgeTasks.ExecutionResult {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	resultCh := make(chan parallelBranchResult, len(branchNodeIDs))
	results := make([]parallelBranchResult, len(branchNodeIDs))
	for index, nodeID := range branchNodeIDs {
		go runParallelBranch(runCtx, cmd, startNode, nodeID, index, resultCh, recorder)
	}
	terminalIndex := collectParallelResults(branchNodeIDs, results, resultCh, cancel)
	if terminalIndex >= 0 {
		return pathResultToExecutionResult(results[terminalIndex].path, cmd.Plan)
	}
	return buildParallelSuccessResult(results, cmd.Plan)
}

func runParallelBranch(
	ctx context.Context,
	cmd ExecuteCommand,
	_ workflowdomain.Node,
	branchNodeID string,
	index int,
	resultCh chan<- parallelBranchResult,
	recorder *sharedresult.NodeResultRecorder,
) {
	state := newRunState(cmd.Plan.NodeCount())
	path := executePath(ctx, cmd, branchNodeID, &state, recorder, strings.TrimSpace(branchNodeID))
	resultCh <- parallelBranchResult{index: index, startNodeID: strings.TrimSpace(branchNodeID), path: path}
}

func collectParallelResults(
	branchNodeIDs []string,
	results []parallelBranchResult,
	resultCh <-chan parallelBranchResult,
	cancel context.CancelFunc,
) int {
	terminalIndex := -1
	for range branchNodeIDs {
		result := <-resultCh
		results[result.index] = result
		if terminalIndex >= 0 || result.path.status == bridgeTasks.RunStatusSuccess {
			continue
		}
		terminalIndex = result.index
		cancel()
	}
	return terminalIndex
}

func buildParallelSuccessResult(results []parallelBranchResult, plan workflowdomain.Plan) bridgeTasks.ExecutionResult {
	parts := make([]string, 0, len(results))
	sessionID := ""
	for _, result := range results {
		parts = append(parts, formatParallelBranchPreview(result, plan))
		if id := strings.TrimSpace(result.path.sessionID); id != "" {
			sessionID = id
		}
	}
	return bridgeTasks.ExecutionResult{
		Status:          bridgeTasks.RunStatusSuccess,
		SessionIDOutput: sessionID,
		ResponsePreview: fmt.Sprintf("workflow completed in parallel: %s", strings.Join(parts, " | ")),
	}
}

func formatParallelBranchPreview(result parallelBranchResult, plan workflowdomain.Plan) string {
	lastNodeID := strings.TrimSpace(result.path.lastNode.ID)
	if lastNodeID == "" {
		return fmt.Sprintf("%s -> %s", result.startNodeID, plan.EndID())
	}
	if lastPreview := strings.TrimSpace(result.path.lastPreview); lastPreview != "" {
		return fmt.Sprintf("%s: %s", lastNodeID, lastPreview)
	}
	return lastNodeID
}
