package llm

import (
	"context"
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
	Thinking  string                `json:"thinking,omitempty"`
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
	Thinking    string `json:"thinking,omitempty"`
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
