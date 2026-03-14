package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (c *Client) parseOpenAIProviderResponse(raw []byte) (*CompletionResponse, error) {
	var response openAIResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("decode openai-compatible response: %w", err)
	}

	return openAIToCompletionResponse(response)
}

// openAIToCompletionResponse 把 provider 响应映射回统一结构。
func openAIToCompletionResponse(response openAIResponse) (*CompletionResponse, error) {
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("openai-compatible response has no choices")
	}

	choice := response.Choices[0]
	message, err := openAIToMessage(choice.Message)
	if err != nil {
		return nil, err
	}
	finishReason, err := openAIFinishReason(normalizeOpenAIFinishReason(choice.FinishReason, message))
	if err != nil {
		return nil, err
	}

	return &CompletionResponse{
		Message:      message,
		FinishReason: finishReason,
		Usage: Usage{
			PromptTokens:     response.Usage.PromptTokens,
			CompletionTokens: response.Usage.CompletionTokens,
			TotalTokens:      response.Usage.TotalTokens,
		},
	}, nil
}

func openAIToMessage(msg openAIMessage) (Message, error) {
	out := Message{
		Role:       Role(strings.TrimSpace(msg.Role)),
		Text:       contentToText(msg.Content),
		ToolCallID: strings.TrimSpace(msg.ToolCallID),
	}

	if out.Role == "" {
		out.Role = RoleAssistant
	}

	if len(msg.ToolCalls) > 0 {
		out.ToolCalls = make([]ToolCall, 0, len(msg.ToolCalls))
		for _, call := range msg.ToolCalls {
			out.ToolCalls = append(out.ToolCalls, ToolCall{
				ID:   call.ID,
				Name: call.Function.Name,
				// 保留 provider 原始 arguments，避免把空字符串静默归一化为 {}。
				// 空/非法参数应在 agent 层按无效 tool call 处理。
				Arguments: json.RawMessage(call.Function.Arguments),
			})
		}
	}

	return out, nil
}

// normalizeOpenAIFinishReason 为不规范的 OpenAI 兼容实现提供最小兜底。
// 当 finish_reason 缺失时，优先按 tool_calls 判定，否则回落为 stop。
func normalizeOpenAIFinishReason(reason string, message Message) string {
	if strings.TrimSpace(reason) != "" {
		return reason
	}
	if len(message.ToolCalls) > 0 {
		return "tool_calls"
	}
	return "stop"
}

// openAIFinishReason 采用严格映射，未知值直接返回错误避免静默降级。
func openAIFinishReason(reason string) (FinishReason, error) {
	switch strings.TrimSpace(reason) {
	case "stop":
		return FinishStop, nil
	case "tool_calls":
		return FinishToolCalls, nil
	case "length":
		return FinishLength, nil
	default:
		return "", fmt.Errorf("unsupported openai finish_reason %q", reason)
	}
}
