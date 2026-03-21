package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
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
	if len(resp.Message.ToolCalls) > 0 {
		if state.strictToolCallProtocol {
			return turnOutcome{}, state.terminalRunError(
				ctx,
				turn,
				strictToolCallProtocolError(state.traceID, turn),
			)
		}
		return a.handleToolCallTurn(ctx, turn, resp, state)
	}

	switch resp.FinishReason {
	case llm.FinishStop:
		return a.handleStopTurn(ctx, turn, resp, state)
	case llm.FinishToolCalls:
		if state.strictToolCallProtocol {
			return turnOutcome{}, state.terminalRunError(
				ctx,
				turn,
				strictToolCallProtocolError(state.traceID, turn),
			)
		}
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
	handled, outcome, err := a.handleGraphQLTextTurn(ctx, turn, resp, state)
	if err != nil || handled {
		return outcome, err
	}
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
	handled, outcome, err := a.handleGraphQLTextTurn(ctx, turn, resp, state)
	if err != nil || handled {
		return outcome, err
	}

	acceptAssistantTurn(state.history, resp)
	a.commitTurn(state.history)
	if err := state.events.terminalSuccess(ctx, state.traceID, turn, content, state.lifecycle); err != nil {
		return turnOutcome{}, err
	}
	return turnOutcome{output: content, done: true}, nil
}

func (a *Agent) handleGraphQLTextTurn(
	ctx context.Context,
	turn int,
	resp *llm.CompletionResponse,
	state *agentRunState,
) (bool, turnOutcome, error) {
	if state.graphQL == nil || resp == nil || strings.TrimSpace(resp.Message.Text) == "" {
		return false, turnOutcome{}, nil
	}
	result, err := state.graphQL.Execute(ctx, resp.Message.Text, state.traceID)
	if !result.Recognized {
		return false, turnOutcome{}, nil
	}
	stepID, err := streaming.ToolStepID(turn, 0)
	if err != nil {
		return true, turnOutcome{}, err
	}
	if err := state.events.toolCallStarted(ctx, state.traceID, turn, stepID, "graphql_text", "graphql-text"); err != nil {
		return true, turnOutcome{}, err
	}
	acceptAssistantTurn(state.history, resp)
	if err != nil {
		if emitErr := state.events.toolCallFinished(ctx, state.traceID, turn, stepID, "graphql_text", "graphql-text", "error", err); emitErr != nil {
			return true, turnOutcome{}, emitErr
		}
		return true, turnOutcome{}, state.terminalRunError(ctx, turn, graphQLTextTurnError(state.traceID, turn, err))
	}
	if emitErr := state.events.toolCallFinished(ctx, state.traceID, turn, stepID, "graphql_text", "graphql-text", "success", nil); emitErr != nil {
		return true, turnOutcome{}, emitErr
	}
	if result.Meta.AwaitingHuman != nil {
		if err := emitGraphQLAwaitingHuman(ctx, turn, state, stepID, result.Meta.AwaitingHuman); err != nil {
			return true, turnOutcome{}, err
		}
		a.commitTurn(state.history)
		return true, turnOutcome{}, &ErrAwaitingHuman{
			QuestionID:    strings.TrimSpace(result.Meta.AwaitingHuman.QuestionID),
			Prompt:        strings.TrimSpace(result.Meta.AwaitingHuman.Prompt),
			SelectionMode: strings.TrimSpace(result.Meta.AwaitingHuman.SelectionMode),
			Options:       append([]tools.AskHumanOption(nil), result.Meta.AwaitingHuman.Options...),
		}
	}
	appendGraphQLExecutionFeedback(state.history, result.Output)
	a.commitTurn(state.history)
	return true, turnOutcome{}, nil
}

func emitGraphQLAwaitingHuman(
	ctx context.Context,
	turn int,
	state *agentRunState,
	stepID string,
	awaiting *tools.AwaitingHumanSignal,
) error {
	if state == nil || awaiting == nil {
		return nil
	}
	if err := state.events.awaitingHuman(
		ctx,
		state.traceID,
		turn,
		stepID,
		tools.GraphQLTextMutationToolName,
		"graphql-text",
		strings.TrimSpace(awaiting.QuestionID),
		strings.TrimSpace(awaiting.Prompt),
		strings.TrimSpace(awaiting.SelectionMode),
		append([]tools.AskHumanOption(nil), awaiting.Options...),
	); err != nil {
		return err
	}
	return nil
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

func graphQLTextTurnError(traceID string, turn int, err error) error {
	return fmt.Errorf("trace_id=%s turn=%d handle_graphql_text: %w", traceID, turn, err)
}

func emptyLengthResponseError(traceID string, turn int, finishReason llm.FinishReason) error {
	return fmt.Errorf("trace_id=%s turn=%d finish_reason=%q with empty content", traceID, turn, finishReason)
}

func unsupportedFinishReasonError(traceID string, turn int, finishReason llm.FinishReason) error {
	return fmt.Errorf("trace_id=%s turn=%d unsupported finish_reason: %q", traceID, turn, finishReason)
}

func strictToolCallProtocolError(traceID string, turn int) error {
	return fmt.Errorf(
		"trace_id=%s turn=%d strict graphql text mode rejects tool_calls finish reason",
		traceID,
		turn,
	)
}

func repeatedNonExecutableToolCallError(traceID string, turn int) error {
	return fmt.Errorf(
		"trace_id=%s turn=%d repeated non-executable tool_calls; aborting tool-call loop",
		traceID,
		turn,
	)
}
