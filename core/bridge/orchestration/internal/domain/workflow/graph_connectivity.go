package workflow

import (
	"fmt"

	bridgeTasks "ghost-os/bridge/tasks"
)

func validateConnectivity(index NodeIndex, graph Graph) error {
	fromStart := traverse(index.startID, graph.outgoing)
	if len(fromStart) != len(index.nodes) {
		return fmt.Errorf("%w: workflow must be fully connected from start node", bridgeTasks.ErrInvalidTaskConfig)
	}
	toEnd := traverse(index.endID, graph.incoming)
	for nodeID := range index.nodes {
		if !toEnd[nodeID] {
			return fmt.Errorf("%w: workflow node %q cannot reach end node", bridgeTasks.ErrInvalidTaskConfig, nodeID)
		}
	}
	return validateLoopCycles(index, graph)
}

func validateLoopCycles(index NodeIndex, graph Graph) error {
	for nodeID, node := range index.nodes {
		if node.Type != NodeTypeLoop {
			continue
		}
		if pathExists(graph.outgoing, node.Loop.BodyNodeID, nodeID) {
			continue
		}
		return fmt.Errorf("%w: workflow loop node %q body path must return to the loop node", bridgeTasks.ErrInvalidTaskConfig, nodeID)
	}
	return nil
}

func traverse(startID string, adjacency map[string][]string) map[string]bool {
	seen := make(map[string]bool, len(adjacency))
	stack := []string{startID}
	for len(stack) > 0 {
		last := len(stack) - 1
		nodeID := stack[last]
		stack = stack[:last]
		if seen[nodeID] {
			continue
		}
		seen[nodeID] = true
		stack = pushUnseen(stack, adjacency[nodeID], seen)
	}
	return seen
}

func pathExists(adjacency map[string][]string, startID string, targetID string) bool {
	seen := make(map[string]bool, len(adjacency))
	stack := []string{startID}
	for len(stack) > 0 {
		last := len(stack) - 1
		nodeID := stack[last]
		stack = stack[:last]
		if nodeID == targetID {
			return true
		}
		if seen[nodeID] {
			continue
		}
		seen[nodeID] = true
		stack = pushUnseen(stack, adjacency[nodeID], seen)
	}
	return false
}

func pushUnseen(stack []string, nodeIDs []string, seen map[string]bool) []string {
	for _, nextID := range nodeIDs {
		if !seen[nextID] {
			stack = append(stack, nextID)
		}
	}
	return stack
}
