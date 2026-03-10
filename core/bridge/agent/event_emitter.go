package agent

import (
	"context"
	"errors"
	"fmt"

	"ghost-os/bridge/tools"
)

// agentEventEmitter 把 EventSink payload 组装和错误包装从对话循环中移走。
type agentEventEmitter struct {
	sink EventSink
}

func newAgentEventEmitter(sink EventSink) agentEventEmitter {
	if sink == nil {
		sink = nopSink{}
	}
	return agentEventEmitter{sink: sink}
}

func (e agentEventEmitter) emit(ctx context.Context, event AgentEvent) error {
	if err := e.sink.Emit(ctx, event); err != nil {
		return &eventEmitError{eventType: event.Type, err: err}
	}
	return nil
}

func (e agentEventEmitter) toolCallStarted(ctx context.Context, traceID string, turn int, stepID string, toolName string, toolCallID string) error {
	return e.emit(ctx, NewEvent(traceID, turn, stepID, EventToolCallStarted, map[string]any{
		"tool":         toolName,
		"tool_call_id": toolCallID,
	}))
}

func (e agentEventEmitter) toolCallFinished(ctx context.Context, traceID string, turn int, stepID string, toolName string, toolCallID string, status string, toolErr error) error {
	payload := map[string]any{
		"tool":         toolName,
		"tool_call_id": toolCallID,
		"status":       status,
	}
	if toolErr != nil {
		payload["error"] = toolErr.Error()
	}
	return e.emit(ctx, NewEvent(traceID, turn, stepID, EventToolCallFinished, payload))
}

func (e agentEventEmitter) awaitingHuman(ctx context.Context, traceID string, turn int, stepID string, toolName string, toolCallID string, questionID string, prompt string, selectionMode string, options []tools.AskHumanOption) error {
	payload := map[string]any{
		"tool":         toolName,
		"tool_call_id": toolCallID,
		"question_id":  questionID,
		"prompt":       prompt,
	}
	if selectionMode != "" {
		payload["selection_mode"] = selectionMode
	}
	if len(options) > 0 {
		payload["options"] = options
	}
	return e.emit(ctx, NewEvent(traceID, turn, stepID, EventAwaitingHuman, payload))
}

func (e agentEventEmitter) terminalError(ctx context.Context, traceID string, turn int, stepID string, runErr error) error {
	if emitErr := e.emit(ctx, NewEvent(traceID, turn, stepID, EventError, map[string]any{
		"message": runErr.Error(),
	})); emitErr != nil {
		return emitErr
	}
	return runErr
}

type eventEmitError struct {
	eventType EventType
	err       error
}

func (e *eventEmitError) Error() string {
	return fmt.Sprintf("emit event %q: %v", e.eventType, e.err)
}

func (e *eventEmitError) Unwrap() error {
	return e.err
}

func isEventEmitError(err error) bool {
	var emitErr *eventEmitError
	return errors.As(err, &emitErr)
}
