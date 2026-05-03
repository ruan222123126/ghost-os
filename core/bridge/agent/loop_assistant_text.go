package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
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
	if result.Invocation != nil || len(result.Invocations) > 0 {
		if handlerErr != nil {
			return state.terminalRunError(ctx, turn, assistantTextTurnError(state.traceID, turn, handlerErr))
		}
		return a.finishAssistantTextToolInvocations(ctx, turn, resp, state, result)
	}
	stepID, err := streaming.ToolStepID(turn, 0)
	if err != nil {
		return err
	}
	if err := state.events.toolCallStarted(ctx, state.traceID, turn, stepID, result.Tool.Name, result.Tool.CallID, ""); err != nil {
		return err
	}
	acceptAssistantTurn(state.history, resp)
	if handlerErr != nil {
		return a.finishAssistantTextTurnWithError(ctx, turn, state, stepID, result, handlerErr)
	}
	if err := state.events.toolCallFinished(ctx, state.traceID, turn, stepID, result.Tool.Name, result.Tool.CallID, "success", nil, ""); err != nil {
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

func (a *Agent) finishAssistantTextToolInvocations(
	ctx context.Context,
	turn int,
	resp *llm.CompletionResponse,
	state *agentRunState,
	result AssistantTextResult,
) error {
	invocations := assistantTextToolInvocations(result)
	acceptAssistantTurn(state.history, assistantTextToolCallResponse(resp, invocations))
	for index, invocation := range invocations {
		outcome, err := state.toolCalls.executeSingleWithStepIndex(
			ctx,
			state.traceID,
			turn,
			index,
			invocation.Tool.Name,
			invocation.Tool.CallID,
			invocation.Invocation.Arguments,
		)
		if err != nil || outcome.stopErr != nil {
			if outcome.stopErr != nil {
				err = outcome.stopErr
			}
			return a.handleToolCallExecutionError(ctx, turn, err, state)
		}
		if outcome.executed {
			for _, message := range buildAssistantTextToolFeedback(
				&invocation.Invocation,
				AssistantTextToolExecutionResult{
					Tool:   invocation.Tool,
					Output: outcome.output,
					Meta:   outcome.meta,
				},
			) {
				state.history.Append(message)
			}
		}
	}
	for _, message := range cloneAssistantTextFeedback(result.Feedback) {
		state.history.Append(message)
	}
	a.commitTurn(state.history)
	return nil
}

func assistantTextToolCallResponse(
	resp *llm.CompletionResponse,
	invocations []AssistantTextToolInvocationEntry,
) *llm.CompletionResponse {
	if resp == nil || len(invocations) == 0 {
		return resp
	}
	cloned := *resp
	clonedMessages := llm.CloneMessages([]llm.Message{resp.Message})
	if len(clonedMessages) != 1 {
		return resp
	}
	cloned.Message = clonedMessages[0]
	cloned.Message.ToolCalls = assistantTextToolCalls(invocations)
	return &cloned
}

func assistantTextToolCalls(invocations []AssistantTextToolInvocationEntry) []llm.ToolCall {
	calls := make([]llm.ToolCall, 0, len(invocations))
	for _, invocation := range invocations {
		calls = append(calls, llm.ToolCall{
			ID:        strings.TrimSpace(invocation.Tool.CallID),
			Name:      strings.TrimSpace(invocation.Tool.Name),
			Arguments: cloneAssistantTextArguments(invocation.Invocation.Arguments),
		})
	}
	return calls
}

func cloneAssistantTextArguments(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	cloned := make([]byte, len(raw))
	copy(cloned, raw)
	return json.RawMessage(cloned)
}

func assistantTextToolInvocations(result AssistantTextResult) []AssistantTextToolInvocationEntry {
	if len(result.Invocations) > 0 {
		return append([]AssistantTextToolInvocationEntry(nil), result.Invocations...)
	}
	if result.Invocation == nil {
		return nil
	}
	return []AssistantTextToolInvocationEntry{
		{
			Tool:       result.Tool,
			Invocation: *result.Invocation,
		},
	}
}

func (a *Agent) finishAssistantTextTurnWithError(
	ctx context.Context,
	turn int,
	state *agentRunState,
	stepID string,
	result AssistantTextResult,
	handlerErr error,
) error {
	if emitErr := state.events.toolCallFinished(ctx, state.traceID, turn, stepID, result.Tool.Name, result.Tool.CallID, "error", handlerErr, ""); emitErr != nil {
		return emitErr
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
		awaitingHumanEventRequest{
			Turn:          turn,
			StepID:        stepID,
			ToolName:      awaiting.Tool.Name,
			ToolCallID:    awaiting.Tool.CallID,
			QuestionID:    awaiting.QuestionID,
			Prompt:        awaiting.Prompt,
			SelectionMode: awaiting.SelectionMode,
			Options:       cloneAssistantTextOptions(awaiting.Options),
		},
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
