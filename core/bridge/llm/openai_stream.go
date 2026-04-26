package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type openAIStreamAccumulator struct {
	message      Message
	finishReason string
	usage        Usage
	toolCalls    toolCallStreamSet
	reasoning    strings.Builder
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
		toolCalls: newToolCallStreamSet(),
	}
}

func (a *openAIStreamAccumulator) ApplyDelta(ctx context.Context, sink LLMStreamSink, delta openAIStreamDelta) error {
	if role := strings.TrimSpace(delta.Role); role != "" {
		a.message.Role = Role(role)
	}
	if err := a.applyReasoningDelta(ctx, sink, delta.ReasoningContent, "reasoning_content"); err != nil {
		return err
	}
	if err := a.applyReasoningDelta(ctx, sink, delta.Reasoning, "reasoning"); err != nil {
		return err
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
		state := a.toolCalls.ensure(toolDelta.Index)
		state.setID(toolDelta.ID)
		state.setName(toolDelta.Function.Name)
		if err := state.start(ctx, sink); err != nil {
			return err
		}
		if err := state.appendArguments(ctx, sink, toolDelta.Function.Arguments); err != nil {
			return err
		}
	}

	return nil
}

func (a *openAIStreamAccumulator) applyReasoningDelta(
	ctx context.Context,
	sink LLMStreamSink,
	value any,
	field string,
) error {
	if value == nil {
		return nil
	}
	reasoning, err := parseOpenAIReasoningText(value, field)
	if err != nil {
		return err
	}
	if reasoning == "" {
		return nil
	}
	a.reasoning.WriteString(reasoning)
	return sink.OnDelta(ctx, LLMDelta{
		Kind:     DeltaKindThinking,
		Thinking: reasoning,
	})
}

func parseOpenAIReasoningText(value any, field string) (string, error) {
	switch typed := value.(type) {
	case string:
		return typed, nil
	case map[string]any:
		return parseOpenAIReasoningTextFromObject(typed, field)
	case []any:
		return parseOpenAIReasoningTextFromArray(typed, field)
	default:
		return "", fmt.Errorf("unsupported openai stream %s payload type %T", field, value)
	}
}

func parseOpenAIReasoningTextFromObject(value map[string]any, field string) (string, error) {
	for _, key := range []string{"text", "content", "reasoning"} {
		raw, ok := value[key]
		if !ok {
			continue
		}
		text, ok := raw.(string)
		if !ok {
			return "", fmt.Errorf("unsupported openai stream %s.%s type %T", field, key, raw)
		}
		return text, nil
	}
	return "", fmt.Errorf("unsupported openai stream %s payload object keys: missing text/content/reasoning", field)
}

func parseOpenAIReasoningTextFromArray(value []any, field string) (string, error) {
	if len(value) == 0 {
		return "", nil
	}

	var builder strings.Builder
	for idx, item := range value {
		text, err := parseOpenAIReasoningText(item, fmt.Sprintf("%s[%d]", field, idx))
		if err != nil {
			return "", err
		}
		builder.WriteString(text)
	}
	return builder.String(), nil
}

func (a *openAIStreamAccumulator) SetFinishReason(ctx context.Context, sink LLMStreamSink, reason string) error {
	a.finishReason = strings.TrimSpace(reason)
	if a.finishReason != "tool_calls" {
		return nil
	}
	return a.toolCalls.endAll(ctx, sink)
}

func (a *openAIStreamAccumulator) CompletionResponse() (*CompletionResponse, error) {
	if a.message.Role == "" {
		a.message.Role = RoleAssistant
	}
	if a.reasoning.Len() > 0 {
		raw, err := json.Marshal(a.reasoning.String())
		if err != nil {
			return nil, fmt.Errorf("encode openai stream reasoning_content: %w", err)
		}
		a.message.ReasoningContent = raw
	}
	a.toolCalls.finalizeMessage(&a.message)

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
