package workflow

import (
	"fmt"
	"strings"

	bridgeTasks "ghost-os/bridge/tasks"
)

func (p Plan) Node(nodeID string) (Node, bool) {
	node, ok := p.nodes[nodeID]
	return node, ok
}

func (p Plan) NodeCount() int {
	return len(p.nodes)
}

func (p Plan) StartID() string {
	return p.startID
}

func (p Plan) EndID() string {
	return p.endID
}

func (p Plan) NextNodeIDs(nodeID string) []string {
	return append([]string(nil), p.graph.outgoing[nodeID]...)
}

func (p Plan) SingleNextNodeID(nodeID string) (string, error) {
	nextIDs := p.NextNodeIDs(nodeID)
	if len(nextIDs) != 1 {
		return "", fmt.Errorf("%w: workflow node %q must have exactly one outgoing edge", bridgeTasks.ErrInvalidTaskConfig, nodeID)
	}
	return nextIDs[0], nil
}

func (p Plan) NeedsRuntimeDependencies() bool {
	for _, node := range p.nodes {
		if node.Type == NodeTypeTool || node.Type == NodeTypeLLM {
			return true
		}
	}
	return false
}

func (p Plan) NeedsService() bool {
	for _, node := range p.nodes {
		if requiresService(node) {
			return true
		}
	}
	return false
}

func requiresService(node Node) bool {
	switch strings.TrimSpace(node.Type) {
	case NodeTypeTool, NodeTypeLLM, NodeTypeAgent:
		return true
	default:
		return false
	}
}
