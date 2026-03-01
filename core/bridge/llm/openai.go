package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
	Tools    []openAITool    `json:"tools,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    any              `json:"content"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAIContentPart struct {
	Type     string           `json:"type"`
	Text     string           `json:"text,omitempty"`
	ImageURL *openAIImagePart `json:"image_url,omitempty"`
}

type openAIImagePart struct {
	URL string `json:"url"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
}

type openAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openAIFunctionCall `json:"function"`
}

type openAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIResponse struct {
	ID      string         `json:"id"`
	Choices []openAIChoice `json:"choices"`
	Usage   openAIUsage    `json:"usage"`
}

type openAIChoice struct {
	Index        int           `json:"index"`
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func (c *Client) buildOpenAIProviderRequest(request CompletionRequest) (providerRequest, error) {
	body, err := toOpenAIRequest(c.opts.Model, request)
	if err != nil {
		return providerRequest{}, err
	}

	headers := make(map[string]string, len(c.opts.Headers)+1)
	if c.opts.APIKey != "" {
		headers["Authorization"] = "Bearer " + c.opts.APIKey
	}
	mergeStringHeaders(headers, c.opts.Headers)

	return providerRequest{
		path:    c.opts.ChatPath,
		body:    body,
		headers: headers,
	}, nil
}

func (c *Client) parseOpenAIProviderResponse(raw []byte) (*CompletionResponse, error) {
	var response openAIResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("decode openai-compatible response: %w", err)
	}

	return openAIToCompletionResponse(response)
}

// toOpenAIRequest 把内部统一请求模型转换为 OpenAI 兼容协议。
func toOpenAIRequest(model string, request CompletionRequest) (openAIRequest, error) {
	messages := make([]openAIMessage, 0, len(request.Messages))
	for _, msg := range request.Messages {
		converted, err := toOpenAIMessage(msg)
		if err != nil {
			return openAIRequest{}, err
		}
		messages = append(messages, converted)
	}

	tools := make([]openAITool, 0, len(request.Tools))
	for _, tool := range request.Tools {
		tools = append(tools, openAITool{
			Type: "function",
			Function: openAIFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  normalizeJSONObject(tool.Parameters),
			},
		})
	}

	return openAIRequest{
		Model:    model,
		Messages: messages,
		Tools:    tools,
	}, nil
}

// toOpenAIMessage 负责 Role 与 tool_calls 的协议层映射。
func toOpenAIMessage(msg Message) (openAIMessage, error) {
	out := openAIMessage{
		Role: string(msg.Role),
	}

	switch msg.Role {
	case RoleSystem, RoleUser:
		out.Content = msg.Text
	case RoleAssistant:
		if strings.TrimSpace(msg.Text) != "" {
			out.Content = msg.Text
		}
		if len(msg.ToolCalls) > 0 {
			toolCalls := make([]openAIToolCall, 0, len(msg.ToolCalls))
			for _, call := range msg.ToolCalls {
				toolCalls = append(toolCalls, openAIToolCall{
					ID:   call.ID,
					Type: "function",
					Function: openAIFunctionCall{
						Name:      call.Name,
						Arguments: string(normalizeJSONObject(call.Arguments)),
					},
				})
			}
			out.ToolCalls = toolCalls
		}
	case RoleTool:
		content, err := toOpenAIToolContent(msg.Text, msg.Content)
		if err != nil {
			return openAIMessage{}, err
		}
		out.Content = content
		out.ToolCallID = msg.ToolCallID
	default:
		return openAIMessage{}, fmt.Errorf("unsupported message role for openai-compatible provider: %q", msg.Role)
	}

	return out, nil
}

func toOpenAIToolContent(text string, content []ContentPart) (any, error) {
	if len(content) == 0 {
		return text, nil
	}

	parts := make([]openAIContentPart, 0, len(content)+1)
	text = strings.TrimSpace(text)
	if text != "" {
		parts = append(parts, openAIContentPart{
			Type: "text",
			Text: text,
		})
	}
	for _, part := range content {
		if part.Image == nil {
			continue
		}
		imageURL, err := resolveOpenAIImageURL(part.Image)
		if err != nil {
			return nil, err
		}
		parts = append(parts, openAIContentPart{
			Type: "image_url",
			ImageURL: &openAIImagePart{
				URL: imageURL,
			},
		})
	}

	if len(parts) == 0 {
		return text, nil
	}
	return parts, nil
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
