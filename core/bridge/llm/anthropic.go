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

type anthropicStreamAccumulator struct {
	role       Role
	stopReason string
	usage      anthropicUsage
	blocks     map[int]*anthropicStreamBlockState
	blockOrder []int
}

type anthropicStreamBlockState struct {
	index   int
	kind    string
	text    strings.Builder
	input   strings.Builder
	id      string
	name    string
	started bool
	ended   bool
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

func (c *Client) streamAnthropicCompletion(ctx context.Context, request CompletionRequest, sink LLMStreamSink) (*CompletionResponse, error) {
	body, err := toAnthropicRequest(c.opts.Model, c.opts.AnthropicMaxTokens, request)
	if err != nil {
		return nil, err
	}
	body.Stream = true

	headers := make(map[string]string, len(c.opts.Headers)+2)
	if c.opts.APIKey != "" {
		headers["x-api-key"] = c.opts.APIKey
	}
	headers["anthropic-version"] = c.opts.AnthropicVersion
	mergeStringHeaders(headers, c.opts.Headers)

	accumulator := newAnthropicStreamAccumulator()
	if err := c.streamJSON(ctx, c.opts.ChatPath, body, headers, func(line []byte) error {
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

func (c *Client) processAnthropicStreamEvent(
	ctx context.Context,
	line []byte,
	sink LLMStreamSink,
	accumulator *anthropicStreamAccumulator,
) error {
	var event anthropicStreamEvent
	if err := json.Unmarshal(line, &event); err != nil {
		return fmt.Errorf("parse anthropic stream event: %w", err)
	}

	return accumulator.ApplyEvent(ctx, sink, event)
}

func newAnthropicStreamAccumulator() *anthropicStreamAccumulator {
	return &anthropicStreamAccumulator{
		role:   RoleAssistant,
		blocks: make(map[int]*anthropicStreamBlockState),
	}
}

func (a *anthropicStreamAccumulator) ApplyEvent(ctx context.Context, sink LLMStreamSink, event anthropicStreamEvent) error {
	switch strings.TrimSpace(event.Type) {
	case "ping":
		return nil
	case "message_start":
		if event.Message != nil {
			if role := strings.TrimSpace(event.Message.Role); role != "" {
				a.role = Role(role)
			}
			a.usage = event.Message.Usage
		}
	case "content_block_start":
		state := a.ensureBlockState(event.Index)
		if event.ContentBlock != nil {
			state.kind = strings.TrimSpace(event.ContentBlock.Type)
			state.id = strings.TrimSpace(event.ContentBlock.ID)
			state.name = strings.TrimSpace(event.ContentBlock.Name)
			if state.kind == "text" && strings.TrimSpace(event.ContentBlock.Text) != "" {
				state.text.WriteString(event.ContentBlock.Text)
				if err := sink.OnDelta(ctx, LLMDelta{
					Kind: DeltaKindText,
					Text: event.ContentBlock.Text,
				}); err != nil {
					return err
				}
			}
			if state.kind == "tool_use" && !state.started {
				if err := sink.OnDelta(ctx, LLMDelta{
					Kind:          DeltaKindToolCallStart,
					ToolCallIndex: event.Index,
					ToolCallID:    state.id,
					ToolName:      state.name,
				}); err != nil {
					return err
				}
				state.started = true
			}
		}
	case "content_block_delta":
		if event.Delta == nil {
			return nil
		}
		state := a.ensureBlockState(event.Index)
		switch strings.TrimSpace(event.Delta.Type) {
		case "text_delta":
			if event.Delta.Text == "" {
				return nil
			}
			state.kind = "text"
			state.text.WriteString(event.Delta.Text)
			if err := sink.OnDelta(ctx, LLMDelta{
				Kind: DeltaKindText,
				Text: event.Delta.Text,
			}); err != nil {
				return err
			}
		case "input_json_delta":
			state.kind = "tool_use"
			if !state.started {
				if err := sink.OnDelta(ctx, LLMDelta{
					Kind:          DeltaKindToolCallStart,
					ToolCallIndex: event.Index,
					ToolCallID:    state.id,
					ToolName:      state.name,
				}); err != nil {
					return err
				}
				state.started = true
			}
			if event.Delta.PartialJSON == "" {
				return nil
			}
			state.input.WriteString(event.Delta.PartialJSON)
			if err := sink.OnDelta(ctx, LLMDelta{
				Kind:              DeltaKindToolCallDelta,
				ToolCallIndex:     event.Index,
				ArgumentsFragment: event.Delta.PartialJSON,
			}); err != nil {
				return err
			}
		}
	case "content_block_stop":
		state := a.blocks[event.Index]
		if state != nil && state.kind == "tool_use" && !state.ended {
			if err := sink.OnDelta(ctx, LLMDelta{
				Kind:          DeltaKindToolCallEnd,
				ToolCallIndex: event.Index,
			}); err != nil {
				return err
			}
			state.ended = true
		}
	case "message_delta":
		if event.Delta != nil && strings.TrimSpace(event.Delta.StopReason) != "" {
			a.stopReason = strings.TrimSpace(event.Delta.StopReason)
		}
		if event.Usage.InputTokens > 0 || event.Usage.OutputTokens > 0 {
			if event.Usage.InputTokens > 0 {
				a.usage.InputTokens = event.Usage.InputTokens
			}
			if event.Usage.OutputTokens > 0 {
				a.usage.OutputTokens = event.Usage.OutputTokens
			}
		}
	case "message_stop":
		return nil
	}

	return nil
}

func (a *anthropicStreamAccumulator) CompletionResponse() (*CompletionResponse, error) {
	response := anthropicResponse{
		Role:       string(a.role),
		StopReason: a.stopReason,
		Usage:      a.usage,
	}
	response.Content = make([]anthropicContentBlock, 0, len(a.blockOrder))
	for _, index := range a.blockOrder {
		state := a.blocks[index]
		if state == nil {
			continue
		}
		switch state.kind {
		case "text":
			response.Content = append(response.Content, anthropicContentBlock{
				Type: "text",
				Text: state.text.String(),
			})
		case "tool_use":
			input := map[string]any{}
			if raw := strings.TrimSpace(state.input.String()); raw != "" {
				if err := json.Unmarshal([]byte(raw), &input); err != nil {
					return nil, fmt.Errorf("decode anthropic tool input for %q: %w", state.name, err)
				}
			}
			response.Content = append(response.Content, anthropicContentBlock{
				Type:  "tool_use",
				ID:    state.id,
				Name:  state.name,
				Input: input,
			})
		}
	}

	return anthropicToCompletionResponse(response)
}

func (a *anthropicStreamAccumulator) ensureBlockState(index int) *anthropicStreamBlockState {
	if state, ok := a.blocks[index]; ok {
		return state
	}
	state := &anthropicStreamBlockState{index: index}
	a.blocks[index] = state
	a.blockOrder = append(a.blockOrder, index)
	return state
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
