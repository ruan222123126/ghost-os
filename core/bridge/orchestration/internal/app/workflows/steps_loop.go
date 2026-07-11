package workflows

import (
	"fmt"
	"strings"

	workflowdomain "ghost-os/bridge/orchestration/internal/domain/workflow"
	bridgeTasks "ghost-os/bridge/tasks"
)

func selectLoopDecision(node workflowdomain.Node, state *runState) (loopDecision, error) {
	if node.Loop == nil {
		return loopDecision{}, fmt.Errorf("workflow loop node %q payload is missing", node.ID)
	}
	count := state.loopIterations[node.ID]
	decision := loopDecision{maxIterations: node.Loop.MaxIterations}
	if count < node.Loop.MaxIterations {
		state.loopIterations[node.ID] = count + 1
		state.iteration = count + 1
		decision.nextNodeID = strings.TrimSpace(node.Loop.BodyNodeID)
		decision.iteration = count + 1
		decision.enteringLoop = true
		return decision, nil
	}
	state.iteration = 0
	decision.nextNodeID = strings.TrimSpace(node.Loop.ExitNodeID)
	return decision, nil
}

func ifOutcome(node workflowdomain.Node, decision ifDecision) NodeOutcome {
	branch := "false"
	if decision.matched {
		branch = "true"
	}
	return NodeOutcome{
		Status:  bridgeTasks.RunStatusSuccess,
		Preview: fmt.Sprintf("if %s branch=%s next=%s", node.ID, branch, decision.nextNodeID),
		OutputValue: map[string]any{
			"branch":       branch,
			"matched":      decision.matched,
			"operator":     decision.operator,
			"value":        decision.targetValue,
			"next_node_id": decision.nextNodeID,
		},
	}
}

func loopOutcome(node workflowdomain.Node, decision loopDecision) NodeOutcome {
	if !decision.enteringLoop {
		return loopExitOutcome(node, decision)
	}
	return NodeOutcome{
		Status:  bridgeTasks.RunStatusSuccess,
		Preview: fmt.Sprintf("loop %s iteration=%d/%d next=%s", node.ID, decision.iteration, decision.maxIterations, decision.nextNodeID),
		OutputValue: map[string]any{
			"entering_loop":  true,
			"iteration":      decision.iteration,
			"max_iterations": decision.maxIterations,
			"next_node_id":   decision.nextNodeID,
		},
	}
}

func loopExitOutcome(node workflowdomain.Node, decision loopDecision) NodeOutcome {
	return NodeOutcome{
		Status:  bridgeTasks.RunStatusSuccess,
		Preview: fmt.Sprintf("loop %s exit next=%s", node.ID, decision.nextNodeID),
		OutputValue: map[string]any{
			"entering_loop":  false,
			"max_iterations": decision.maxIterations,
			"next_node_id":   decision.nextNodeID,
		},
	}
}
