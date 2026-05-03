package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
)

var (
	errAgentCompleterRequired  = errors.New("agent completer is nil")
	errAgentHistoryRequired    = errors.New("agent history is nil")
	errAgentRequired           = errors.New("agent is nil")
	errAgentToolCatalogMissing = errors.New("agent tool catalog is nil")
)

type agentRunState struct {
	traceID string
	sink    streaming.Sink

	history                *History
	completion             completionRunner
	toolCalls              toolCallExecutor
	assistantTextHandlers  []AssistantTextHandler
	beforeCompletion       BeforeCompletionHook
	events                 agentEventEmitter
	lifecycle              StreamLifecyclePayloadBuilder
	strictToolCallProtocol bool

	consecutiveNonExecutableToolCallTurns int
}

func newAgentRunState(a *Agent, sink streaming.Sink, traceID string) (agentRunState, error) {
	lifecycle := StreamLifecyclePayloadBuilder{}
	attemptState := &completionAttemptState{}
	state := agentRunState{
		traceID:   normalizeTraceID(traceID),
		sink:      sink,
		history:   NewHistory(""),
		lifecycle: lifecycle,
	}
	state.events = newAgentEventEmitter(sink, lifecycle.sessionID)
	if err := validateAgentForRun(a); err != nil {
		return state, err
	}

	history := cloneHistoryForRun(a.history)
	lifecycle = a.streamLifecycle
	events := newAgentEventEmitter(sink, lifecycle.SessionID)
	retryPolicy := DefaultCompletionRetryPolicy()
	if a.completionRetryPolicy != nil {
		retryPolicy = *a.completionRetryPolicy
	}
	return agentRunState{
		traceID:                normalizeTraceID(traceID),
		sink:                   sink,
		history:                history,
		completion:             newCompletionRunnerWithPolicy(a.completer, a.tools, history, a.responseOptions, retryPolicy).withAttemptState(attemptState),
		toolCalls:              newToolCallExecutor(a.tools, history, nil, events),
		assistantTextHandlers:  append([]AssistantTextHandler(nil), a.assistantTextHandlers...),
		beforeCompletion:       a.beforeCompletion,
		events:                 events,
		lifecycle:              lifecycle,
		strictToolCallProtocol: a.strictToolCallProtocol,
	}, nil
}

func cloneHistoryForRun(history *History) *History {
	cloned := history.Clone()
	maxMessages := history.maxMessages
	if maxMessages <= 0 {
		maxMessages = DefaultMaxHistoryMessages
	}
	cloned.maxMessages = maxMessages
	cloned.trimToMax()
	return cloned
}

func validateAgentForRun(a *Agent) error {
	if a == nil {
		return errAgentRequired
	}
	if a.completer == nil {
		return errAgentCompleterRequired
	}
	if a.tools == nil {
		return errAgentToolCatalogMissing
	}
	if a.history == nil {
		return errAgentHistoryRequired
	}
	return nil
}

func normalizeTraceID(traceID string) string {
	trimmed := strings.TrimSpace(traceID)
	if trimmed != "" {
		return trimmed
	}
	return fmt.Sprintf("agent-%d", time.Now().UnixNano())
}

func (state agentRunState) complete(ctx context.Context, turn int) (*llm.CompletionResponse, error) {
	if state.beforeCompletion != nil {
		if err := state.beforeCompletion(ctx, turn, state.history); err != nil {
			return nil, err
		}
	}
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
