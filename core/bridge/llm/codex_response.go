package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (c *Client) parseCodexProviderResponse(raw []byte) (*CompletionResponse, error) {
	var response codexResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("decode codex response: %w", err)
	}
	completion, err := codexToCompletionResponse(response)
	if err != nil {
		return nil, err
	}
	c.stampCodexConversationState(completion)
	return completion, nil
}

func codexToCompletionResponse(response codexResponse) (*CompletionResponse, error) {
	message := Message{Role: RoleAssistant}
	textParts := make([]string, 0, 1)
	toolCalls := make([]ToolCall, 0, 1)
	for _, item := range response.Output {
		switch strings.TrimSpace(item.Type) {
		case "message":
			for _, content := range item.Content {
				switch strings.TrimSpace(content.Type) {
				case "output_text", "text":
					if strings.TrimSpace(content.Text) != "" {
						textParts = append(textParts, content.Text)
					}
				}
			}
		case "function_call":
			toolCalls = append(toolCalls, ToolCall{
				ID:        codexCallID(item),
				Name:      strings.TrimSpace(item.Name),
				Arguments: json.RawMessage(item.Arguments),
			})
		}
	}
	if len(textParts) > 0 {
		message.Text = strings.Join(textParts, "\n")
	}
	if len(toolCalls) > 0 {
		message.ToolCalls = toolCalls
	}

	finishReason := FinishStop
	switch {
	case len(toolCalls) > 0:
		finishReason = FinishToolCalls
	case codexIncompleteByLength(response):
		finishReason = FinishLength
	}

	out := &CompletionResponse{
		Message:      message,
		FinishReason: finishReason,
		ConversationState: ConversationState{
			Provider:           ProviderCodex,
			PreviousResponseID: strings.TrimSpace(response.ID),
		},
	}
	if response.Usage != nil {
		out.Usage = Usage{
			PromptTokens:     response.Usage.InputTokens,
			CompletionTokens: response.Usage.OutputTokens,
			TotalTokens:      response.Usage.TotalTokens,
		}
	}
	return out, nil
}

func codexIncompleteByLength(response codexResponse) bool {
	if strings.TrimSpace(response.Status) != "incomplete" || response.IncompleteDetails == nil {
		return false
	}
	switch strings.TrimSpace(response.IncompleteDetails.Reason) {
	case "max_output_tokens", "max_completion_tokens":
		return true
	default:
		return false
	}
}

func codexCallID(item codexOutputItem) string {
	if value := strings.TrimSpace(item.CallID); value != "" {
		return value
	}
	return strings.TrimSpace(item.ID)
}

func (c *Client) stampCodexConversationState(response *CompletionResponse) {
	if response == nil {
		return
	}
	response.ConversationState.Provider = ProviderCodex
	response.ConversationState.BaseURL = strings.TrimSpace(c.opts.BaseURL)
	response.ConversationState.Model = strings.TrimSpace(c.opts.Model)
}
