package group

import (
	"fmt"

	bridgeTasks "ghost-os/bridge/tasks"
)

func addEdge(graph graphData, edge Edge) error {
	fromNode, toNode, err := edgeNodes(graph, edge)
	if err != nil {
		return err
	}
	if edge.FromNodeID == edge.ToNodeID {
		return fmt.Errorf("%w: orchestration does not allow self-loop edge %q", bridgeTasks.ErrInvalidTaskConfig, edge.FromNodeID)
	}
	graph.edgeCount++
	return addTypedEdge(graph, edge, fromNode, toNode)
}

func edgeNodes(graph graphData, edge Edge) (Node, Node, error) {
	fromNode, ok := graph.nodes[edge.FromNodeID]
	if !ok {
		return Node{}, Node{}, fmt.Errorf("%w: orchestration edge references unknown from_node_id %q", bridgeTasks.ErrInvalidTaskConfig, edge.FromNodeID)
	}
	toNode, ok := graph.nodes[edge.ToNodeID]
	if !ok {
		return Node{}, Node{}, fmt.Errorf("%w: orchestration edge references unknown to_node_id %q", bridgeTasks.ErrInvalidTaskConfig, edge.ToNodeID)
	}
	return fromNode, toNode, nil
}

func addTypedEdge(graph graphData, edge Edge, fromNode Node, toNode Node) error {
	switch edge.Kind {
	case EdgeKindControl:
		return addControlEdge(graph, edge, fromNode, toNode)
	case EdgeKindMember:
		return addMemberEdge(graph, edge, fromNode, toNode)
	default:
		return fmt.Errorf("%w: unsupported orchestration edge kind %q", bridgeTasks.ErrInvalidTaskConfig, edge.Kind)
	}
}

func addControlEdge(graph graphData, edge Edge, fromNode Node, toNode Node) error {
	if !isValidControlEdge(fromNode.Type, toNode.Type) {
		return fmt.Errorf("%w: orchestration control edge %q -> %q is invalid", bridgeTasks.ErrInvalidTaskConfig, edge.FromNodeID, edge.ToNodeID)
	}
	graph.rawControlOut[edge.FromNodeID]++
	graph.rawControlIn[edge.ToNodeID]++
	if fromNode.Type != NodeTypeGroup || toNode.Type != NodeTypeGroup {
		return nil
	}
	return addGroupControlEdge(graph, edge)
}

func addGroupControlEdge(graph graphData, edge Edge) error {
	if _, exists := graph.controlNext[edge.FromNodeID]; exists {
		return fmt.Errorf("%w: orchestration node %q allows only one control outgoing edge", bridgeTasks.ErrInvalidTaskConfig, edge.FromNodeID)
	}
	graph.controlNext[edge.FromNodeID] = edge.ToNodeID
	graph.controlOut[edge.FromNodeID]++
	graph.controlIn[edge.ToNodeID]++
	return nil
}

func addMemberEdge(graph graphData, edge Edge, fromNode Node, toNode Node) error {
	if fromNode.Type != NodeTypeAgent || toNode.Type != NodeTypeGroup {
		return fmt.Errorf("%w: orchestration member edge must be agent -> group", bridgeTasks.ErrInvalidTaskConfig)
	}
	graph.groupMembers[edge.ToNodeID] = append(graph.groupMembers[edge.ToNodeID], edge.FromNodeID)
	return nil
}

func isValidControlEdge(fromType string, toType string) bool {
	switch fromType {
	case NodeTypeStart:
		return toType == NodeTypeGroup || toType == NodeTypeEnd
	case NodeTypeGroup:
		return toType == NodeTypeGroup || toType == NodeTypeEnd
	default:
		return false
	}
}
