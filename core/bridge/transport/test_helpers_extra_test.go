package transport

import (
	"context"
	"errors"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	bridgeorchestration "ghost-os/bridge/orchestration"
	"ghost-os/bridge/streaming"
)

const cancelledHumanDialogueMessage = "Conversation cancelled by user."

func mustAppEvent(
	t *testing.T,
	traceID string,
	sessionID string,
	turn int,
	stepID string,
	eventType streaming.EventType,
	payload any,
) streaming.Event {
	t.Helper()

	event, err := streaming.NewEvent(traceID, sessionID, turn, stepID, eventType, payload)
	if err != nil {
		t.Fatalf("NewEvent returned error: %v", err)
	}
	return event
}

func mustAppAssistantStepID(t *testing.T, turn int) string {
	t.Helper()

	stepID, err := streaming.AssistantStepID(turn)
	if err != nil {
		t.Fatalf("AssistantStepID returned error: %v", err)
	}
	return stepID
}

func mustAppToolStepID(t *testing.T, turn int, toolIndex int) string {
	t.Helper()

	stepID, err := streaming.ToolStepID(turn, toolIndex)
	if err != nil {
		t.Fatalf("ToolStepID returned error: %v", err)
	}
	return stepID
}

type proTestRuntimeFactory struct {
	deps bridgeorchestration.RuntimeDependencies
	err  error
}

func (f proTestRuntimeFactory) Build(bridgeconfig.Store) (bridgeorchestration.RuntimeDependencies, error) {
	if f.err != nil {
		return bridgeorchestration.RuntimeDependencies{}, f.err
	}
	return f.deps, nil
}

type proTestCompleter struct {
	responses []*llm.CompletionResponse
	requests  []llm.CompletionRequest
}

func (f *proTestCompleter) Complete(_ context.Context, request llm.CompletionRequest) (*llm.CompletionResponse, error) {
	f.requests = append(f.requests, request)
	if len(f.responses) == 0 {
		return nil, errors.New("unexpected complete call")
	}
	response := f.responses[0]
	f.responses = f.responses[1:]
	return response, nil
}
