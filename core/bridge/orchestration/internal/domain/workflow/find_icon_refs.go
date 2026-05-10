package workflow

import (
	"fmt"

	bridgeTasks "ghost-os/bridge/tasks"
)

func ResolveActionNode(node Node, findIconOutput any) (Node, error) {
	switch node.Type {
	case NodeTypeTool:
		return resolveToolNode(node, findIconOutput)
	case NodeTypeLLM:
		return resolveLLMNode(node, findIconOutput)
	case NodeTypeAgent:
		return resolveAgentNode(node, findIconOutput)
	default:
		return node, nil
	}
}

func ResolveIfValue(value string, findIconOutput any) (string, error) {
	return ResolveFindIconString(value, findIconOutput)
}

func ResolveFindIconString(input string, findIconOutput any) (string, error) {
	resolved, err := ResolveFindIconTemplateString(input, findIconOutput)
	if err != nil {
		return "", err
	}
	if text, ok := resolved.(string); ok {
		return text, nil
	}
	return StringifyFindIconTemplateValue(resolved), nil
}

func resolveToolNode(node Node, findIconOutput any) (Node, error) {
	if node.Tool == nil {
		return node, fmt.Errorf("workflow tool node %q payload is missing", node.ID)
	}
	resolvedArgs, err := ResolveFindIconTemplateValue(node.Tool.Arguments, findIconOutput)
	if err != nil {
		return node, fmt.Errorf("workflow tool node %q resolve arguments: %w", node.ID, err)
	}
	arguments, ok := normalizeResolvedArguments(resolvedArgs)
	if !ok {
		return node, fmt.Errorf("workflow tool arguments must resolve to JSON object")
	}
	resolved := node
	toolNode := *node.Tool
	toolNode.Arguments = arguments
	resolved.Tool = &toolNode
	return resolved, nil
}

func normalizeResolvedArguments(resolvedArgs any) (map[string]any, bool) {
	if resolvedArgs == nil {
		return map[string]any{}, true
	}
	arguments, ok := resolvedArgs.(map[string]any)
	return arguments, ok
}

func resolveLLMNode(node Node, findIconOutput any) (Node, error) {
	if node.LLM == nil {
		return node, fmt.Errorf("workflow llm node %q payload is missing", node.ID)
	}
	prompt, err := ResolveFindIconString(node.LLM.Prompt, findIconOutput)
	if err != nil {
		return node, fmt.Errorf("workflow llm node %q resolve prompt: %w", node.ID, err)
	}
	systemPrompt, err := ResolveFindIconString(node.LLM.SystemPrompt, findIconOutput)
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

func resolveAgentNode(node Node, findIconOutput any) (Node, error) {
	if node.Agent == nil {
		return node, fmt.Errorf("workflow agent node %q payload is missing", node.ID)
	}
	message, err := ResolveFindIconString(node.Agent.Message, findIconOutput)
	if err != nil {
		return node, fmt.Errorf("workflow agent node %q resolve message: %w", node.ID, err)
	}
	overrides, err := resolveAgentRuntimeOverrides(node.ID, node.Agent.RuntimeOverrides, findIconOutput)
	if err != nil {
		return node, err
	}
	resolved := node
	agentNode := *node.Agent
	agentNode.Message = message
	agentNode.RuntimeOverrides = overrides
	resolved.Agent = &agentNode
	return resolved, nil
}

func resolveAgentRuntimeOverrides(
	nodeID string,
	input *RuntimeOverrides,
	findIconOutput any,
) (*RuntimeOverrides, error) {
	if input == nil || input.SystemPrompt == "" {
		return bridgeTasks.CloneTaskRuntimeOverrides(input), nil
	}
	systemPrompt, err := ResolveFindIconString(input.SystemPrompt, findIconOutput)
	if err != nil {
		return nil, fmt.Errorf("workflow agent node %q resolve runtime_overrides.system_prompt: %w", nodeID, err)
	}
	overrides := bridgeTasks.CloneTaskRuntimeOverrides(input)
	overrides.SystemPrompt = systemPrompt
	return overrides, nil
}
