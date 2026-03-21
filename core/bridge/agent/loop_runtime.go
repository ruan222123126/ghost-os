package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
)

type agentRunState struct {
	traceID string
	sink    streaming.Sink

	history                *History
	completion             completionRunner
	toolCalls              toolCallExecutor
	graphQL                tools.GraphQLTextExecutor
	events                 agentEventEmitter
	lifecycle              StreamLifecyclePayloadBuilder
	strictToolCallProtocol bool

	consecutiveNonExecutableToolCallTurns int
}

func newAgentRunState(a *Agent, sink streaming.Sink, traceID string) agentRunState {
	history := NewHistory("")
	if a != nil && a.history != nil {
		history = a.history.Clone()
	}

	lifecycle := StreamLifecyclePayloadBuilder{}
	if a != nil {
		lifecycle = a.streamLifecycle
	}

	events := newAgentEventEmitter(sink, lifecycle.SessionID)
	return agentRunState{
		traceID:                normalizeTraceID(traceID),
		sink:                   sink,
		history:                history,
		completion:             newCompletionRunner(a.completer, a.tools, history),
		toolCalls:              newToolCallExecutor(a.tools, history, nil, events),
		graphQL:                a.graphQL,
		events:                 events,
		lifecycle:              lifecycle,
		strictToolCallProtocol: a.strictToolCallProtocol,
	}
}

func normalizeTraceID(traceID string) string {
	trimmed := strings.TrimSpace(traceID)
	if trimmed != "" {
		return trimmed
	}
	return fmt.Sprintf("agent-%d", time.Now().UnixNano())
}

func (state agentRunState) complete(ctx context.Context, turn int) (*llm.CompletionResponse, error) {
	return state.completion.complete(
		ctx,
		state.sink,
		state.traceID,
		state.lifecycle.sessionID(),
		turn,
	)
}

func (state agentRunState) maxTurnsExceeded(ctx context.Context, lastTurn int, maxTurns int) error {
	runErr := fmt.Errorf("trace_id=%s max turns exceeded: %d", state.traceID, maxTurns)
	return state.terminalRunError(ctx, lastTurn, runErr)
}

func (state agentRunState) terminalRunError(ctx context.Context, turn int, runErr error) error {
	stepID, err := streaming.AssistantStepID(turn)
	if err != nil {
		return err
	}
	return state.events.terminalError(ctx, state.traceID, turn, stepID, runErr)
}
