package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
)

// agentEventEmitter 把 EventSink payload 组装和错误包装从对话循环中移走。
type agentEventEmitter struct {
	sink      streaming.Sink
	sessionID func() string
}

func newAgentEventEmitter(sink streaming.Sink, sessionID func() string) agentEventEmitter {
	if sink == nil {
		sink = streaming.NopSink{}
	}
	return agentEventEmitter{
		sink:      sink,
		sessionID: sessionID,
	}
}

func (e agentEventEmitter) currentSessionID() string {
	if e.sessionID == nil {
		return ""
	}
	return strings.TrimSpace(e.sessionID())
}

func (e agentEventEmitter) newEvent(traceID string, turn int, stepID string, eventType streaming.EventType, payload any) (streaming.Event, error) {
	return streaming.NewEvent(traceID, e.currentSessionID(), turn, stepID, eventType, payload)
}

func (e agentEventEmitter) emit(ctx context.Context, event streaming.Event) error {
	if _, err := e.sink.Emit(ctx, event); err != nil {
		return &eventEmitError{eventType: event.Type, err: err}
	}
	return nil
}

func (e agentEventEmitter) runStarted(ctx context.Context, traceID string, builder StreamLifecyclePayloadBuilder) error {
	payload, err := builder.runStartedPayload()
	if err != nil {
		return e.terminalError(ctx, traceID, 0, "", fmt.Errorf("build run_started payload: %w", err))
	}
	event, err := streaming.NewEvent(traceID, builder.sessionID(), 0, "", streaming.EventRunStarted, payload)
	if err != nil {
		return err
	}
	return e.emit(ctx, event)
}

func (e agentEventEmitter) toolCallStarted(ctx context.Context, traceID string, turn int, stepID string, toolName string, toolCallID string) error {
	event, err := e.newEvent(traceID, turn, stepID, streaming.EventToolCallStarted, map[string]any{
		"tool":         toolName,
		"tool_call_id": toolCallID,
	})
	if err != nil {
		return err
	}
	return e.emit(ctx, event)
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
	event, err := e.newEvent(traceID, turn, stepID, streaming.EventToolCallFinished, payload)
	if err != nil {
		return err
	}
	return e.emit(ctx, event)
}

type awaitingHumanEventRequest struct {
	Turn          int
	StepID        string
	ToolName      string
	ToolCallID    string
	QuestionID    string
	Prompt        string
	SelectionMode string
	Options       []tools.AskHumanOption
}

func (e agentEventEmitter) awaitingHuman(ctx context.Context, traceID string, request awaitingHumanEventRequest) error {
	payload := map[string]any{
		"tool":         request.ToolName,
		"tool_call_id": request.ToolCallID,
		"question_id":  request.QuestionID,
		"prompt":       request.Prompt,
	}
	if request.SelectionMode != "" {
		payload["selection_mode"] = request.SelectionMode
	}
	if len(request.Options) > 0 {
		payload["options"] = request.Options
	}
	event, err := e.newEvent(traceID, request.Turn, request.StepID, streaming.EventAwaitingHuman, payload)
	if err != nil {
		return err
	}
	return e.emit(ctx, event)
}

func (e agentEventEmitter) terminalSuccess(ctx context.Context, traceID string, turn int, response string, builder StreamLifecyclePayloadBuilder) error {
	messagePayload, err := builder.messagePayload(response)
	if err != nil {
		stepID, stepErr := streaming.AssistantStepID(turn)
		if stepErr != nil {
			return stepErr
		}
		return e.terminalError(ctx, traceID, turn, stepID, fmt.Errorf("build message payload: %w", err))
	}
	donePayload, err := builder.donePayload(response)
	if err != nil {
		stepID, stepErr := streaming.AssistantStepID(turn)
		if stepErr != nil {
			return stepErr
		}
		return e.terminalError(ctx, traceID, turn, stepID, fmt.Errorf("build done payload: %w", err))
	}
	stepID, err := streaming.AssistantStepID(turn)
	if err != nil {
		return err
	}
	messageEvent, err := streaming.NewEvent(traceID, builder.sessionID(), turn, stepID, streaming.EventMessage, messagePayload)
	if err != nil {
		return err
	}
	if err := e.emit(ctx, messageEvent); err != nil {
		return err
	}
	doneEvent, err := streaming.NewEvent(traceID, builder.sessionID(), turn, "", streaming.EventDone, donePayload)
	if err != nil {
		return err
	}
	return e.emit(ctx, doneEvent)
}

func (e agentEventEmitter) terminalError(ctx context.Context, traceID string, turn int, stepID string, runErr error) error {
	event, err := e.newEvent(traceID, turn, stepID, streaming.EventError, map[string]any{
		"message": runErr.Error(),
	})
	if err != nil {
		return err
	}
	if emitErr := e.emit(ctx, event); emitErr != nil {
		return emitErr
	}
	return runErr
}

type eventEmitError struct {
	eventType streaming.EventType
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
