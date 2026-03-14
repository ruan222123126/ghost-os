package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
)

type turnOutcome struct {
	output string
	done   bool
}

func (a *Agent) runTurn(ctx context.Context, turn int, state *agentRunState) (turnOutcome, error) {
	a.lastTurn = turn
	resp, err := state.complete(ctx, turn)
	if err != nil {
		return turnOutcome{}, state.terminalRunError(ctx, turn, completionTurnError(state.traceID, turn, err))
	}
	return a.handleTurnResponse(ctx, turn, resp, state)
}

func (a *Agent) handleTurnResponse(
	ctx context.Context,
	turn int,
	resp *llm.CompletionResponse,
	state *agentRunState,
) (turnOutcome, error) {
	switch resp.FinishReason {
	case llm.FinishStop:
		return a.handleStopTurn(ctx, turn, resp, state)
	case llm.FinishToolCalls:
		return a.handleToolCallTurn(ctx, turn, resp, state)
	case llm.FinishLength:
		return a.handleLengthTurn(ctx, turn, resp, state)
	default:
		return turnOutcome{}, state.terminalRunError(
			ctx,
			turn,
			unsupportedFinishReasonError(state.traceID, turn, resp.FinishReason),
		)
	}
}

func (a *Agent) handleStopTurn(
	ctx context.Context,
	turn int,
	resp *llm.CompletionResponse,
	state *agentRunState,
) (turnOutcome, error) {
	acceptAssistantTurn(state.history, resp)
	a.commitTurn(state.history)
	output := resp.Message.Text
	if err := state.events.terminalSuccess(ctx, state.traceID, turn, output, state.lifecycle); err != nil {
		return turnOutcome{}, err
	}
	return turnOutcome{output: output, done: true}, nil
}

func (a *Agent) handleLengthTurn(
	ctx context.Context,
	turn int,
	resp *llm.CompletionResponse,
	state *agentRunState,
) (turnOutcome, error) {
	content := strings.TrimSpace(resp.Message.Text)
	if content == "" {
		return turnOutcome{}, state.terminalRunError(
			ctx,
			turn,
			emptyLengthResponseError(state.traceID, turn, resp.FinishReason),
		)
	}

	acceptAssistantTurn(state.history, resp)
	a.commitTurn(state.history)
	if err := state.events.terminalSuccess(ctx, state.traceID, turn, content, state.lifecycle); err != nil {
		return turnOutcome{}, err
	}
	return turnOutcome{output: content, done: true}, nil
}

func (a *Agent) handleToolCallTurn(
	ctx context.Context,
	turn int,
	resp *llm.CompletionResponse,
	state *agentRunState,
) (turnOutcome, error) {
	sanitizedMsg, issues, err := state.prepareToolCallTurn(ctx, turn, resp.Message)
	if err != nil {
		return turnOutcome{}, err
	}
	if len(sanitizedMsg.ToolCalls) == 0 {
		return turnOutcome{}, state.recordNonExecutableToolCallTurn(ctx, turn, resp.Message, issues)
	}

	acceptAssistantTurn(state.history, sanitizedToolCallResponse(resp, sanitizedMsg, len(issues) > 0))
	stats, err := state.toolCalls.execute(ctx, state.traceID, turn, sanitizedMsg.ToolCalls)
	if err != nil {
		return turnOutcome{}, a.handleToolCallExecutionError(ctx, turn, err, state)
	}
	return turnOutcome{}, state.finalizeToolCallTurn(ctx, turn, stats)
}

func (a *Agent) handleToolCallExecutionError(
	ctx context.Context,
	turn int,
	err error,
	state *agentRunState,
) error {
	if isEventEmitError(err) {
		return err
	}
	if commitsPartialToolCallTurn(err) {
		a.commitTurn(state.history)
		return err
	}
	return state.terminalRunError(ctx, turn, toolCallTurnError(state.traceID, turn, err))
}

func (state *agentRunState) prepareToolCallTurn(
	ctx context.Context,
	turn int,
	msg llm.Message,
) (llm.Message, []invalidToolCallIssue, error) {
	sanitizedMsg, issues := sanitizeAssistantToolCalls(msg)
	if len(issues) == 0 {
		return sanitizedMsg, nil, nil
	}
	if err := state.toolCalls.reportInvalidCalls(ctx, state.traceID, turn, issues); err != nil {
		return llm.Message{}, nil, err
	}
	state.history.SetConversationState(llm.ConversationState{})
	return sanitizedMsg, issues, nil
}

func (state *agentRunState) recordNonExecutableToolCallTurn(
	ctx context.Context,
	turn int,
	msg llm.Message,
	issues []invalidToolCallIssue,
) error {
	state.history.Append(invalidToolCallAssistantMessage(msg, issues))
	state.consecutiveNonExecutableToolCallTurns++
	if state.consecutiveNonExecutableToolCallTurns < maxConsecutiveNonExecutableToolCallTurns {
		return nil
	}
	return state.terminalRunError(ctx, turn, repeatedNonExecutableToolCallError(state.traceID, turn))
}

func (state *agentRunState) finalizeToolCallTurn(
	ctx context.Context,
	turn int,
	stats toolCallTurnStats,
) error {
	if stats.nonExecutable() {
		state.consecutiveNonExecutableToolCallTurns++
	} else {
		state.consecutiveNonExecutableToolCallTurns = 0
	}
	if state.consecutiveNonExecutableToolCallTurns < maxConsecutiveNonExecutableToolCallTurns {
		return nil
	}
	return state.terminalRunError(ctx, turn, repeatedNonExecutableToolCallError(state.traceID, turn))
}

func sanitizedToolCallResponse(
	resp *llm.CompletionResponse,
	msg llm.Message,
	clearConversationState bool,
) *llm.CompletionResponse {
	if resp == nil {
		return nil
	}
	cloned := *resp
	cloned.Message = msg
	if clearConversationState {
		cloned.ConversationState = llm.ConversationState{}
	}
	return &cloned
}

func commitsPartialToolCallTurn(err error) bool {
	var awaitingErr *ErrAwaitingHuman
	if errors.As(err, &awaitingErr) {
		return true
	}

	var handoffErr *ErrIterationHandoff
	return errors.As(err, &handoffErr)
}

func completionTurnError(traceID string, turn int, err error) error {
	return fmt.Errorf("trace_id=%s turn=%d complete_once: %w", traceID, turn, err)
}

func toolCallTurnError(traceID string, turn int, err error) error {
	return fmt.Errorf("trace_id=%s turn=%d handle_tool_calls: %w", traceID, turn, err)
}

func emptyLengthResponseError(traceID string, turn int, finishReason llm.FinishReason) error {
	return fmt.Errorf("trace_id=%s turn=%d finish_reason=%q with empty content", traceID, turn, finishReason)
}

func unsupportedFinishReasonError(traceID string, turn int, finishReason llm.FinishReason) error {
	return fmt.Errorf("trace_id=%s turn=%d unsupported finish_reason: %q", traceID, turn, finishReason)
}

func repeatedNonExecutableToolCallError(traceID string, turn int) error {
	return fmt.Errorf(
		"trace_id=%s turn=%d repeated non-executable tool_calls; aborting tool-call loop",
		traceID,
		turn,
	)
}
