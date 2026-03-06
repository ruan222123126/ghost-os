package agent

import (
	"context"

	"ghost-os/bridge/llm"
)

type llmDeltaBridge struct {
	sink    EventSink
	traceID string
	turn    int
	stepID  string
}

func newLLMDeltaBridge(sink EventSink, traceID string, turn int) *llmDeltaBridge {
	return &llmDeltaBridge{
		sink:    sink,
		traceID: traceID,
		turn:    turn,
		stepID:  AssistantStepID(turn),
	}
}

func (b *llmDeltaBridge) OnDelta(ctx context.Context, delta llm.LLMDelta) error {
	payload := map[string]any{
		"kind": string(delta.Kind),
	}

	switch delta.Kind {
	case llm.DeltaKindText:
		payload["text"] = delta.Text
	case llm.DeltaKindToolCallStart:
		payload["tool_call_index"] = delta.ToolCallIndex
		payload["tool_call_id"] = delta.ToolCallID
		payload["tool_name"] = delta.ToolName
	case llm.DeltaKindToolCallDelta:
		payload["tool_call_index"] = delta.ToolCallIndex
		payload["arguments_fragment"] = delta.ArgumentsFragment
	case llm.DeltaKindToolCallEnd:
		payload["tool_call_index"] = delta.ToolCallIndex
	}

	return b.sink.Emit(ctx, NewEvent(b.traceID, b.turn, b.stepID, EventCompletionDelta, payload))
}
