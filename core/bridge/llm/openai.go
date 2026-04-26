package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
	Tools    []openAITool    `json:"tools,omitempty"`
	Stream   bool            `json:"stream,omitempty"`
}

type openAIMessage struct {
	Role             string           `json:"role"`
	Content          any              `json:"content"`
	ReasoningContent json.RawMessage  `json:"reasoning_content,omitempty"`
	ToolCalls        []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string           `json:"tool_call_id,omitempty"`
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

type openAIStreamChunk struct {
	ID      string               `json:"id"`
	Choices []openAIStreamChoice `json:"choices"`
	Usage   *openAIUsage         `json:"usage,omitempty"`
}

type openAIStreamChoice struct {
	Index        int               `json:"index"`
	Delta        openAIStreamDelta `json:"delta"`
	FinishReason *string           `json:"finish_reason"`
}

type openAIStreamDelta struct {
	Role             string                 `json:"role,omitempty"`
	Content          string                 `json:"content,omitempty"`
	ReasoningContent any                    `json:"reasoning_content,omitempty"`
	Reasoning        any                    `json:"reasoning,omitempty"`
	ToolCalls        []openAIStreamToolCall `json:"tool_calls,omitempty"`
}

type openAIStreamToolCall struct {
	Index    int                      `json:"index"`
	ID       string                   `json:"id,omitempty"`
	Type     string                   `json:"type,omitempty"`
	Function openAIStreamFunctionCall `json:"function"`
}

type openAIStreamFunctionCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

func (c *Client) buildOpenAIProviderRequest(request CompletionRequest) (providerRequest, error) {
	body, err := toOpenAIRequest(c.opts.Model, request)
	if err != nil {
		return providerRequest{}, err
	}

	defaults := map[string]string{}
	if apiKey := strings.TrimSpace(c.opts.APIKey); apiKey != "" {
		defaults["Authorization"] = "Bearer " + apiKey
	}

	return providerRequest{
		path:    c.opts.ChatPath,
		body:    body,
		headers: c.providerHeaders(defaults),
	}, nil
}

func (c *Client) streamOpenAICompletion(ctx context.Context, request CompletionRequest, sink LLMStreamSink) (*CompletionResponse, error) {
	body, err := toOpenAIRequest(c.opts.Model, request)
	if err != nil {
		return nil, err
	}
	body.Stream = true

	defaults := map[string]string{}
	if apiKey := strings.TrimSpace(c.opts.APIKey); apiKey != "" {
		defaults["Authorization"] = "Bearer " + apiKey
	}

	accumulator := newOpenAIStreamAccumulator()
	if err := c.streamJSON(ctx, c.opts.ChatPath, body, c.providerHeaders(defaults), func(line []byte) error {
		return c.processOpenAIStreamChunk(ctx, line, sink, accumulator)
	}); err != nil {
		return nil, err
	}

	return accumulator.CompletionResponse()
}

// toOpenAIRequest 把内部统一请求模型转换为 OpenAI 兼容协议。
func toOpenAIRequest(model string, request CompletionRequest) (openAIRequest, error) {
	if err := validateRequestMessageToolProtocol(request.Messages); err != nil {
		return openAIRequest{}, err
	}

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
	case RoleSystem:
		out.Content = msg.Text
	case RoleUser:
		content, err := toOpenAIContent(msg.Text, msg.Content)
		if err != nil {
			return openAIMessage{}, err
		}
		out.Content = content
	case RoleAssistant:
		reasoningContent, err := normalizeOpenAIReasoningContent(msg.ReasoningContent)
		if err != nil {
			return openAIMessage{}, err
		}
		out.ReasoningContent = reasoningContent
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
		content, err := toOpenAIContent(msg.Text, msg.Content)
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

func normalizeOpenAIReasoningContent(raw json.RawMessage) (json.RawMessage, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}
	if !json.Valid([]byte(trimmed)) {
		return nil, fmt.Errorf("assistant reasoning_content must be valid JSON")
	}
	return cloneRawJSON(json.RawMessage(trimmed)), nil
}

func toOpenAIContent(text string, content []ContentPart) (any, error) {
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
		switch strings.ToLower(strings.TrimSpace(part.Type)) {
		case "", ContentTypeText:
			if value := strings.TrimSpace(part.Text); value != "" {
				parts = append(parts, openAIContentPart{
					Type: "text",
					Text: value,
				})
			}
		case ContentTypeImage:
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
	}

	if len(parts) == 0 {
		return text, nil
	}
	return parts, nil
}
