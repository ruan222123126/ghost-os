package tasks

import "strings"

const (
	OrchestrationNodeTypeGroup = "group"
	OrchestrationNodeTypeAgent = "agent"

	OrchestrationEdgeKindControl = "control"
	OrchestrationEdgeKindMember  = "member"

	OrchestrationSpeakingModeSequential = "sequential"
	OrchestrationSpeakingModeParallel   = "parallel"
	OrchestrationSpeakingModeOwner      = "owner"
)

type OrchestrationDefinition struct {
	Nodes []OrchestrationNode `json:"nodes"`
	Edges []OrchestrationEdge `json:"edges"`
}

type OrchestrationNode struct {
	ID    string                  `json:"id"`
	Type  string                  `json:"type"`
	Group *OrchestrationGroupNode `json:"group,omitempty"`
	Agent *OrchestrationAgentNode `json:"agent,omitempty"`
}

type OrchestrationGroupNode struct {
	Title         string `json:"title"`
	SharedContext string `json:"shared_context,omitempty"`
	SpeakingMode  string `json:"speaking_mode"`
	OwnerAgentID  string `json:"owner_agent_id,omitempty"`
	MaxRounds     int    `json:"max_rounds"`
}

type OrchestrationAgentNode struct {
	Title            string                `json:"title"`
	Message          string                `json:"message"`
	RuntimeOverrides *TaskRuntimeOverrides `json:"runtime_overrides,omitempty"`
}

type OrchestrationEdge struct {
	FromNodeID string `json:"from_node_id"`
	ToNodeID   string `json:"to_node_id"`
	Kind       string `json:"kind"`
}

func CloneOrchestrationDefinition(input *OrchestrationDefinition) *OrchestrationDefinition {
	if input == nil {
		return nil
	}
	out := &OrchestrationDefinition{
		Nodes: make([]OrchestrationNode, len(input.Nodes)),
		Edges: make([]OrchestrationEdge, len(input.Edges)),
	}
	for index, node := range input.Nodes {
		out.Nodes[index] = OrchestrationNode{
			ID:    strings.TrimSpace(node.ID),
			Type:  strings.TrimSpace(node.Type),
			Group: cloneOrchestrationGroupNode(node.Group),
			Agent: cloneOrchestrationAgentNode(node.Agent),
		}
	}
	for index, edge := range input.Edges {
		out.Edges[index] = OrchestrationEdge{
			FromNodeID: strings.TrimSpace(edge.FromNodeID),
			ToNodeID:   strings.TrimSpace(edge.ToNodeID),
			Kind:       strings.TrimSpace(edge.Kind),
		}
	}
	return out
}

func cloneOrchestrationGroupNode(input *OrchestrationGroupNode) *OrchestrationGroupNode {
	if input == nil {
		return nil
	}
	return &OrchestrationGroupNode{
		Title:         strings.TrimSpace(input.Title),
		SharedContext: strings.TrimSpace(input.SharedContext),
		SpeakingMode:  strings.TrimSpace(input.SpeakingMode),
		OwnerAgentID:  strings.TrimSpace(input.OwnerAgentID),
		MaxRounds:     input.MaxRounds,
	}
}

func cloneOrchestrationAgentNode(input *OrchestrationAgentNode) *OrchestrationAgentNode {
	if input == nil {
		return nil
	}
	return &OrchestrationAgentNode{
		Title:            strings.TrimSpace(input.Title),
		Message:          strings.TrimSpace(input.Message),
		RuntimeOverrides: CloneTaskRuntimeOverrides(input.RuntimeOverrides),
	}
}
