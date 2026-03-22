package tasks

import "strings"

type WorkflowDefinition struct {
	Nodes []WorkflowNode `json:"nodes"`
	Edges []WorkflowEdge `json:"edges"`
}

type WorkflowNode struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type WorkflowEdge struct {
	FromNodeID string `json:"from_node_id"`
	ToNodeID   string `json:"to_node_id"`
}

func CloneWorkflowDefinition(input *WorkflowDefinition) *WorkflowDefinition {
	if input == nil {
		return nil
	}
	out := &WorkflowDefinition{
		Nodes: make([]WorkflowNode, len(input.Nodes)),
		Edges: make([]WorkflowEdge, len(input.Edges)),
	}
	for index, node := range input.Nodes {
		out.Nodes[index] = WorkflowNode{
			ID:   strings.TrimSpace(node.ID),
			Type: strings.TrimSpace(node.Type),
		}
	}
	for index, edge := range input.Edges {
		out.Edges[index] = WorkflowEdge{
			FromNodeID: strings.TrimSpace(edge.FromNodeID),
			ToNodeID:   strings.TrimSpace(edge.ToNodeID),
		}
	}
	return out
}
