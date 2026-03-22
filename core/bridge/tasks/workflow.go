package tasks

import "strings"

type WorkflowDefinition struct {
	Nodes []WorkflowNode `json:"nodes"`
	Edges []WorkflowEdge `json:"edges"`
}

type WorkflowNode struct {
	ID    string             `json:"id"`
	Type  string             `json:"type"`
	Tool  *WorkflowToolNode  `json:"tool,omitempty"`
	LLM   *WorkflowLLMNode   `json:"llm,omitempty"`
	Agent *WorkflowAgentNode `json:"agent,omitempty"`
}

type WorkflowToolNode struct {
	ToolName  string         `json:"tool_name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type WorkflowLLMNode struct {
	Prompt       string `json:"prompt"`
	SystemPrompt string `json:"system_prompt,omitempty"`
}

type WorkflowAgentNode struct {
	Message string `json:"message"`
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
			ID:    strings.TrimSpace(node.ID),
			Type:  strings.TrimSpace(node.Type),
			Tool:  cloneWorkflowToolNode(node.Tool),
			LLM:   cloneWorkflowLLMNode(node.LLM),
			Agent: cloneWorkflowAgentNode(node.Agent),
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

func cloneWorkflowToolNode(input *WorkflowToolNode) *WorkflowToolNode {
	if input == nil {
		return nil
	}
	return &WorkflowToolNode{
		ToolName:  strings.TrimSpace(input.ToolName),
		Arguments: CloneActionParams(input.Arguments),
	}
}

func cloneWorkflowLLMNode(input *WorkflowLLMNode) *WorkflowLLMNode {
	if input == nil {
		return nil
	}
	return &WorkflowLLMNode{
		Prompt:       strings.TrimSpace(input.Prompt),
		SystemPrompt: strings.TrimSpace(input.SystemPrompt),
	}
}

func cloneWorkflowAgentNode(input *WorkflowAgentNode) *WorkflowAgentNode {
	if input == nil {
		return nil
	}
	return &WorkflowAgentNode{Message: strings.TrimSpace(input.Message)}
}
