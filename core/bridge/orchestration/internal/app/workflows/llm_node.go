package workflows

import (
	"context"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
	sharedtext "ghost-os/bridge/orchestration/internal/shared/text"
	bridgeTasks "ghost-os/bridge/tasks"
)

func executeLLMNode(
	ctx context.Context,
	deps RuntimeDependencies,
	node bridgeTasks.WorkflowNode,
) NodeOutcome {
	if deps.Client == nil {
		return NodeOutcome{Err: fmt.Errorf("workflow llm runtime is not configured")}
	}
	response, err := deps.Client.Complete(ctx, llm.CompletionRequest{Messages: llmMessages(*node.LLM)})
	if err != nil {
		return NodeOutcome{Err: err}
	}
	text := responseText(response)
	if text == "" {
		return NodeOutcome{Err: fmt.Errorf("workflow llm node %q returned empty response", node.ID)}
	}
	return NodeOutcome{
		Status:      bridgeTasks.RunStatusSuccess,
		Preview:     sharedtext.TruncateRunes(text, bridgeTasks.MaxResponsePreviewRunes),
		OutputText:  text,
		OutputValue: text,
	}
}

func llmMessages(node bridgeTasks.WorkflowLLMNode) []llm.Message {
	messages := make([]llm.Message, 0, 2)
	if strings.TrimSpace(node.SystemPrompt) != "" {
		messages = append(messages, llm.Message{Role: llm.RoleSystem, Text: node.SystemPrompt})
	}
	messages = append(messages, llm.Message{Role: llm.RoleUser, Text: node.Prompt})
	return messages
}

func responseText(response *llm.CompletionResponse) string {
	if response == nil {
		return ""
	}
	if text := strings.TrimSpace(response.Message.Text); text != "" {
		return text
	}
	return responseContentText(response.Message.Content)
}

func responseContentText(parts []llm.ContentPart) string {
	texts := make([]string, 0, len(parts))
	for _, part := range parts {
		if text := strings.TrimSpace(part.Text); text != "" {
			texts = append(texts, text)
		}
	}
	return strings.TrimSpace(strings.Join(texts, "\n"))
}
