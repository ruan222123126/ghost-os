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

func (Runner) Execute(ctx context.Context, cmd ExecuteCommand) bridgeTasks.ExecutionResult {
	startNode, state, err := initRunState(cmd.Plan)
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: bridgeTasks.RunStatusError, Error: err.Error()}
	}
	recorder := sharedresult.NewNodeResultRecorder(cmd.Plan.NodeCount())
	branches := cmd.Plan.NextNodeIDs(startNode.ID)
	if len(branches) > 1 {
		return executeParallelStart(ctx, cmd, startNode, branches, recorder)
	}
	result := executePath(ctx, cmd, cmd.Plan.StartID(), &state, recorder, "")
	execution := pathResultToExecutionResult(result, cmd.Plan)
	execution.NodeResults = recorder.Snapshot()
	return execution
}

func initRunState(plan workflowdomain.Plan) (workflowdomain.Node, runState, error) {
	startNode, ok := plan.Node(plan.StartID())
	if !ok {
		return workflowdomain.Node{}, runState{}, fmt.Errorf("workflow start node %q is missing", plan.StartID())
	}
	return startNode, newRunState(plan.NodeCount()), nil
}

func newRunState(nodeCount int) runState {
	return runState{
		nodeOutputs:    make(map[string]string, nodeCount),
		loopIterations: make(map[string]int, nodeCount),
	}
}

func (s *runState) recordNode(node workflowdomain.Node, outcome NodeOutcome) {
	output := strings.TrimSpace(outcome.OutputText)
	if output == "" {
		output = strings.TrimSpace(outcome.Preview)
	}
	s.lastOutputText = output
	s.nodeOutputs[node.ID] = output
	if findIconOutput, ok := workflowdomain.ExtractFindIconOutput(node, ScreenControlToolID, outcome.OutputValue); ok {
		s.findIconOutput = findIconOutput
	}
}

func (s runState) sourceText(sourceNodeID string) string {
	if strings.TrimSpace(sourceNodeID) == "" {
		return s.lastOutputText
	}
	return strings.TrimSpace(s.nodeOutputs[sourceNodeID])
}

func executeParallelStart(
	ctx context.Context,
	cmd ExecuteCommand,
	startNode workflowdomain.Node,
	branches []string,
	recorder *sharedresult.NodeResultRecorder,
) bridgeTasks.ExecutionResult {
	recordParallelStart(startNode, branches, recorder)
	result := executeParallelBranches(ctx, cmd, startNode, branches, recorder)
	result.NodeResults = recorder.Snapshot()
	return result
}

func recordParallelStart(
	startNode workflowdomain.Node,
	branches []string,
	recorder *sharedresult.NodeResultRecorder,
) {
	recorder.Record(sharedresult.NodeRecord{
		NodeID:    startNode.ID,
		NodeType:  startNode.Type,
		Status:    bridgeTasks.RunStatusSuccess,
		StartedAt: time.Now().UTC(),
		Input:     nodeInputSnapshot(startNode),
		Output:    map[string]any{"next_node_ids": append([]string(nil), branches...)},
		Preview:   fmt.Sprintf("parallel branches: %s", strings.Join(branches, ", ")),
	})
}
