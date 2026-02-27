package llm

import (
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

func (c *Client) buildAnthropicProviderRequest(request CompletionRequest) (providerRequest, error) {
	body, err := toAnthropicRequest(c.opts.Model, c.opts.AnthropicMaxTokens, request)
	if err != nil {
		return providerRequest{}, err
	}

	headers := make(map[string]string, len(c.opts.Headers)+2)
	if c.opts.APIKey != "" {
		headers["x-api-key"] = c.opts.APIKey
	}
	headers["anthropic-version"] = c.opts.AnthropicVersion
	mergeStringHeaders(headers, c.opts.Headers)

	return providerRequest{
		path:    c.opts.ChatPath,
		body:    body,
		headers: headers,
	}, nil
}

func (c *Client) parseAnthropicProviderResponse(raw []byte) (*CompletionResponse, error) {
	var response anthropicResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("decode anthropic response: %w", err)
	}
	return anthropicToCompletionResponse(response)
}

// toAnthropicRequest 把统一请求转换为 Anthropic messages/tool_use 协议。
func toAnthropicRequest(model string, maxTokens int, request CompletionRequest) (anthropicRequest, error) {
	convertedMessages := make([]anthropicMessage, 0, len(request.Messages))
	systemParts := make([]string, 0, 2)

	for _, msg := range request.Messages {
		switch msg.Role {
		case RoleSystem:
			system := strings.TrimSpace(msg.Text)
			if system != "" {
				systemParts = append(systemParts, system)
			}
		case RoleUser:
			convertedMessages = append(convertedMessages, anthropicMessage{
				Role:    "user",
				Content: msg.Text,
			})
		case RoleAssistant:
			assistantMessage, err := toAnthropicAssistantMessage(msg)
			if err != nil {
				return anthropicRequest{}, err
			}
			convertedMessages = append(convertedMessages, assistantMessage)
		case RoleTool:
			convertedMessages = append(convertedMessages, anthropicMessage{
				Role: "user",
				Content: []anthropicContentBlock{
					{
						Type:      "tool_result",
						ToolUseID: msg.ToolCallID,
						Content:   msg.Text,
					},
				},
			})
		default:
			return anthropicRequest{}, fmt.Errorf("unsupported message role for anthropic: %q", msg.Role)
		}
	}

	convertedTools := make([]anthropicTool, 0, len(request.Tools))
	for _, tool := range request.Tools {
		schema, err := decodeJSONObject(tool.Parameters)
		if err != nil {
			return anthropicRequest{}, fmt.Errorf("decode schema for tool %q: %w", tool.Name, err)
		}
		convertedTools = append(convertedTools, anthropicTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: schema,
		})
	}

	out := anthropicRequest{
		Model:     model,
		MaxTokens: maxTokens,
		Messages:  convertedMessages,
		Tools:     convertedTools,
	}
	if len(systemParts) > 0 {
		out.System = strings.Join(systemParts, "\n\n")
	}

	return out, nil
}

// toAnthropicAssistantMessage 将 assistant 文本与 tool calls 编码为 content blocks。
func toAnthropicAssistantMessage(msg Message) (anthropicMessage, error) {
	blocks := make([]anthropicContentBlock, 0, len(msg.ToolCalls)+1)

	text := strings.TrimSpace(msg.Text)
	if text != "" {
		blocks = append(blocks, anthropicContentBlock{
			Type: "text",
			Text: text,
		})
	}

	for _, call := range msg.ToolCalls {
		input, err := decodeJSONObject(call.Arguments)
		if err != nil {
			return anthropicMessage{}, fmt.Errorf("decode arguments for tool %q: %w", call.Name, err)
		}
		blocks = append(blocks, anthropicContentBlock{
			Type:  "tool_use",
			ID:    call.ID,
			Name:  call.Name,
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

// decodeJSONObject 确保 schema/arguments 统一按 JSON object 解码。
func decodeJSONObject(raw json.RawMessage) (map[string]any, error) {
	normalized := normalizeJSONObject(raw)
	out := make(map[string]any)
	if err := json.Unmarshal(normalized, &out); err != nil {
		return nil, err
	}
	return out, nil
}
