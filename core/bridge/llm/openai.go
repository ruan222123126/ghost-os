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
	Role      string                 `json:"role,omitempty"`
	Content   string                 `json:"content,omitempty"`
	ToolCalls []openAIStreamToolCall `json:"tool_calls,omitempty"`
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

type openAIStreamAccumulator struct {
	message      Message
	finishReason string
	usage        Usage
	toolStates   map[int]*openAIStreamToolState
	toolOrder    []int
}

type openAIStreamToolState struct {
	index   int
	call    ToolCall
	args    strings.Builder
	started bool
	ended   bool
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

func (c *Client) streamOpenAICompletion(ctx context.Context, request CompletionRequest, sink LLMStreamSink) (*CompletionResponse, error) {
	body, err := toOpenAIRequest(c.opts.Model, request)
	if err != nil {
		return nil, err
	}
	body.Stream = true

	headers := make(map[string]string, len(c.opts.Headers)+1)
	if c.opts.APIKey != "" {
		headers["Authorization"] = "Bearer " + c.opts.APIKey
	}
	mergeStringHeaders(headers, c.opts.Headers)

	accumulator := newOpenAIStreamAccumulator()
	if err := c.streamJSON(ctx, c.opts.ChatPath, body, headers, func(line []byte) error {
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

func (c *Client) processOpenAIStreamChunk(
	ctx context.Context,
	line []byte,
	sink LLMStreamSink,
	accumulator *openAIStreamAccumulator,
) error {
	var chunk openAIStreamChunk
	if err := json.Unmarshal(line, &chunk); err != nil {
		return fmt.Errorf("parse openai stream chunk: %w", err)
	}
	if chunk.Usage != nil {
		accumulator.usage = Usage{
			PromptTokens:     chunk.Usage.PromptTokens,
			CompletionTokens: chunk.Usage.CompletionTokens,
			TotalTokens:      chunk.Usage.TotalTokens,
		}
	}
	if len(chunk.Choices) == 0 {
		return nil
	}

	choice := chunk.Choices[0]
	if err := accumulator.ApplyDelta(ctx, sink, choice.Delta); err != nil {
		return err
	}
	if choice.FinishReason != nil && strings.TrimSpace(*choice.FinishReason) != "" {
		return accumulator.SetFinishReason(ctx, sink, *choice.FinishReason)
	}
	return nil
}

func newOpenAIStreamAccumulator() *openAIStreamAccumulator {
	return &openAIStreamAccumulator{
		message: Message{
			Role: RoleAssistant,
		},
		toolStates: make(map[int]*openAIStreamToolState),
	}
}

func (a *openAIStreamAccumulator) ApplyDelta(ctx context.Context, sink LLMStreamSink, delta openAIStreamDelta) error {
	if role := strings.TrimSpace(delta.Role); role != "" {
		a.message.Role = Role(role)
	}
	if text := delta.Content; text != "" {
		a.message.Text += text
		if err := sink.OnDelta(ctx, LLMDelta{
			Kind: DeltaKindText,
			Text: text,
		}); err != nil {
			return err
		}
	}

	for _, toolDelta := range delta.ToolCalls {
		state := a.ensureToolState(toolDelta.Index)
		if id := strings.TrimSpace(toolDelta.ID); id != "" {
			state.call.ID = id
		}
		if name := strings.TrimSpace(toolDelta.Function.Name); name != "" {
			state.call.Name = name
		}
		if !state.started {
			if err := sink.OnDelta(ctx, LLMDelta{
				Kind:          DeltaKindToolCallStart,
				ToolCallIndex: state.index,
				ToolCallID:    state.call.ID,
				ToolName:      state.call.Name,
			}); err != nil {
				return err
			}
			state.started = true
		}
		if fragment := toolDelta.Function.Arguments; fragment != "" {
			state.args.WriteString(fragment)
			if err := sink.OnDelta(ctx, LLMDelta{
				Kind:              DeltaKindToolCallDelta,
				ToolCallIndex:     state.index,
				ArgumentsFragment: fragment,
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

func (a *openAIStreamAccumulator) SetFinishReason(ctx context.Context, sink LLMStreamSink, reason string) error {
	a.finishReason = strings.TrimSpace(reason)
	if a.finishReason == "tool_calls" {
		for _, index := range a.toolOrder {
			state := a.toolStates[index]
			if state == nil || state.ended {
				continue
			}
			if err := sink.OnDelta(ctx, LLMDelta{
				Kind:          DeltaKindToolCallEnd,
				ToolCallIndex: index,
			}); err != nil {
				return err
			}
			state.ended = true
		}
	}
	return nil
}

func (a *openAIStreamAccumulator) CompletionResponse() (*CompletionResponse, error) {
	if a.message.Role == "" {
		a.message.Role = RoleAssistant
	}
	if len(a.toolOrder) > 0 {
		a.message.ToolCalls = make([]ToolCall, 0, len(a.toolOrder))
		for _, index := range a.toolOrder {
			state := a.toolStates[index]
			if state == nil {
				continue
			}
			state.call.Arguments = json.RawMessage(state.args.String())
			a.message.ToolCalls = append(a.message.ToolCalls, ToolCall{
				ID:        state.call.ID,
				Name:      state.call.Name,
				Arguments: cloneRawJSON(state.call.Arguments),
			})
		}
	}
	finishReason, err := openAIFinishReason(normalizeOpenAIFinishReason(a.finishReason, a.message))
	if err != nil {
		return nil, err
	}
	return &CompletionResponse{
		Message:      a.message,
		FinishReason: finishReason,
		Usage:        a.usage,
	}, nil
}

func (a *openAIStreamAccumulator) ensureToolState(index int) *openAIStreamToolState {
	if state, ok := a.toolStates[index]; ok {
		return state
	}
	state := &openAIStreamToolState{index: index}
	a.toolStates[index] = state
	a.toolOrder = append(a.toolOrder, index)
	return state
}
