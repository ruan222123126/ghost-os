package agent

import (
	"context"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
)

func (a *Agent) handleAssistantTextTurn(
	ctx context.Context,
	turn int,
	resp *llm.CompletionResponse,
	state *agentRunState,
) (bool, turnOutcome, error) {
	if resp == nil || strings.TrimSpace(resp.Message.Text) == "" {
		return false, turnOutcome{}, nil
	}
	for _, handler := range state.assistantTextHandlers {
		handled, outcome, err := a.runAssistantTextHandler(ctx, turn, resp, state, handler)
		if err != nil || handled {
			return handled, outcome, err
		}
	}
	return false, turnOutcome{}, nil
}

func (a *Agent) runAssistantTextHandler(
	ctx context.Context,
	turn int,
	resp *llm.CompletionResponse,
	state *agentRunState,
	handler AssistantTextHandler,
) (bool, turnOutcome, error) {
	if handler == nil {
		return false, turnOutcome{}, nil
	}
	result, err := handler.HandleAssistantText(ctx, AssistantTextRequest{
		Text:    resp.Message.Text,
		TraceID: state.traceID,
	})
	if !result.Recognized {
		return false, turnOutcome{}, nil
	}
	if validateErr := validateAssistantTextResult(result); validateErr != nil {
		return true, turnOutcome{}, state.terminalRunError(ctx, turn, assistantTextTurnError(state.traceID, turn, validateErr))
	}
	return true, turnOutcome{}, a.finalizeAssistantTextTurn(ctx, turn, resp, state, result, err)
}

func (a *Agent) finalizeAssistantTextTurn(
	ctx context.Context,
	turn int,
	resp *llm.CompletionResponse,
	state *agentRunState,
	result AssistantTextResult,
	handlerErr error,
) error {
	if result.Invocation != nil {
		if handlerErr != nil {
			return state.terminalRunError(ctx, turn, assistantTextTurnError(state.traceID, turn, handlerErr))
		}
		return a.finishAssistantTextToolInvocation(ctx, turn, resp, state, result)
	}
	stepID, err := streaming.ToolStepID(turn, 0)
	if err != nil {
		return err
	}
	if err := state.events.toolCallStarted(ctx, state.traceID, turn, stepID, result.Tool.Name, result.Tool.CallID); err != nil {
		return err
	}
	acceptAssistantTurn(state.history, resp)
	if handlerErr != nil {
		return a.finishAssistantTextTurnWithError(ctx, turn, state, stepID, result, handlerErr)
	}
	if err := state.events.toolCallFinished(ctx, state.traceID, turn, stepID, result.Tool.Name, result.Tool.CallID, "success", nil); err != nil {
		return err
	}
	if result.AwaitingHuman != nil {
		return a.finishAssistantTextTurnAwaitingHuman(ctx, turn, state, stepID, result.AwaitingHuman)
	}
	for _, message := range cloneAssistantTextFeedback(result.Feedback) {
		state.history.Append(message)
	}
	a.commitTurn(state.history)
	return nil
}

func (a *Agent) finishAssistantTextToolInvocation(
	ctx context.Context,
	turn int,
	resp *llm.CompletionResponse,
	state *agentRunState,
	result AssistantTextResult,
) error {
	acceptAssistantTurn(state.history, resp)
	outcome, err := state.toolCalls.executeSingle(
		ctx,
		state.traceID,
		turn,
		result.Tool.Name,
		result.Tool.CallID,
		result.Invocation.Arguments,
	)
	if err != nil || outcome.stopErr != nil {
		if outcome.stopErr != nil {
			err = outcome.stopErr
		}
		return a.handleToolCallExecutionError(ctx, turn, err, state)
	}
	if outcome.executed {
		for _, message := range cloneAssistantTextFeedback(result.Feedback) {
			state.history.Append(message)
		}
		for _, message := range buildAssistantTextToolFeedback(
			result.Invocation,
			AssistantTextToolExecutionResult{
				Tool:   result.Tool,
				Output: outcome.output,
				Meta:   outcome.meta,
			},
		) {
			state.history.Append(message)
		}
	}
	a.commitTurn(state.history)
	return nil
}

func (a *Agent) finishAssistantTextTurnWithError(
	ctx context.Context,
	turn int,
	state *agentRunState,
	stepID string,
	result AssistantTextResult,
	handlerErr error,
) error {
	if emitErr := state.events.toolCallFinished(ctx, state.traceID, turn, stepID, result.Tool.Name, result.Tool.CallID, "error", handlerErr); emitErr != nil {
		return emitErr
	}
	if protocolErr, ok := tools.AsGraphQLTextProtocolError(handlerErr); ok && protocolErr.Recoverable() {
		for _, message := range cloneAssistantTextFeedback(result.Feedback) {
			state.history.Append(message)
		}
		a.commitTurn(state.history)
		return nil
	}
	return state.terminalRunError(ctx, turn, assistantTextTurnError(state.traceID, turn, handlerErr))
}

func (a *Agent) finishAssistantTextTurnAwaitingHuman(
	ctx context.Context,
	turn int,
	state *agentRunState,
	stepID string,
	awaiting *AssistantTextAwaitingHuman,
) error {
	if err := state.events.awaitingHuman(
		ctx,
		state.traceID,
		turn,
		stepID,
		awaiting.Tool.Name,
		awaiting.Tool.CallID,
		awaiting.QuestionID,
		awaiting.Prompt,
		awaiting.SelectionMode,
		cloneAssistantTextOptions(awaiting.Options),
	); err != nil {
		return err
	}
	a.commitTurn(state.history)
	return &ErrAwaitingHuman{
		QuestionID:    awaiting.QuestionID,
		Prompt:        awaiting.Prompt,
		SelectionMode: awaiting.SelectionMode,
		Options:       cloneAssistantTextOptions(awaiting.Options),
	}
}

func assistantTextTurnError(traceID string, turn int, err error) error {
	return fmt.Errorf("trace_id=%s turn=%d handle_assistant_text: %w", traceID, turn, err)
}

func strictToolCallProtocolError(traceID string, turn int) error {
	return fmt.Errorf(
		"trace_id=%s turn=%d strict text protocol rejects tool_calls finish reason",
		traceID,
		turn,
	)
}
