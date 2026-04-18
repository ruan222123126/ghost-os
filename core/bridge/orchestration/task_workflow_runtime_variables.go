package orchestration

import (
	"encoding/json"
	"fmt"
	"strings"
)

type workflowVariableStore struct {
	values  map[string]any
	inputs  map[string]any
	outputs map[string]any
	nodes   map[string]any
}

func newWorkflowVariableStore() workflowVariableStore {
	inputs := make(map[string]any)
	outputs := make(map[string]any)
	nodes := make(map[string]any)
	values := map[string]any{
		"inputs":       inputs,
		"outputs":      outputs,
		"nodes":        nodes,
		"last":         "",
		"last_output":  "",
		"last_node_id": "",
	}
	return workflowVariableStore{
		values:  values,
		inputs:  inputs,
		outputs: outputs,
		nodes:   nodes,
	}
}

func (s *workflowVariableStore) seedInputs(start *WorkflowStartNode) error {
	if start == nil {
		return nil
	}
	for _, input := range start.Inputs {
		name := strings.TrimSpace(input.Name)
		if name == "" || len(input.Default) == 0 {
			continue
		}
		value, err := decodeWorkflowInputValue(input.Default)
		if err != nil {
			return fmt.Errorf("workflow start input %q default parse failed: %w", name, err)
		}
		s.inputs[name] = value
		s.values[name] = value
	}
	return nil
}

func decodeWorkflowInputValue(raw json.RawMessage) (any, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func (s *workflowVariableStore) recordNode(
	nodeID string,
	outputText string,
	outputValue any,
	status string,
) {
	normalizedNodeID := strings.TrimSpace(nodeID)
	if normalizedNodeID == "" {
		return
	}
	value := outputValue
	if value == nil {
		value = outputText
	}
	s.outputs[normalizedNodeID] = value
	s.nodes[normalizedNodeID] = map[string]any{
		"output": value,
		"text":   outputText,
		"status": status,
	}
	s.values[normalizedNodeID] = value
	s.values["last"] = value
	s.values["last_output"] = outputText
	s.values["last_node_id"] = normalizedNodeID
}

func (s workflowVariableStore) resolveArguments(input map[string]any) (map[string]any, error) {
	if len(input) == 0 {
		return map[string]any{}, nil
	}
	resolved, err := resolveWorkflowTemplateValue(input, s.values)
	if err != nil {
		return nil, err
	}
	arguments, ok := resolved.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("workflow tool arguments must resolve to JSON object")
	}
	return arguments, nil
}

func (s workflowVariableStore) resolveString(input string) (string, error) {
	resolved, err := resolveWorkflowTemplateString(input, s.values)
	if err != nil {
		return "", err
	}
	if resolved == nil {
		return "", nil
	}
	if text, ok := resolved.(string); ok {
		return text, nil
	}
	return stringifyWorkflowTemplateValue(resolved), nil
}

func resolveWorkflowActionNode(node WorkflowNode, vars workflowVariableStore) (WorkflowNode, error) {
	switch node.Type {
	case workflowNodeTypeTool:
		return resolveWorkflowToolNode(node, vars)
	case workflowNodeTypeLLM:
		return resolveWorkflowLLMNode(node, vars)
	case workflowNodeTypeAgent:
		return resolveWorkflowAgentNode(node, vars)
	default:
		return node, nil
	}
}

func resolveWorkflowToolNode(node WorkflowNode, vars workflowVariableStore) (WorkflowNode, error) {
	if node.Tool == nil {
		return node, fmt.Errorf("workflow tool node %q payload is missing", node.ID)
	}
	arguments, err := vars.resolveArguments(node.Tool.Arguments)
	if err != nil {
		return node, fmt.Errorf("workflow tool node %q resolve arguments: %w", node.ID, err)
	}
	resolvedNode := node
	resolvedTool := *node.Tool
	resolvedTool.Arguments = arguments
	resolvedNode.Tool = &resolvedTool
	return resolvedNode, nil
}

func resolveWorkflowLLMNode(node WorkflowNode, vars workflowVariableStore) (WorkflowNode, error) {
	if node.LLM == nil {
		return node, fmt.Errorf("workflow llm node %q payload is missing", node.ID)
	}
	prompt, err := vars.resolveString(node.LLM.Prompt)
	if err != nil {
		return node, fmt.Errorf("workflow llm node %q resolve prompt: %w", node.ID, err)
	}
	systemPrompt, err := vars.resolveString(node.LLM.SystemPrompt)
	if err != nil {
		return node, fmt.Errorf("workflow llm node %q resolve system_prompt: %w", node.ID, err)
	}
	resolvedNode := node
	resolvedLLM := *node.LLM
	resolvedLLM.Prompt = prompt
	resolvedLLM.SystemPrompt = systemPrompt
	resolvedNode.LLM = &resolvedLLM
	return resolvedNode, nil
}

func resolveWorkflowAgentNode(node WorkflowNode, vars workflowVariableStore) (WorkflowNode, error) {
	if node.Agent == nil {
		return node, fmt.Errorf("workflow agent node %q payload is missing", node.ID)
	}
	message, err := vars.resolveString(node.Agent.Message)
	if err != nil {
		return node, fmt.Errorf("workflow agent node %q resolve message: %w", node.ID, err)
	}
	resolvedNode := node
	resolvedAgent := *node.Agent
	resolvedAgent.Message = message
	resolvedNode.Agent = &resolvedAgent
	return resolvedNode, nil
}
