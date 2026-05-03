package orchestration

import (
	"fmt"
	"strings"
)

func resolveWorkflowActionNode(node WorkflowNode, findIconOutput any) (WorkflowNode, error) {
	switch node.Type {
	case workflowNodeTypeTool:
		return resolveWorkflowToolNode(node, findIconOutput)
	case workflowNodeTypeLLM:
		return resolveWorkflowLLMNode(node, findIconOutput)
	case workflowNodeTypeAgent:
		return resolveWorkflowAgentNode(node, findIconOutput)
	default:
		return node, nil
	}
}

func resolveWorkflowToolNode(node WorkflowNode, findIconOutput any) (WorkflowNode, error) {
	if node.Tool == nil {
		return node, fmt.Errorf("workflow tool node %q payload is missing", node.ID)
	}
	resolvedArgs, err := resolveWorkflowFindIconTemplateValue(node.Tool.Arguments, findIconOutput)
	if err != nil {
		return node, fmt.Errorf("workflow tool node %q resolve arguments: %w", node.ID, err)
	}
	if resolvedArgs == nil {
		resolvedArgs = map[string]any{}
	}
	arguments, ok := resolvedArgs.(map[string]any)
	if !ok {
		return node, fmt.Errorf("workflow tool arguments must resolve to JSON object")
	}
	resolved := node
	toolNode := *node.Tool
	toolNode.Arguments = arguments
	resolved.Tool = &toolNode
	return resolved, nil
}

func resolveWorkflowLLMNode(node WorkflowNode, findIconOutput any) (WorkflowNode, error) {
	if node.LLM == nil {
		return node, fmt.Errorf("workflow llm node %q payload is missing", node.ID)
	}
	prompt, err := resolveWorkflowFindIconString(node.LLM.Prompt, findIconOutput)
	if err != nil {
		return node, fmt.Errorf("workflow llm node %q resolve prompt: %w", node.ID, err)
	}
	systemPrompt, err := resolveWorkflowFindIconString(node.LLM.SystemPrompt, findIconOutput)
	if err != nil {
		return node, fmt.Errorf("workflow llm node %q resolve system_prompt: %w", node.ID, err)
	}
	resolved := node
	llmNode := *node.LLM
	llmNode.Prompt = prompt
	llmNode.SystemPrompt = systemPrompt
	resolved.LLM = &llmNode
	return resolved, nil
}

func resolveWorkflowAgentNode(node WorkflowNode, findIconOutput any) (WorkflowNode, error) {
	if node.Agent == nil {
		return node, fmt.Errorf("workflow agent node %q payload is missing", node.ID)
	}
	message, err := resolveWorkflowFindIconString(node.Agent.Message, findIconOutput)
	if err != nil {
		return node, fmt.Errorf("workflow agent node %q resolve message: %w", node.ID, err)
	}
	resolved := node
	agentNode := *node.Agent
	agentNode.Message = message
	agentNode.RuntimeOverrides, err = resolveWorkflowAgentRuntimeOverrides(node.ID, node.Agent.RuntimeOverrides, findIconOutput)
	if err != nil {
		return node, err
	}
	resolved.Agent = &agentNode
	return resolved, nil
}

func resolveWorkflowAgentRuntimeOverrides(
	nodeID string,
	input *TaskRuntimeOverrides,
	findIconOutput any,
) (*TaskRuntimeOverrides, error) {
	if input == nil || strings.TrimSpace(input.SystemPrompt) == "" {
		return cloneTaskRuntimeOverrides(input), nil
	}
	systemPrompt, err := resolveWorkflowFindIconString(input.SystemPrompt, findIconOutput)
	if err != nil {
		return nil, fmt.Errorf("workflow agent node %q resolve runtime_overrides.system_prompt: %w", nodeID, err)
	}
	overrides := cloneTaskRuntimeOverrides(input)
	if overrides == nil {
		return nil, nil
	}
	overrides.SystemPrompt = systemPrompt
	return overrides, nil
}

func resolveWorkflowIfValue(value string, findIconOutput any) (string, error) {
	return resolveWorkflowFindIconString(value, findIconOutput)
}

func resolveWorkflowFindIconString(input string, findIconOutput any) (string, error) {
	resolved, err := resolveWorkflowFindIconTemplateString(input, findIconOutput)
	if err != nil {
		return "", err
	}
	if text, ok := resolved.(string); ok {
		return text, nil
	}
	return stringifyWorkflowFindIconTemplateValue(resolved), nil
}

func extractWorkflowFindIconOutput(node WorkflowNode, outputValue any) (any, bool) {
	if node.Type != workflowNodeTypeTool || node.Tool == nil || node.Tool.ToolName != screenControlToolID {
		return nil, false
	}
	return extractWorkflowFindIconOutputValue(outputValue)
}

func extractWorkflowFindIconOutputValue(outputValue any) (any, bool) {
	record, ok := outputValue.(map[string]any)
	if !ok {
		return nil, false
	}
	if _, exists := record["matches"]; exists {
		return buildWorkflowFindIconReferenceValue(record)
	}
	rawSteps, ok := record["steps"].([]map[string]any)
	if ok {
		for index := len(rawSteps) - 1; index >= 0; index -= 1 {
			if rawSteps[index]["action"] == "find_icon" {
				return buildWorkflowFindIconReferenceValue(rawSteps[index]["output"])
			}
		}
		return nil, false
	}
	steps, ok := record["steps"].([]any)
	if !ok {
		return nil, false
	}
	for index := len(steps) - 1; index >= 0; index -= 1 {
		step, ok := steps[index].(map[string]any)
		if ok && step["action"] == "find_icon" {
			return buildWorkflowFindIconReferenceValue(step["output"])
		}
	}
	return nil, false
}

func buildWorkflowFindIconReferenceValue(outputValue any) (map[string]any, bool) {
	match, err := workflowFindIconMatchCenter(outputValue)
	if err != nil {
		return nil, false
	}
	value := map[string]any{
		"x": match.x,
		"y": match.y,
	}
	if match.displayID >= 0 {
		value["display_id"] = match.displayID
	}
	return value, true
}
