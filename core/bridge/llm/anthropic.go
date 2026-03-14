package llm

import (
	"context"
	"fmt"
	"strings"
)

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
	Stream    bool               `json:"stream,omitempty"`
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
	Type      string                `json:"type"`
	Text      string                `json:"text,omitempty"`
	Source    *anthropicImageSource `json:"source,omitempty"`
	ID        string                `json:"id,omitempty"`
	Name      string                `json:"name,omitempty"`
	Input     map[string]any        `json:"input,omitempty"`
	ToolUseID string                `json:"tool_use_id,omitempty"`
	Content   any                   `json:"content,omitempty"`
	IsError   bool                  `json:"is_error,omitempty"`
}

type anthropicImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type,omitempty"`
	Data      string `json:"data,omitempty"`
	URL       string `json:"url,omitempty"`
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

type anthropicStreamEvent struct {
	Type         string                 `json:"type"`
	Index        int                    `json:"index,omitempty"`
	Message      *anthropicResponse     `json:"message,omitempty"`
	ContentBlock *anthropicContentBlock `json:"content_block,omitempty"`
	Delta        *anthropicStreamDelta  `json:"delta,omitempty"`
	Usage        anthropicUsage         `json:"usage,omitempty"`
}

type anthropicStreamDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
	StopReason  string `json:"stop_reason,omitempty"`
}

func (c *Client) buildAnthropicProviderRequest(request CompletionRequest) (providerRequest, error) {
	body, err := toAnthropicRequest(c.opts.Model, c.opts.AnthropicMaxTokens, request)
	if err != nil {
		return providerRequest{}, err
	}

	defaults := map[string]string{
		"anthropic-version": c.opts.AnthropicVersion,
	}
	if apiKey := strings.TrimSpace(c.opts.APIKey); apiKey != "" {
		defaults["x-api-key"] = apiKey
	}

	return providerRequest{
		path:    c.opts.ChatPath,
		body:    body,
		headers: c.providerHeaders(defaults),
	}, nil
}

func (c *Client) streamAnthropicCompletion(ctx context.Context, request CompletionRequest, sink LLMStreamSink) (*CompletionResponse, error) {
	body, err := toAnthropicRequest(c.opts.Model, c.opts.AnthropicMaxTokens, request)
	if err != nil {
		return nil, err
	}
	body.Stream = true

	defaults := map[string]string{
		"anthropic-version": c.opts.AnthropicVersion,
	}
	if apiKey := strings.TrimSpace(c.opts.APIKey); apiKey != "" {
		defaults["x-api-key"] = apiKey
	}

	accumulator := newAnthropicStreamAccumulator()
	if err := c.streamJSON(ctx, c.opts.ChatPath, body, c.providerHeaders(defaults), func(line []byte) error {
		return c.processAnthropicStreamEvent(ctx, line, sink, accumulator)
	}); err != nil {
		return nil, err
	}

	return accumulator.CompletionResponse()
}

// toAnthropicRequest 把统一请求转换为 Anthropic messages/tool_use 协议。
func toAnthropicRequest(model string, maxTokens int, request CompletionRequest) (anthropicRequest, error) {
	if err := validateRequestMessageToolProtocol(request.Messages); err != nil {
		return anthropicRequest{}, err
	}

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
			content, err := toAnthropicToolResultContent(msg)
			if err != nil {
				return anthropicRequest{}, err
			}
			convertedMessages = append(convertedMessages, anthropicMessage{
				Role: "user",
				Content: []anthropicContentBlock{
					{
						Type:      "tool_result",
						ToolUseID: msg.ToolCallID,
						Content:   content,
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

func toAnthropicToolResultContent(msg Message) (any, error) {
	if len(msg.Content) == 0 {
		return msg.Text, nil
	}

	blocks := make([]anthropicContentBlock, 0, len(msg.Content)+1)
	text := strings.TrimSpace(msg.Text)
	if text != "" {
		blocks = append(blocks, anthropicContentBlock{
			Type: "text",
			Text: text,
		})
	}
	for _, part := range msg.Content {
		if part.Image == nil {
			continue
		}
		source, err := toAnthropicImageSource(part.Image)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, anthropicContentBlock{
			Type:   "image",
			Source: source,
		})
	}
	if len(blocks) == 0 {
		return msg.Text, nil
	}
	return blocks, nil
}

func toAnthropicImageSource(image *ImageContent) (*anthropicImageSource, error) {
	source, err := resolveImageSource(image)
	if err != nil {
		return nil, err
	}
	if source.URL != "" {
		return &anthropicImageSource{
			Type: "url",
			URL:  source.URL,
		}, nil
	}
	return &anthropicImageSource{
		Type:      "base64",
		MediaType: source.MediaType,
		Data:      source.Base64Data,
	}, nil
}
