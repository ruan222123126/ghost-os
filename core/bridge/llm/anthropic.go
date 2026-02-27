package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type anthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema"`
}

type anthropicContentBlock struct {
	Type      string         `json:"type"`
	Text      string         `json:"text,omitempty"`
	ID        string         `json:"id,omitempty"`
	Name      string         `json:"name,omitempty"`
	Input     map[string]any `json:"input,omitempty"`
	ToolUseID string         `json:"tool_use_id,omitempty"`
	Content   any            `json:"content,omitempty"`
	IsError   bool           `json:"is_error,omitempty"`
}

type anthropicResponse struct {
	ID         string                  `json:"id"`
	Role       string                  `json:"role"`
	Content    []anthropicContentBlock `json:"content"`
	StopReason string                  `json:"stop_reason"`
	Usage      anthropicUsage          `json:"usage"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

func (c *Client) completeAnthropic(ctx context.Context, messages []ChatMessage, tools []ToolDef) (*ChatResponse, error) {
	request, err := buildAnthropicRequest(c.opts.Model, c.opts.AnthropicMaxTokens, messages, tools)
	if err != nil {
		return nil, err
	}

	headers := make(map[string]string, len(c.opts.Headers)+2)
	if c.opts.APIKey != "" {
		headers["x-api-key"] = c.opts.APIKey
	}
	headers["anthropic-version"] = c.opts.AnthropicVersion
	mergeStringHeaders(headers, c.opts.Headers)

	raw, statusCode, err := c.postJSON(ctx, c.opts.ChatPath, request, headers)
	if err != nil {
		return nil, err
	}
	if err := ensureSuccessStatus(statusCode, raw); err != nil {
		return nil, err
	}

	var response anthropicResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("decode anthropic response: %w", err)
	}

	return anthropicToChatResponse(response)
}

func buildAnthropicRequest(model string, maxTokens int, messages []ChatMessage, tools []ToolDef) (anthropicRequest, error) {
	convertedMessages := make([]anthropicMessage, 0, len(messages))
	systemParts := make([]string, 0, 2)

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			system := strings.TrimSpace(ContentText(msg.Content))
			if system != "" {
				systemParts = append(systemParts, system)
			}
		case "user":
			convertedMessages = append(convertedMessages, anthropicMessage{
				Role:    "user",
				Content: ContentText(msg.Content),
			})
		case "assistant":
			assistantMessage, err := toAnthropicAssistantMessage(msg)
			if err != nil {
				return anthropicRequest{}, err
			}
			convertedMessages = append(convertedMessages, assistantMessage)
		case "tool":
			convertedMessages = append(convertedMessages, anthropicMessage{
				Role: "user",
				Content: []anthropicContentBlock{
					{
						Type:      "tool_result",
						ToolUseID: msg.ToolCallID,
						Content:   ContentText(msg.Content),
					},
				},
			})
		default:
			return anthropicRequest{}, fmt.Errorf("unsupported chat role for anthropic: %q", msg.Role)
		}
	}

	convertedTools := make([]anthropicTool, 0, len(tools))
	for _, tool := range tools {
		if tool.Type != "function" {
			continue
		}
		convertedTools = append(convertedTools, anthropicTool{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			InputSchema: tool.Function.Parameters,
		})
	}

	request := anthropicRequest{
		Model:     model,
		MaxTokens: maxTokens,
		Messages:  convertedMessages,
		Tools:     convertedTools,
	}
	if len(systemParts) > 0 {
		request.System = strings.Join(systemParts, "\n\n")
	}

	return request, nil
}

func toAnthropicAssistantMessage(msg ChatMessage) (anthropicMessage, error) {
	blocks := make([]anthropicContentBlock, 0, len(msg.ToolCalls)+1)

	text := strings.TrimSpace(ContentText(msg.Content))
	if text != "" {
		blocks = append(blocks, anthropicContentBlock{
			Type: "text",
			Text: text,
		})
	}

	for _, call := range msg.ToolCalls {
		input, err := parseToolArguments(call.Function.Arguments)
		if err != nil {
			return anthropicMessage{}, fmt.Errorf("parse tool arguments for %q: %w", call.Function.Name, err)
		}

		blocks = append(blocks, anthropicContentBlock{
			Type:  "tool_use",
			ID:    call.ID,
			Name:  call.Function.Name,
			Input: input,
		})
	}

	if len(blocks) == 0 {
		return anthropicMessage{
			Role:    "assistant",
			Content: "",
		}, nil
	}

	if len(msg.ToolCalls) == 0 && len(blocks) == 1 && blocks[0].Type == "text" {
		return anthropicMessage{
			Role:    "assistant",
			Content: blocks[0].Text,
		}, nil
	}

	return anthropicMessage{
		Role:    "assistant",
		Content: blocks,
	}, nil
}

func parseToolArguments(arguments string) (map[string]any, error) {
	raw := strings.TrimSpace(arguments)
	if raw == "" {
		return map[string]any{}, nil
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func anthropicToChatResponse(response anthropicResponse) (*ChatResponse, error) {
	message := ChatMessage{
		Role: "assistant",
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
			argumentsJSON := []byte("{}")
			if block.Input != nil {
				encoded, err := json.Marshal(block.Input)
				if err != nil {
					return nil, fmt.Errorf("encode anthropic tool input for %q: %w", block.Name, err)
				}
				argumentsJSON = encoded
			}
			toolCalls = append(toolCalls, ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: FunctionCall{
					Name:      block.Name,
					Arguments: string(argumentsJSON),
				},
			})
		}
	}

	if len(textParts) > 0 {
		message.Content = strings.Join(textParts, "\n")
	}
	if len(toolCalls) > 0 {
		message.ToolCalls = toolCalls
	}

	return &ChatResponse{
		ID: response.ID,
		Choices: []Choice{
			{
				Index:        0,
				Message:      message,
				FinishReason: anthropicStopReasonToFinishReason(response.StopReason),
			},
		},
		Usage: Usage{
			PromptTokens:     response.Usage.InputTokens,
			CompletionTokens: response.Usage.OutputTokens,
			TotalTokens:      response.Usage.InputTokens + response.Usage.OutputTokens,
		},
	}, nil
}

func anthropicStopReasonToFinishReason(reason string) string {
	switch strings.TrimSpace(reason) {
	case "tool_use":
		return "tool_calls"
	case "end_turn", "stop_sequence":
		return "stop"
	case "max_tokens":
		return "length"
	default:
		return "stop"
	}
}
