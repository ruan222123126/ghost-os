package agent

import (
	"context"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
)

type llmDeltaBridge struct {
	sink      streaming.Sink
	traceID   string
	sessionID string
	turn      int
	stepID    string
}

func newLLMDeltaBridge(sink streaming.Sink, traceID string, sessionID string, turn int) (*llmDeltaBridge, error) {
	stepID, err := streaming.AssistantStepID(turn)
	if err != nil {
		return nil, err
	}
	return &llmDeltaBridge{
		sink:      sink,
		traceID:   traceID,
		sessionID: sessionID,
		turn:      turn,
		stepID:    stepID,
	}, nil
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

	event, err := streaming.NewEvent(b.traceID, b.sessionID, b.turn, b.stepID, streaming.EventCompletionDelta, payload)
	if err != nil {
		return err
	}
	_, err = b.sink.Emit(ctx, event)
	return err
}
