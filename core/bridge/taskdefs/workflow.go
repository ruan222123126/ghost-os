package taskdefs

import (
	"encoding/json"
	"strings"
)

type WorkflowDefinition struct {
	Nodes []WorkflowNode `json:"nodes"`
	Edges []WorkflowEdge `json:"edges"`
}

type WorkflowNode struct {
	ID    string             `json:"id"`
	Type  string             `json:"type"`
	Start *WorkflowStartNode `json:"start,omitempty"`
	Tool  *WorkflowToolNode  `json:"tool,omitempty"`
	LLM   *WorkflowLLMNode   `json:"llm,omitempty"`
	Agent *WorkflowAgentNode `json:"agent,omitempty"`
	If    *WorkflowIfNode    `json:"if,omitempty"`
	Loop  *WorkflowLoopNode  `json:"loop,omitempty"`
}

type WorkflowStartNode struct {
	Inputs []WorkflowInputVariable `json:"inputs,omitempty"`
}

type WorkflowInputVariable struct {
	Name        string          `json:"name"`
	Type        string          `json:"type"`
	Required    bool            `json:"required,omitempty"`
	Default     json.RawMessage `json:"default,omitempty"`
	Description string          `json:"description,omitempty"`
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
	Message          string                `json:"message"`
	RuntimeOverrides *TaskRuntimeOverrides `json:"runtime_overrides,omitempty"`
}

type WorkflowIfNode struct {
	SourceNodeID string `json:"source_node_id,omitempty"`
	Operator     string `json:"operator"`
	Value        string `json:"value,omitempty"`
	TrueNodeID   string `json:"true_node_id"`
	FalseNodeID  string `json:"false_node_id"`
}

type WorkflowLoopNode struct {
	MaxIterations int    `json:"max_iterations"`
	BodyNodeID    string `json:"body_node_id"`
	ExitNodeID    string `json:"exit_node_id"`
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
			Start: cloneWorkflowStartNode(node.Start),
			Tool:  cloneWorkflowToolNode(node.Tool),
			LLM:   cloneWorkflowLLMNode(node.LLM),
			Agent: cloneWorkflowAgentNode(node.Agent),
			If:    cloneWorkflowIfNode(node.If),
			Loop:  cloneWorkflowLoopNode(node.Loop),
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

func cloneWorkflowStartNode(input *WorkflowStartNode) *WorkflowStartNode {
	if input == nil {
		return nil
	}
	out := &WorkflowStartNode{
		Inputs: make([]WorkflowInputVariable, len(input.Inputs)),
	}
	for index, variable := range input.Inputs {
		out.Inputs[index] = cloneWorkflowInputVariable(variable)
	}
	return out
}

func cloneWorkflowInputVariable(input WorkflowInputVariable) WorkflowInputVariable {
	return WorkflowInputVariable{
		Name:        strings.TrimSpace(input.Name),
		Type:        strings.TrimSpace(input.Type),
		Required:    input.Required,
		Default:     cloneWorkflowInputDefault(input.Default),
		Description: strings.TrimSpace(input.Description),
	}
}

func cloneWorkflowInputDefault(input json.RawMessage) json.RawMessage {
	if len(input) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), input...)
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
	return &WorkflowAgentNode{
		Message:          strings.TrimSpace(input.Message),
		RuntimeOverrides: CloneTaskRuntimeOverrides(input.RuntimeOverrides),
	}
}

func cloneWorkflowIfNode(input *WorkflowIfNode) *WorkflowIfNode {
	if input == nil {
		return nil
	}
	return &WorkflowIfNode{
		SourceNodeID: strings.TrimSpace(input.SourceNodeID),
		Operator:     strings.TrimSpace(input.Operator),
		Value:        strings.TrimSpace(input.Value),
		TrueNodeID:   strings.TrimSpace(input.TrueNodeID),
		FalseNodeID:  strings.TrimSpace(input.FalseNodeID),
	}
}

func cloneWorkflowLoopNode(input *WorkflowLoopNode) *WorkflowLoopNode {
	if input == nil {
		return nil
	}
	return &WorkflowLoopNode{
		MaxIterations: input.MaxIterations,
		BodyNodeID:    strings.TrimSpace(input.BodyNodeID),
		ExitNodeID:    strings.TrimSpace(input.ExitNodeID),
	}
}
