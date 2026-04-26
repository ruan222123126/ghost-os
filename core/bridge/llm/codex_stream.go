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
		return a.handleOutputTextDelta(ctx, sink, event.Delta)
	case "response.reasoning_summary_text.delta", "response.reasoning_text.delta":
		return a.handleThinkingDelta(ctx, sink, event.Delta)
	case "response.function_call_arguments.delta":
		return a.handleToolArgumentsDelta(ctx, sink, event.OutputIndex, event.Delta)
	case "response.output_item.added":
		return a.handleOutputItemAdded(ctx, sink, event.OutputIndex, event.Item)
	case "response.output_item.done":
		return a.handleOutputItemDone(ctx, sink, event.OutputIndex, event.Item)
	case "response.completed":
		return a.handleResponseCompleted(event.Response)
	default:
		return nil
	}
}

func (a *codexStreamAccumulator) handleOutputTextDelta(
	ctx context.Context,
	sink LLMStreamSink,
	delta string,
) error {
	if delta == "" {
		return nil
	}
	a.message.Text += delta
	return sink.OnDelta(ctx, LLMDelta{
		Kind: DeltaKindText,
		Text: delta,
	})
}

func (a *codexStreamAccumulator) handleThinkingDelta(
	ctx context.Context,
	sink LLMStreamSink,
	delta string,
) error {
	if delta == "" {
		return nil
	}
	return sink.OnDelta(ctx, LLMDelta{
		Kind:     DeltaKindThinking,
		Thinking: delta,
	})
}

func (a *codexStreamAccumulator) handleToolArgumentsDelta(
	ctx context.Context,
	sink LLMStreamSink,
	outputIndex int,
	delta string,
) error {
	state := a.toolCalls.ensure(outputIndex)
	if err := state.start(ctx, sink); err != nil {
		return err
	}
	return state.appendArguments(ctx, sink, delta)
}

func (a *codexStreamAccumulator) handleOutputItemAdded(
	ctx context.Context,
	sink LLMStreamSink,
	outputIndex int,
	item *codexOutputItem,
) error {
	if !isCodexFunctionCallItem(item) {
		return nil
	}
	state := a.toolCalls.ensure(outputIndex)
	a.updateToolState(state, *item)
	return state.start(ctx, sink)
}

func (a *codexStreamAccumulator) handleOutputItemDone(
	ctx context.Context,
	sink LLMStreamSink,
	outputIndex int,
	item *codexOutputItem,
) error {
	if !isCodexFunctionCallItem(item) {
		return nil
	}
	state := a.toolCalls.ensure(outputIndex)
	a.updateToolState(state, *item)
	state.seedArguments(item.Arguments)
	if err := state.start(ctx, sink); err != nil {
		return err
	}
	return state.end(ctx, sink)
}

func isCodexFunctionCallItem(item *codexOutputItem) bool {
	return item != nil && strings.TrimSpace(item.Type) == "function_call"
}

func (a *codexStreamAccumulator) handleResponseCompleted(response *codexResponse) error {
	if response == nil {
		return nil
	}
	resp, err := codexToCompletionResponse(*response)
	if err != nil {
		return err
	}
	a.final = resp
	a.usage = resp.Usage
	a.finishReason = resp.FinishReason
	return nil
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
