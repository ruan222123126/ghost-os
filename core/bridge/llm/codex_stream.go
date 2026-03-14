package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type codexStreamAccumulator struct {
	message      Message
	finishReason FinishReason
	usage        Usage
	toolCalls    toolCallStreamSet
	final        *CompletionResponse
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
		message:   Message{Role: RoleAssistant},
		toolCalls: newToolCallStreamSet(),
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
		state := a.toolCalls.ensure(event.OutputIndex)
		if err := state.start(ctx, sink); err != nil {
			return err
		}
		return state.appendArguments(ctx, sink, event.Delta)
	case "response.output_item.added":
		if event.Item == nil || strings.TrimSpace(event.Item.Type) != "function_call" {
			return nil
		}
		state := a.toolCalls.ensure(event.OutputIndex)
		a.updateToolState(state, *event.Item)
		return state.start(ctx, sink)
	case "response.output_item.done":
		if event.Item == nil || strings.TrimSpace(event.Item.Type) != "function_call" {
			return nil
		}
		state := a.toolCalls.ensure(event.OutputIndex)
		a.updateToolState(state, *event.Item)
		state.seedArguments(event.Item.Arguments)
		if err := state.start(ctx, sink); err != nil {
			return err
		}
		return state.end(ctx, sink)
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

	a.toolCalls.finalizeMessage(&a.message)
	if a.toolCalls.len() > 0 {
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

func (a *codexStreamAccumulator) updateToolState(state *toolCallStreamState, item codexOutputItem) {
	if state == nil {
		return
	}
	state.setID(codexCallID(item))
	state.setName(item.Name)
}
