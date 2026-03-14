package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type codexToolEnvelope struct {
	Status string `json:"status"`
	Output string `json:"output"`
	Error  string `json:"error"`
}

type codexRequest struct {
	Model              string           `json:"model"`
	Instructions       string           `json:"instructions,omitempty"`
	Input              []codexInputItem `json:"input,omitempty"`
	Tools              []codexTool      `json:"tools,omitempty"`
	ToolChoice         string           `json:"tool_choice"`
	ParallelToolCalls  bool             `json:"parallel_tool_calls"`
	PreviousResponseID string           `json:"previous_response_id,omitempty"`
	Stream             bool             `json:"stream,omitempty"`
}

type codexInputItem struct {
	Type      string              `json:"type,omitempty"`
	Role      string              `json:"role,omitempty"`
	Content   []codexInputContent `json:"content,omitempty"`
	CallID    string              `json:"call_id,omitempty"`
	Name      string              `json:"name,omitempty"`
	Arguments string              `json:"arguments,omitempty"`
	Output    string              `json:"output,omitempty"`
}

type codexInputContent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

type codexTool struct {
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Strict      bool           `json:"strict"`
	Parameters  map[string]any `json:"parameters"`
}

type codexResponse struct {
	ID                string                  `json:"id"`
	Status            string                  `json:"status"`
	Output            []codexOutputItem       `json:"output"`
	Usage             *codexUsage             `json:"usage,omitempty"`
	IncompleteDetails *codexIncompleteDetails `json:"incomplete_details,omitempty"`
}

type codexUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type codexIncompleteDetails struct {
	Reason string `json:"reason"`
}

type codexOutputItem struct {
	ID        string               `json:"id,omitempty"`
	Type      string               `json:"type"`
	Role      string               `json:"role,omitempty"`
	Content   []codexOutputContent `json:"content,omitempty"`
	Name      string               `json:"name,omitempty"`
	Arguments string               `json:"arguments,omitempty"`
	CallID    string               `json:"call_id,omitempty"`
	Status    string               `json:"status,omitempty"`
}

type codexOutputContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type codexStreamEvent struct {
	Type        string           `json:"type"`
	OutputIndex int              `json:"output_index,omitempty"`
	Item        *codexOutputItem `json:"item,omitempty"`
	Delta       string           `json:"delta,omitempty"`
	Response    *codexResponse   `json:"response,omitempty"`
}

type codexStreamAccumulator struct {
	message      Message
	finishReason FinishReason
	usage        Usage
	toolStates   map[int]*codexStreamToolState
	toolOrder    []int
	final        *CompletionResponse
}

type codexStreamToolState struct {
	index   int
	call    ToolCall
	args    strings.Builder
	started bool
	ended   bool
}

type codexRequestOptions struct {
	ForceStateless bool
}

func (c *Client) buildCodexProviderRequest(request CompletionRequest) (providerRequest, error) {
	body, err := toCodexRequest(c.opts.Model, request)
	if err != nil {
		return providerRequest{}, err
	}
	return c.codexProviderRequestFromBody(body), nil
}

func (c *Client) buildCodexProviderRequestStateless(request CompletionRequest) (providerRequest, error) {
	body, err := toCodexRequestWithOptions(c.opts.Model, request, codexRequestOptions{ForceStateless: true})
	if err != nil {
		return providerRequest{}, err
	}
	return c.codexProviderRequestFromBody(body), nil
}

func (c *Client) codexProviderRequestFromBody(body codexRequest) providerRequest {
	headers := make(map[string]string, len(c.opts.Headers)+1)
	if c.opts.APIKey != "" {
		headers["Authorization"] = "Bearer " + c.opts.APIKey
	}
	mergeStringHeaders(headers, c.opts.Headers)

	return providerRequest{
		path:    c.opts.ChatPath,
		body:    body,
		headers: headers,
	}
}

func (c *Client) parseCodexProviderResponse(raw []byte) (*CompletionResponse, error) {
	var response codexResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("decode codex response: %w", err)
	}
	completion, err := codexToCompletionResponse(response)
	if err != nil {
		return nil, err
	}
	c.stampCodexConversationState(completion)
	return completion, nil
}

func (c *Client) streamCodexCompletion(ctx context.Context, request CompletionRequest, sink LLMStreamSink) (*CompletionResponse, error) {
	return c.streamCodexCompletionWithOptions(ctx, request, sink, codexRequestOptions{})
}

func (c *Client) streamCodexCompletionStateless(ctx context.Context, request CompletionRequest, sink LLMStreamSink) (*CompletionResponse, error) {
	return c.streamCodexCompletionWithOptions(ctx, request, sink, codexRequestOptions{ForceStateless: true})
}

func (c *Client) streamCodexCompletionWithOptions(ctx context.Context, request CompletionRequest, sink LLMStreamSink, options codexRequestOptions) (*CompletionResponse, error) {
	body, err := toCodexRequestWithOptions(c.opts.Model, request, options)
	if err != nil {
		return nil, err
	}
	body.Stream = true

	payload := c.codexProviderRequestFromBody(body)
	accumulator := newCodexStreamAccumulator()
	if err := c.streamJSON(ctx, payload.path, payload.body, payload.headers, func(line []byte) error {
		return c.processCodexStreamEvent(ctx, line, sink, accumulator)
	}); err != nil {
		return nil, err
	}

	completion, err := accumulator.CompletionResponse()
	if err != nil {
		return nil, err
	}
	c.stampCodexConversationState(completion)
	return completion, nil
}

func toCodexRequest(model string, request CompletionRequest) (codexRequest, error) {
	return toCodexRequestWithOptions(model, request, codexRequestOptions{})
}

func toCodexRequestWithOptions(model string, request CompletionRequest, options codexRequestOptions) (codexRequest, error) {
	instructions := codexInstructions(request.Messages)
	messages := request.Messages
	previousResponseID := ""
	if !options.ForceStateless {
		previousResponseID = strings.TrimSpace(request.ConversationState.PreviousResponseID)
	}
	if previousResponseID != "" {
		messages = codexIncrementalMessages(messages)
	}

	var (
		input []codexInputItem
		err   error
	)
	if options.ForceStateless {
		input, err = codexFallbackInput(messages)
		if err != nil {
			return codexRequest{}, err
		}
	} else {
		input, err = codexMessagesToInput(messages)
		if err != nil {
			return codexRequest{}, err
		}
	}

	tools := make([]codexTool, 0, len(request.Tools))
	for _, tool := range request.Tools {
		parameters, err := decodeJSONObject(tool.Parameters)
		if err != nil {
			return codexRequest{}, fmt.Errorf("decode schema for tool %q: %w", tool.Name, err)
		}
		sanitizeCodexToolSchema(parameters)
		tools = append(tools, codexTool{
			Type:        "function",
			Name:        tool.Name,
			Description: tool.Description,
			Strict:      false,
			Parameters:  parameters,
		})
	}

	return codexRequest{
		Model:              model,
		Instructions:       instructions,
		Input:              input,
		Tools:              tools,
		ToolChoice:         "auto",
		ParallelToolCalls:  false,
		PreviousResponseID: previousResponseID,
	}, nil
}

func codexMessagesToInput(messages []Message) ([]codexInputItem, error) {
	input := make([]codexInputItem, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case RoleSystem:
			continue
		case RoleUser:
			item, ok, err := toCodexMessageInput(msg)
			if err != nil {
				return nil, err
			}
			if ok {
				input = append(input, item)
			}
		case RoleAssistant:
			if len(msg.ToolCalls) == 0 {
				item, ok, err := toCodexMessageInput(msg)
				if err != nil {
					return nil, err
				}
				if ok {
					input = append(input, item)
				}
			}
			if msg.Role == RoleAssistant {
				for _, call := range msg.ToolCalls {
					input = append(input, codexInputItem{
						Type:      "function_call",
						CallID:    strings.TrimSpace(call.ID),
						Name:      strings.TrimSpace(call.Name),
						Arguments: string(normalizeJSONObject(call.Arguments)),
					})
				}
			}
		case RoleTool:
			input = append(input, codexInputItem{
				Type:   "function_call_output",
				CallID: strings.TrimSpace(msg.ToolCallID),
				Output: toCodexToolOutput(msg),
			})
		default:
			return nil, fmt.Errorf("unsupported message role for codex provider: %q", msg.Role)
		}
	}
	return input, nil
}

func codexFallbackInput(messages []Message) ([]codexInputItem, error) {
	lastUserIndex := -1
	for index := len(messages) - 1; index >= 0; index-- {
		if messages[index].Role == RoleUser {
			lastUserIndex = index
			break
		}
	}
	if lastUserIndex < 0 {
		return codexMessagesToInput(messages)
	}

	input := make([]codexInputItem, 0, len(messages)-lastUserIndex)
	if userItem, ok, err := codexFallbackUserItem(messages[:lastUserIndex], messages[lastUserIndex]); err != nil {
		return nil, err
	} else if ok {
		input = append(input, userItem)
	}

	for _, msg := range messages[lastUserIndex+1:] {
		switch msg.Role {
		case RoleSystem, RoleUser:
			continue
		case RoleAssistant:
			for _, call := range msg.ToolCalls {
				input = append(input, codexInputItem{
					Type:      "function_call",
					CallID:    strings.TrimSpace(call.ID),
					Name:      strings.TrimSpace(call.Name),
					Arguments: string(normalizeJSONObject(call.Arguments)),
				})
			}
		case RoleTool:
			input = append(input, codexInputItem{
				Type:   "function_call_output",
				CallID: strings.TrimSpace(msg.ToolCallID),
				Output: toCodexToolOutput(msg),
			})
		default:
			return nil, fmt.Errorf("unsupported message role for codex provider: %q", msg.Role)
		}
	}

	return input, nil
}

func codexFallbackUserItem(prior []Message, current Message) (codexInputItem, bool, error) {
	currentText := strings.TrimSpace(codexMessageText(current))
	if currentText == "" {
		item, ok, err := toCodexMessageInput(current)
		if err != nil || !ok {
			return codexInputItem{}, ok, err
		}
		return item, true, nil
	}

	transcript := codexConversationTranscript(prior)
	text := currentText
	if transcript != "" {
		text = "Conversation so far:\n" + transcript + "\n\nCurrent request: " + currentText
	}

	return codexInputItem{
		Type: "message",
		Role: "user",
		Content: []codexInputContent{
			{Type: "input_text", Text: text},
		},
	}, true, nil
}

func codexConversationTranscript(messages []Message) string {
	lines := make([]string, 0, len(messages))
	for _, msg := range messages {
		text := strings.TrimSpace(codexMessageText(msg))
		if text == "" {
			continue
		}
		switch msg.Role {
		case RoleUser:
			lines = append(lines, "User: "+text)
		case RoleAssistant:
			if len(msg.ToolCalls) > 0 {
				continue
			}
			lines = append(lines, "Assistant: "+text)
		}
	}
	return strings.Join(lines, "\n")
}

func codexMessageText(msg Message) string {
	parts := make([]string, 0, len(msg.Content)+1)
	if text := strings.TrimSpace(msg.Text); text != "" {
		parts = append(parts, text)
	}
	for _, part := range msg.Content {
		if strings.EqualFold(strings.TrimSpace(part.Type), ContentTypeText) || strings.TrimSpace(part.Type) == "" {
			if text := strings.TrimSpace(part.Text); text != "" {
				parts = append(parts, text)
			}
		}
	}
	return strings.Join(parts, "\n")
}

func codexIncrementalMessages(messages []Message) []Message {
	if len(messages) == 0 {
		return nil
	}

	lastAssistantIndex := -1
	for index, msg := range messages {
		if msg.Role == RoleAssistant {
			lastAssistantIndex = index
		}
	}

	start := lastAssistantIndex + 1
	if lastAssistantIndex < 0 {
		start = 0
	}
	out := make([]Message, 0, len(messages)-start)
	if start < len(messages) {
		out = append(out, CloneMessages(messages[start:])...)
	}
	return out
}

func codexInstructions(messages []Message) string {
	parts := make([]string, 0, 2)
	for _, msg := range messages {
		if msg.Role != RoleSystem {
			continue
		}
		if text := strings.TrimSpace(msg.Text); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n\n")
}

func toCodexMessageInput(msg Message) (codexInputItem, bool, error) {
	content, err := toCodexInputContent(msg)
	if err != nil {
		return codexInputItem{}, false, err
	}
	if len(content) == 0 {
		return codexInputItem{}, false, nil
	}
	return codexInputItem{
		Type:    "message",
		Role:    string(msg.Role),
		Content: content,
	}, true, nil
}

func toCodexInputContent(msg Message) ([]codexInputContent, error) {
	parts := make([]codexInputContent, 0, len(msg.Content)+1)
	if text := strings.TrimSpace(msg.Text); text != "" {
		parts = append(parts, codexInputContent{
			Type: "input_text",
			Text: text,
		})
	}
	for _, part := range msg.Content {
		switch strings.ToLower(strings.TrimSpace(part.Type)) {
		case "", ContentTypeText:
			if text := strings.TrimSpace(part.Text); text != "" {
				parts = append(parts, codexInputContent{
					Type: "input_text",
					Text: text,
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
			parts = append(parts, codexInputContent{
				Type:     "input_image",
				ImageURL: imageURL,
			})
		}
	}
	return parts, nil
}

func toCodexToolOutput(msg Message) string {
	baseText := strings.TrimSpace(msg.Text)
	if parsed, ok := parseCodexToolEnvelope(baseText); ok {
		if parsed.Status == "error" && strings.TrimSpace(parsed.Error) != "" {
			baseText = parsed.Error
		} else if strings.TrimSpace(parsed.Output) != "" {
			baseText = parsed.Output
		} else {
			baseText = ""
		}
	}

	if len(msg.Content) == 0 {
		return baseText
	}

	type toolContentPart struct {
		Type      string `json:"type"`
		Text      string `json:"text,omitempty"`
		ImageURL  string `json:"image_url,omitempty"`
		Path      string `json:"path,omitempty"`
		MimeType  string `json:"mime_type,omitempty"`
		SHA256    string `json:"sha256,omitempty"`
		ByteCount int    `json:"bytes,omitempty"`
	}
	payload := struct {
		Text    string            `json:"text,omitempty"`
		Content []toolContentPart `json:"content,omitempty"`
	}{
		Text: baseText,
	}
	for _, part := range msg.Content {
		entry := toolContentPart{Type: strings.TrimSpace(part.Type), Text: strings.TrimSpace(part.Text)}
		if part.Image != nil {
			entry.Path = strings.TrimSpace(part.Image.Path)
			entry.ImageURL = strings.TrimSpace(part.Image.URL)
			entry.MimeType = strings.TrimSpace(part.Image.MimeType)
			entry.SHA256 = strings.TrimSpace(part.Image.SHA256)
			entry.ByteCount = part.Image.Bytes
		}
		payload.Content = append(payload.Content, entry)
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return baseText
	}
	return string(encoded)
}

func parseCodexToolEnvelope(raw string) (codexToolEnvelope, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return codexToolEnvelope{}, false
	}
	var envelope codexToolEnvelope
	if err := json.Unmarshal([]byte(trimmed), &envelope); err != nil {
		return codexToolEnvelope{}, false
	}
	if strings.TrimSpace(envelope.Status) == "" {
		return codexToolEnvelope{}, false
	}
	return envelope, true
}

func sanitizeCodexToolSchema(schema map[string]any) {
	for key, value := range schema {
		schema[key] = sanitizeCodexToolSchemaValue(value)
	}

	ty := codexSchemaType(schema)
	if ty == "" {
		ty = inferCodexSchemaType(schema)
	}
	if ty == "" {
		ty = "string"
	}
	if ty == "integer" {
		ty = "number"
	}
	schema["type"] = ty

	if ty == "object" {
		if _, ok := schema["properties"].(map[string]any); !ok {
			schema["properties"] = map[string]any{}
		}
		if additionalProperties, ok := schema["additionalProperties"]; ok {
			if _, isBool := additionalProperties.(bool); !isBool {
				schema["additionalProperties"] = sanitizeCodexToolSchemaValue(additionalProperties)
			}
		}
	}
	if ty == "array" {
		if _, ok := schema["items"]; !ok {
			schema["items"] = map[string]any{"type": "string"}
		}
	}
}

func sanitizeCodexToolSchemaValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		sanitizeCodexToolSchema(typed)
		return typed
	case []any:
		for index, item := range typed {
			typed[index] = sanitizeCodexToolSchemaValue(item)
		}
		return typed
	default:
		return value
	}
}

func codexSchemaType(schema map[string]any) string {
	if raw, ok := schema["type"]; ok {
		switch typed := raw.(type) {
		case string:
			return normalizeCodexSchemaType(typed)
		case []any:
			for _, candidate := range typed {
				if asString, ok := candidate.(string); ok {
					normalized := normalizeCodexSchemaType(asString)
					if normalized != "" {
						return normalized
					}
				}
			}
		}
	}
	return ""
}

func inferCodexSchemaType(schema map[string]any) string {
	switch {
	case schema["properties"] != nil || schema["required"] != nil || schema["additionalProperties"] != nil:
		return "object"
	case schema["items"] != nil || schema["prefixItems"] != nil:
		return "array"
	case schema["enum"] != nil || schema["const"] != nil || schema["format"] != nil:
		return "string"
	case schema["minimum"] != nil || schema["maximum"] != nil || schema["exclusiveMinimum"] != nil || schema["exclusiveMaximum"] != nil || schema["multipleOf"] != nil:
		return "number"
	default:
		return ""
	}
}

func normalizeCodexSchemaType(raw string) string {
	switch strings.TrimSpace(raw) {
	case "object", "array", "string", "number", "integer", "boolean":
		return strings.TrimSpace(raw)
	default:
		return ""
	}
}

func codexToCompletionResponse(response codexResponse) (*CompletionResponse, error) {
	message := Message{Role: RoleAssistant}
	textParts := make([]string, 0, 1)
	toolCalls := make([]ToolCall, 0, 1)
	for _, item := range response.Output {
		switch strings.TrimSpace(item.Type) {
		case "message":
			for _, content := range item.Content {
				switch strings.TrimSpace(content.Type) {
				case "output_text", "text":
					if strings.TrimSpace(content.Text) != "" {
						textParts = append(textParts, content.Text)
					}
				}
			}
		case "function_call":
			toolCalls = append(toolCalls, ToolCall{
				ID:        codexCallID(item),
				Name:      strings.TrimSpace(item.Name),
				Arguments: json.RawMessage(item.Arguments),
			})
		}
	}
	if len(textParts) > 0 {
		message.Text = strings.Join(textParts, "\n")
	}
	if len(toolCalls) > 0 {
		message.ToolCalls = toolCalls
	}

	finishReason := FinishStop
	if len(toolCalls) > 0 {
		finishReason = FinishToolCalls
	} else if codexIncompleteByLength(response) {
		finishReason = FinishLength
	}

	out := &CompletionResponse{
		Message:      message,
		FinishReason: finishReason,
		ConversationState: ConversationState{
			Provider:           ProviderCodex,
			PreviousResponseID: strings.TrimSpace(response.ID),
		},
	}
	if response.Usage != nil {
		out.Usage = Usage{
			PromptTokens:     response.Usage.InputTokens,
			CompletionTokens: response.Usage.OutputTokens,
			TotalTokens:      response.Usage.TotalTokens,
		}
	}
	return out, nil
}

func codexIncompleteByLength(response codexResponse) bool {
	if strings.TrimSpace(response.Status) != "incomplete" || response.IncompleteDetails == nil {
		return false
	}
	switch strings.TrimSpace(response.IncompleteDetails.Reason) {
	case "max_output_tokens", "max_completion_tokens":
		return true
	default:
		return false
	}
}

func codexCallID(item codexOutputItem) string {
	if value := strings.TrimSpace(item.CallID); value != "" {
		return value
	}
	return strings.TrimSpace(item.ID)
}

func (c *Client) processCodexStreamEvent(
	ctx context.Context,
	line []byte,
	sink LLMStreamSink,
	accumulator *codexStreamAccumulator,
) error {
	var event codexStreamEvent
	if err := json.Unmarshal(line, &event); err != nil {
		return fmt.Errorf("parse codex stream event: %w", err)
	}
	return accumulator.ApplyEvent(ctx, sink, event)
}

func newCodexStreamAccumulator() *codexStreamAccumulator {
	return &codexStreamAccumulator{
		message:    Message{Role: RoleAssistant},
		toolStates: make(map[int]*codexStreamToolState),
	}
}

func (a *codexStreamAccumulator) ApplyEvent(ctx context.Context, sink LLMStreamSink, event codexStreamEvent) error {
	switch strings.TrimSpace(event.Type) {
	case "response.created", "response.in_progress":
		return nil
	case "response.output_text.delta":
		if event.Delta == "" {
			return nil
		}
		a.message.Text += event.Delta
		return sink.OnDelta(ctx, LLMDelta{
			Kind: DeltaKindText,
			Text: event.Delta,
		})
	case "response.function_call_arguments.delta":
		state := a.ensureToolState(event.OutputIndex)
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
		if event.Delta == "" {
			return nil
		}
		state.args.WriteString(event.Delta)
		return sink.OnDelta(ctx, LLMDelta{
			Kind:              DeltaKindToolCallDelta,
			ToolCallIndex:     state.index,
			ArgumentsFragment: event.Delta,
		})
	case "response.output_item.added":
		if event.Item == nil || strings.TrimSpace(event.Item.Type) != "function_call" {
			return nil
		}
		state := a.ensureToolState(event.OutputIndex)
		a.updateToolState(state, *event.Item)
		if state.started {
			return nil
		}
		if err := sink.OnDelta(ctx, LLMDelta{
			Kind:          DeltaKindToolCallStart,
			ToolCallIndex: state.index,
			ToolCallID:    state.call.ID,
			ToolName:      state.call.Name,
		}); err != nil {
			return err
		}
		state.started = true
		return nil
	case "response.output_item.done":
		if event.Item == nil {
			return nil
		}
		if strings.TrimSpace(event.Item.Type) != "function_call" {
			return nil
		}
		state := a.ensureToolState(event.OutputIndex)
		a.updateToolState(state, *event.Item)
		if state.args.Len() == 0 && strings.TrimSpace(event.Item.Arguments) != "" {
			state.args.WriteString(event.Item.Arguments)
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
		if state.ended {
			return nil
		}
		if err := sink.OnDelta(ctx, LLMDelta{
			Kind:          DeltaKindToolCallEnd,
			ToolCallIndex: state.index,
		}); err != nil {
			return err
		}
		state.ended = true
		return nil
	case "response.completed":
		if event.Response == nil {
			return nil
		}
		resp, err := codexToCompletionResponse(*event.Response)
		if err != nil {
			return err
		}
		a.final = resp
		a.usage = resp.Usage
		a.finishReason = resp.FinishReason
		return nil
	default:
		return nil
	}
}

func (a *codexStreamAccumulator) CompletionResponse() (*CompletionResponse, error) {
	if a.final != nil {
		return a.final, nil
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
	if len(a.message.ToolCalls) > 0 {
		a.finishReason = FinishToolCalls
	} else if a.finishReason == "" {
		a.finishReason = FinishStop
	}
	return &CompletionResponse{
		Message:      a.message,
		FinishReason: a.finishReason,
		Usage:        a.usage,
	}, nil
}

func (c *Client) stampCodexConversationState(response *CompletionResponse) {
	if response == nil {
		return
	}
	response.ConversationState.Provider = ProviderCodex
	response.ConversationState.BaseURL = strings.TrimSpace(c.opts.BaseURL)
	response.ConversationState.Model = strings.TrimSpace(c.opts.Model)
}

func (a *codexStreamAccumulator) ensureToolState(index int) *codexStreamToolState {
	if state, ok := a.toolStates[index]; ok {
		return state
	}
	state := &codexStreamToolState{index: index}
	a.toolStates[index] = state
	a.toolOrder = append(a.toolOrder, index)
	return state
}

func (a *codexStreamAccumulator) updateToolState(state *codexStreamToolState, item codexOutputItem) {
	if state == nil {
		return
	}
	if id := codexCallID(item); id != "" {
		state.call.ID = id
	}
	if name := strings.TrimSpace(item.Name); name != "" {
		state.call.Name = name
	}
}
