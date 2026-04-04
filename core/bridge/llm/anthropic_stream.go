package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type anthropicStreamAccumulator struct {
	role       Role
	stopReason string
	usage      anthropicUsage
	blocks     map[int]*anthropicStreamBlockState
	blockOrder []int
}

type anthropicStreamBlockState struct {
	index int
	kind  string
	text  strings.Builder
	tool  *toolCallStreamState
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
		a.handleMessageStart(event)
		return nil
	case "content_block_start":
		return a.handleContentBlockStart(ctx, sink, event)
	case "content_block_delta":
		return a.handleContentBlockDelta(ctx, sink, event)
	case "content_block_stop":
		return a.handleContentBlockStop(ctx, sink, event.Index)
	case "message_delta":
		a.handleMessageDelta(event)
		return nil
	case "message_stop":
		return nil
	default:
		return nil
	}
}

func (a *anthropicStreamAccumulator) handleMessageStart(event anthropicStreamEvent) {
	if event.Message == nil {
		return
	}
	if role := strings.TrimSpace(event.Message.Role); role != "" {
		a.role = Role(role)
	}
	a.usage = event.Message.Usage
}

func (a *anthropicStreamAccumulator) handleContentBlockStart(
	ctx context.Context,
	sink LLMStreamSink,
	event anthropicStreamEvent,
) error {
	if event.ContentBlock == nil {
		return nil
	}
	state := a.ensureBlockState(event.Index)
	state.kind = strings.TrimSpace(event.ContentBlock.Type)
	if state.kind == "text" {
		if strings.TrimSpace(event.ContentBlock.Text) == "" {
			return nil
		}
		return a.emitTextDelta(ctx, sink, state, event.ContentBlock.Text)
	}
	if state.kind != "tool_use" {
		return nil
	}
	tool := state.ensureTool()
	tool.setID(event.ContentBlock.ID)
	tool.setName(event.ContentBlock.Name)
	return tool.start(ctx, sink)
}

func (a *anthropicStreamAccumulator) handleContentBlockDelta(
	ctx context.Context,
	sink LLMStreamSink,
	event anthropicStreamEvent,
) error {
	if event.Delta == nil {
		return nil
	}
	state := a.ensureBlockState(event.Index)
	switch strings.TrimSpace(event.Delta.Type) {
	case "text_delta":
		state.kind = "text"
		return a.emitTextDelta(ctx, sink, state, event.Delta.Text)
	case "input_json_delta":
		state.kind = "tool_use"
		tool := state.ensureTool()
		if err := tool.start(ctx, sink); err != nil {
			return err
		}
		return tool.appendArguments(ctx, sink, event.Delta.PartialJSON)
	default:
		return nil
	}
}

func (a *anthropicStreamAccumulator) emitTextDelta(
	ctx context.Context,
	sink LLMStreamSink,
	state *anthropicStreamBlockState,
	text string,
) error {
	if text == "" {
		return nil
	}
	state.text.WriteString(text)
	return sink.OnDelta(ctx, LLMDelta{
		Kind: DeltaKindText,
		Text: text,
	})
}

func (a *anthropicStreamAccumulator) handleContentBlockStop(
	ctx context.Context,
	sink LLMStreamSink,
	index int,
) error {
	state := a.blocks[index]
	if state == nil || state.kind != "tool_use" || state.tool == nil {
		return nil
	}
	return state.tool.end(ctx, sink)
}

func (a *anthropicStreamAccumulator) handleMessageDelta(event anthropicStreamEvent) {
	if event.Delta != nil {
		stopReason := strings.TrimSpace(event.Delta.StopReason)
		if stopReason != "" {
			a.stopReason = stopReason
		}
	}
	if event.Usage.InputTokens > 0 {
		a.usage.InputTokens = event.Usage.InputTokens
	}
	if event.Usage.OutputTokens > 0 {
		a.usage.OutputTokens = event.Usage.OutputTokens
	}
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
			if state.tool != nil {
				decoded, err := decodeJSONObjectString(state.tool.args.String())
				if err != nil {
					return nil, fmt.Errorf("decode anthropic tool input for %q: %w", state.tool.call.Name, err)
				}
				input = decoded
			}
			response.Content = append(response.Content, anthropicContentBlock{
				Type:  "tool_use",
				ID:    state.tool.call.ID,
				Name:  state.tool.call.Name,
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

func (s *anthropicStreamBlockState) ensureTool() *toolCallStreamState {
	if s.tool != nil {
		return s.tool
	}
	s.tool = &toolCallStreamState{index: s.index}
	return s.tool
}
