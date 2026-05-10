package workflow

import (
	"fmt"
	"strings"

	bridgeTasks "ghost-os/bridge/tasks"
)

func validateIfTargets(nodeID string, node Node, index NodeIndex, graph Graph) error {
	if sourceID := strings.TrimSpace(node.If.SourceNodeID); sourceID != "" {
		if _, ok := index.nodes[sourceID]; !ok {
			return fmt.Errorf("%w: workflow if node %q references unknown source_node_id %q", bridgeTasks.ErrInvalidTaskConfig, nodeID, sourceID)
		}
	}
	if _, ok := index.nodes[node.If.TrueNodeID]; !ok {
		return fmt.Errorf("%w: workflow if node %q references unknown true_node_id %q", bridgeTasks.ErrInvalidTaskConfig, nodeID, node.If.TrueNodeID)
	}
	if _, ok := index.nodes[node.If.FalseNodeID]; !ok {
		return fmt.Errorf("%w: workflow if node %q references unknown false_node_id %q", bridgeTasks.ErrInvalidTaskConfig, nodeID, node.If.FalseNodeID)
	}
	if !outgoingMatchesTargets(graph.outgoing[nodeID], node.If.TrueNodeID, node.If.FalseNodeID) {
		return fmt.Errorf("%w: workflow if node %q outgoing edges must match true_node_id/false_node_id", bridgeTasks.ErrInvalidTaskConfig, nodeID)
	}
	return nil
}

func validateLoopTargets(nodeID string, node Node, index NodeIndex, graph Graph) error {
	if _, ok := index.nodes[node.Loop.BodyNodeID]; !ok {
		return fmt.Errorf("%w: workflow loop node %q references unknown body_node_id %q", bridgeTasks.ErrInvalidTaskConfig, nodeID, node.Loop.BodyNodeID)
	}
	if _, ok := index.nodes[node.Loop.ExitNodeID]; !ok {
		return fmt.Errorf("%w: workflow loop node %q references unknown exit_node_id %q", bridgeTasks.ErrInvalidTaskConfig, nodeID, node.Loop.ExitNodeID)
	}
	if !outgoingMatchesTargets(graph.outgoing[nodeID], node.Loop.BodyNodeID, node.Loop.ExitNodeID) {
		return fmt.Errorf("%w: workflow loop node %q outgoing edges must match body_node_id/exit_node_id", bridgeTasks.ErrInvalidTaskConfig, nodeID)
	}
	return nil
}

func outgoingMatchesTargets(outgoing []string, firstID string, secondID string) bool {
	if len(outgoing) != 2 {
		return false
	}
	firstSeen := false
	secondSeen := false
	for _, nodeID := range outgoing {
		firstSeen = firstSeen || nodeID == firstID
		secondSeen = secondSeen || nodeID == secondID
	}
	return firstSeen && secondSeen
}
