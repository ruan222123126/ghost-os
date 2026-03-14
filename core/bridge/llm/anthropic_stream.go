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
			if state.kind == "text" && strings.TrimSpace(event.ContentBlock.Text) != "" {
				state.text.WriteString(event.ContentBlock.Text)
				if err := sink.OnDelta(ctx, LLMDelta{
					Kind: DeltaKindText,
					Text: event.ContentBlock.Text,
				}); err != nil {
					return err
				}
			}
			if state.kind == "tool_use" {
				tool := state.ensureTool()
				tool.setID(event.ContentBlock.ID)
				tool.setName(event.ContentBlock.Name)
				if err := tool.start(ctx, sink); err != nil {
					return err
				}
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
			tool := state.ensureTool()
			if err := tool.start(ctx, sink); err != nil {
				return err
			}
			if err := tool.appendArguments(ctx, sink, event.Delta.PartialJSON); err != nil {
				return err
			}
		}
	case "content_block_stop":
		state := a.blocks[event.Index]
		if state == nil || state.kind != "tool_use" || state.tool == nil {
			return nil
		}
		return state.tool.end(ctx, sink)
	case "message_delta":
		if event.Delta != nil && strings.TrimSpace(event.Delta.StopReason) != "" {
			a.stopReason = strings.TrimSpace(event.Delta.StopReason)
		}
		if event.Usage.InputTokens > 0 {
			a.usage.InputTokens = event.Usage.InputTokens
		}
		if event.Usage.OutputTokens > 0 {
			a.usage.OutputTokens = event.Usage.OutputTokens
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
