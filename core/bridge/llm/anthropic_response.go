package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (c *Client) parseAnthropicProviderResponse(raw []byte) (*CompletionResponse, error) {
	var response anthropicResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("decode anthropic response: %w", err)
	}
	return anthropicToCompletionResponse(response)
}

// anthropicToCompletionResponse 把 Anthropic content blocks 还原为统一消息结构。
func anthropicToCompletionResponse(response anthropicResponse) (*CompletionResponse, error) {
	message := Message{
		Role: RoleAssistant,
	}

	textParts := make([]string, 0, 1)
	toolCalls := make([]ToolCall, 0, 1)
	for _, block := range response.Content {
		switch block.Type {
		case "text":
			if strings.TrimSpace(block.Text) != "" {
				textParts = append(textParts, block.Text)
			}
		case "tool_use":
			args := json.RawMessage(`{}`)
			if block.Input != nil {
				encoded, err := json.Marshal(block.Input)
				if err != nil {
					return nil, fmt.Errorf("encode anthropic tool input for %q: %w", block.Name, err)
				}
				args = encoded
			}

			toolCalls = append(toolCalls, ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: normalizeJSONObject(args),
			})
		}
	}

	if len(textParts) > 0 {
		message.Text = strings.Join(textParts, "\n")
	}
	if len(toolCalls) > 0 {
		message.ToolCalls = toolCalls
	}
	finishReason, err := anthropicFinishReason(response.StopReason)
	if err != nil {
		return nil, err
	}

	return &CompletionResponse{
		Message:      message,
		FinishReason: finishReason,
		Usage: Usage{
			PromptTokens:     response.Usage.InputTokens,
			CompletionTokens: response.Usage.OutputTokens,
			TotalTokens:      response.Usage.InputTokens + response.Usage.OutputTokens,
		},
	}, nil
}

// anthropicFinishReason 使用严格映射，未知 stop_reason 直接报错。
func anthropicFinishReason(reason string) (FinishReason, error) {
	switch strings.TrimSpace(reason) {
	case "end_turn", "stop_sequence":
		return FinishStop, nil
	case "tool_use":
		return FinishToolCalls, nil
	case "max_tokens":
		return FinishLength, nil
	default:
		return "", fmt.Errorf("unsupported anthropic stop_reason %q", reason)
	}
}
