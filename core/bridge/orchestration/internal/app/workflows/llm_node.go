package workflows

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/llm"
	sharedtext "ghost-os/bridge/orchestration/internal/shared/text"
	bridgeTasks "ghost-os/bridge/tasks"
)

func executeLLMNode(
	ctx context.Context,
	cmd ExecuteCommand,
	node bridgeTasks.WorkflowNode,
	branchID string,
	iteration int,
) NodeOutcome {
	handle, err := startLLMRunCard(ctx, cmd.Cards, node, branchID, iteration)
	if err != nil {
		return NodeOutcome{Err: err}
	}
	response, err := completeLLMNode(ctx, cmd.Runtime, node)
	if err != nil {
		return llmNodeError(ctx, handle, err)
	}
	text := responseText(response)
	if text == "" {
		return llmNodeError(ctx, handle, fmt.Errorf("workflow llm node %q returned empty response", node.ID))
	}
	if err := finishLLMRunCard(ctx, handle, text); err != nil {
		return NodeOutcome{Err: err}
	}
	return NodeOutcome{
		Status:      bridgeTasks.RunStatusSuccess,
		Preview:     sharedtext.TruncateRunes(text, bridgeTasks.MaxResponsePreviewRunes),
		OutputText:  text,
		OutputValue: text,
	}
}

func completeLLMNode(
	ctx context.Context,
	deps RuntimeDependencies,
	node bridgeTasks.WorkflowNode,
) (*llm.CompletionResponse, error) {
	if deps.Client == nil {
		return nil, fmt.Errorf("workflow llm runtime is not configured")
	}
	return deps.Client.Complete(ctx, llm.CompletionRequest{Messages: llmMessages(*node.LLM)})
}

func startLLMRunCard(
	ctx context.Context,
	observer CardObserver,
	node bridgeTasks.WorkflowNode,
	branchID string,
	iteration int,
) (CardHandle, error) {
	if observer == nil {
		return nil, nil
	}
	return observer.StartCard(ctx, CardStartRequest{
		Kind:      bridgeTasks.RunCardKindWorkflowLLM,
		Title:     node.ID,
		NodeID:    node.ID,
		NodeType:  node.Type,
		BranchID:  branchID,
		Iteration: iteration,
		StartedAt: time.Now().UTC(),
	})
}

func llmNodeError(ctx context.Context, handle CardHandle, err error) NodeOutcome {
	if finishErr := finishLLMRunCardError(ctx, handle, err); finishErr != nil {
		return NodeOutcome{Err: finishErr}
	}
	return NodeOutcome{Err: err}
}

func finishLLMRunCard(ctx context.Context, handle CardHandle, text string) error {
	if handle == nil {
		return nil
	}
	return handle.Finish(ctx, CardFinishRequest{
		Status:     bridgeTasks.RunStatusSuccess,
		Preview:    text,
		FinalText:  text,
		FinishedAt: time.Now().UTC(),
	})
}

func finishLLMRunCardError(ctx context.Context, handle CardHandle, err error) error {
	if handle == nil {
		return nil
	}
	return handle.Finish(ctx, CardFinishRequest{
		Status:     bridgeTasks.RunStatusError,
		Error:      err.Error(),
		FinishedAt: time.Now().UTC(),
	})
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
