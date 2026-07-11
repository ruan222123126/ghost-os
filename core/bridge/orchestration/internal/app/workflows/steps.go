package workflows

import (
	"context"
	"fmt"
	"strings"

	workflowdomain "ghost-os/bridge/orchestration/internal/domain/workflow"
	bridgeTasks "ghost-os/bridge/tasks"
)

type ifDecision struct {
	nextNodeID  string
	operator    string
	targetValue string
	matched     bool
}

type loopDecision struct {
	nextNodeID    string
	iteration     int
	maxIterations int
	enteringLoop  bool
}

func executeStep(
	ctx context.Context,
	cmd ExecuteCommand,
	node workflowdomain.Node,
	state *runState,
	branchID string,
) stepResult {
	switch node.Type {
	case workflowdomain.NodeTypeStart:
		nextID, err := cmd.Plan.SingleNextNodeID(node.ID)
		return stepResult{nextNodeID: nextID, err: err}
	case workflowdomain.NodeTypeTool, workflowdomain.NodeTypeLLM, workflowdomain.NodeTypeAgent:
		return executeActionStep(ctx, cmd, node, state, branchID)
	case workflowdomain.NodeTypeIf:
		decision, err := selectIfDecision(node, state)
		return stepResult{nextNodeID: decision.nextNodeID, outcome: ifOutcome(node, decision), err: err}
	case workflowdomain.NodeTypeLoop:
		decision, err := selectLoopDecision(node, state)
		return stepResult{nextNodeID: decision.nextNodeID, outcome: loopOutcome(node, decision), err: err}
	default:
		return stepResult{err: fmt.Errorf("unsupported workflow node type %q", node.Type)}
	}
}

func executeActionStep(
	ctx context.Context,
	cmd ExecuteCommand,
	node workflowdomain.Node,
	state *runState,
	branchID string,
) stepResult {
	outcome := executeActionNode(ctx, cmd, node, state, branchID)
	if outcome.Err != nil || outcome.Status == bridgeTasks.RunStatusAwaitingHuman {
		return stepResult{outcome: outcome, executed: true, err: outcome.Err}
	}
	nextID, err := cmd.Plan.SingleNextNodeID(node.ID)
	return stepResult{nextNodeID: nextID, outcome: outcome, executed: true, err: err}
}

func executeActionNode(
	ctx context.Context,
	cmd ExecuteCommand,
	node workflowdomain.Node,
	state *runState,
	branchID string,
) NodeOutcome {
	resolvedNode, err := workflowdomain.ResolveActionNode(node, state.findIconOutput)
	if err != nil {
		return NodeOutcome{Err: err, InputSnapshot: nodeInputSnapshot(node)}
	}
	inputSnapshot := nodeInputSnapshot(resolvedNode)
	outcome := executeResolvedActionNode(ctx, cmd, node, resolvedNode, branchID, state.iteration)
	outcome.InputSnapshot = inputSnapshot
	return outcome
}

func executeResolvedActionNode(
	ctx context.Context,
	cmd ExecuteCommand,
	node workflowdomain.Node,
	resolvedNode workflowdomain.Node,
	branchID string,
	iteration int,
) NodeOutcome {
	switch node.Type {
	case workflowdomain.NodeTypeTool:
		return ExecuteToolNode(ctx, cmd.Runtime, resolvedNode, cmd.TraceID, cmd.TemplateUploader)
	case workflowdomain.NodeTypeLLM:
		return executeLLMNode(ctx, cmd, resolvedNode, branchID, iteration)
	case workflowdomain.NodeTypeAgent:
		return executeAgentNode(ctx, cmd.Agent, resolvedNode, cmd.TraceID, branchID, iteration)
	default:
		return NodeOutcome{Err: fmt.Errorf("unsupported workflow node type %q", node.Type)}
	}
}

func selectIfDecision(node workflowdomain.Node, state *runState) (ifDecision, error) {
	if node.If == nil {
		return ifDecision{}, fmt.Errorf("workflow if node %q payload is missing", node.ID)
	}
	sourceText := state.sourceText(node.If.SourceNodeID)
	targetValue, err := workflowdomain.ResolveIfValue(node.If.Value, state.findIconOutput)
	if err != nil {
		return ifDecision{}, fmt.Errorf("workflow if node %q resolve value: %w", node.ID, err)
	}
	matched, err := workflowdomain.EvaluateIfCondition(node.If.Operator, sourceText, targetValue)
	if err != nil {
		return ifDecision{}, fmt.Errorf("workflow if node %q: %w", node.ID, err)
	}
	return buildIfDecision(node, targetValue, matched), nil
}

func buildIfDecision(node workflowdomain.Node, targetValue string, matched bool) ifDecision {
	decision := ifDecision{operator: strings.TrimSpace(node.If.Operator), targetValue: strings.TrimSpace(targetValue), matched: matched}
	if matched {
		decision.nextNodeID = strings.TrimSpace(node.If.TrueNodeID)
		return decision
	}
	decision.nextNodeID = strings.TrimSpace(node.If.FalseNodeID)
	return decision
}
