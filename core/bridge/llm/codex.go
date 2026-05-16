package llm

import (
	"context"
	"fmt"
	"strings"
)

type codexToolEnvelope struct {
	Status string `json:"status"`
	Output string `json:"output"`
	Error  string `json:"error"`
}

type codexRequest struct {
	Model                string            `json:"model"`
	Instructions         string            `json:"instructions,omitempty"`
	Input                []codexInputItem  `json:"input,omitempty"`
	Tools                []codexTool       `json:"tools,omitempty"`
	ToolChoice           string            `json:"tool_choice"`
	ParallelToolCalls    bool              `json:"parallel_tool_calls"`
	PromptCacheKey       string            `json:"prompt_cache_key,omitempty"`
	PromptCacheRetention string            `json:"prompt_cache_retention,omitempty"`
	SafetyIdentifier     string            `json:"safety_identifier,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
	Store                *bool             `json:"store,omitempty"`
	PreviousResponseID   string            `json:"previous_response_id,omitempty"`
	Stream               bool              `json:"stream,omitempty"`
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
	defaults := map[string]string{}
	if apiKey := strings.TrimSpace(c.opts.APIKey); apiKey != "" {
		defaults["Authorization"] = "Bearer " + apiKey
	}

	return providerRequest{
		path:    c.opts.ChatPath,
		body:    body,
		headers: c.providerHeaders(defaults),
	}
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
	input, previousResponseID, err := codexRequestInput(request, options)
	if err != nil {
		return codexRequest{}, err
	}
	responseOptions := CloneResponseOptions(request.ResponseOptions)

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
		Model:                model,
		Instructions:         instructions,
		Input:                input,
		Tools:                tools,
		ToolChoice:           codexToolChoice(request.ToolChoice),
		ParallelToolCalls:    false,
		PromptCacheKey:       responseOptions.PromptCacheKey,
		PromptCacheRetention: responseOptions.PromptCacheRetention,
		SafetyIdentifier:     responseOptions.SafetyIdentifier,
		Metadata:             responseOptions.Metadata,
		Store:                responseOptions.Store,
		PreviousResponseID:   previousResponseID,
	}, nil
}

func codexToolChoice(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "auto"
	}
	return trimmed
}

func codexRequestInput(
	request CompletionRequest,
	options codexRequestOptions,
) ([]codexInputItem, string, error) {
	messages := request.Messages
	previousResponseID := ""
	if !options.ForceStateless {
		previousResponseID = strings.TrimSpace(request.ConversationState.PreviousResponseID)
	}
	if previousResponseID != "" {
		messages = codexIncrementalMessages(messages)
	}

	input, err := codexInputForMessages(messages, options.ForceStateless)
	if err != nil {
		return nil, "", err
	}
	if previousResponseID != "" && (len(input) == 0 || codexRequiresStatelessReplay(input)) {
		// previous_response_id 模式必须携带增量 input；窗口为空时显式回到无状态输入。
		previousResponseID = ""
		input, err = codexFallbackInput(request.Messages)
		if err != nil {
			return nil, "", err
		}
	}
	if len(input) == 0 {
		return nil, "", fmt.Errorf("codex request input is empty")
	}
	return input, previousResponseID, nil
}

func codexRequiresStatelessReplay(input []codexInputItem) bool {
	if len(input) == 0 {
		return true
	}

	matchedCalls := make(map[string]bool, len(input))
	for _, item := range input {
		if strings.TrimSpace(item.Type) != "function_call" {
			continue
		}
		callID := strings.TrimSpace(item.CallID)
		if callID == "" {
			continue
		}
		matchedCalls[callID] = true
	}

	for _, item := range input {
		if strings.TrimSpace(item.Type) != "function_call_output" {
			continue
		}
		callID := strings.TrimSpace(item.CallID)
		if callID == "" {
			continue
		}
		if matchedCalls[callID] {
			continue
		}
	}

	return false
}

func codexInputForMessages(messages []Message, forceStateless bool) ([]codexInputItem, error) {
	if forceStateless {
		return codexFallbackInput(messages)
	}
	return codexMessagesToInput(messages)
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
