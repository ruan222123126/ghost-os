package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
)

type toolCallStep struct {
	call          llm.ToolCall
	turn          int
	stepID        string
	rawToolCallID string
	rawToolName   string
}

type resolvedToolCall struct {
	step       toolCallStep
	toolCallID string
	toolName   string
	args       json.RawMessage
	tool       tools.Tool
}

type toolCallOutcome struct {
	executed bool
	stopErr  error
}

func (e toolCallExecutor) execute(ctx context.Context, traceID string, turn int, calls []indexedToolCall) (toolCallTurnStats, error) {
	stats := toolCallTurnStats{totalCalls: len(calls)}
	if len(calls) == 0 {
		return stats, errors.New("finish_reason=tool_calls but tool_calls is empty")
	}

	for _, indexedCall := range calls {
		outcome, err := e.executeIndexedCall(ctx, traceID, turn, indexedCall)
		if err != nil {
			return stats, err
		}
		if outcome.executed {
			stats.executed++
		}
		if outcome.stopErr != nil {
			return stats, outcome.stopErr
		}
	}
	return stats, nil
}

func (e toolCallExecutor) executeIndexedCall(ctx context.Context, traceID string, turn int, indexedCall indexedToolCall) (toolCallOutcome, error) {
	step, err := e.startToolCall(ctx, traceID, turn, indexedCall)
	if err != nil {
		return toolCallOutcome{}, err
	}

	resolved, outcome, handled, err := e.resolveToolCall(ctx, traceID, step)
	if err != nil || handled {
		return outcome, err
	}
	return e.runResolvedToolCall(ctx, traceID, resolved)
}

func (e toolCallExecutor) startToolCall(ctx context.Context, traceID string, turn int, indexedCall indexedToolCall) (toolCallStep, error) {
	stepID, err := streaming.ToolStepID(turn, indexedCall.index)
	if err != nil {
		return toolCallStep{}, err
	}

	call := indexedCall.call
	step := toolCallStep{
		call:          call,
		turn:          turn,
		stepID:        stepID,
		rawToolCallID: strings.TrimSpace(call.ID),
		rawToolName:   strings.TrimSpace(call.Name),
	}
	if err := e.events.toolCallStarted(ctx, traceID, turn, stepID, step.rawToolName, step.rawToolCallID); err != nil {
		return toolCallStep{}, err
	}
	return step, nil
}

func (e toolCallExecutor) resolveToolCall(ctx context.Context, traceID string, step toolCallStep) (resolvedToolCall, toolCallOutcome, bool, error) {
	toolCallID, toolName, args, err := validateToolCall(step.call)
	if err != nil {
		outcome, finishErr := e.finishInvalidToolCall(ctx, traceID, step, toolCallID, toolName, err)
		return resolvedToolCall{}, outcome, true, finishErr
	}

	fmt.Fprintf(e.stderr, "[%s] tool_call: %s %s\n", traceID, toolName, summarizeToolArgs(args))
	tool := e.tools.Get(toolName)
	if tool == nil {
		outcome, finishErr := e.finishMissingToolCall(ctx, traceID, step, toolCallID, toolName)
		return resolvedToolCall{}, outcome, true, finishErr
	}

	return resolvedToolCall{
		step:       step,
		toolCallID: toolCallID,
		toolName:   toolName,
		args:       args,
		tool:       tool,
	}, toolCallOutcome{}, false, nil
}

func (e toolCallExecutor) finishInvalidToolCall(ctx context.Context, traceID string, step toolCallStep, toolCallID string, toolName string, callErr error) (toolCallOutcome, error) {
	fmt.Fprintf(e.stderr, "[%s] invalid_tool_call: id=%q name=%q error=%v\n", traceID, step.rawToolCallID, step.rawToolName, callErr)

	resolvedToolCallID := coalesceToolCallID(toolCallID, step.rawToolCallID)
	resolvedToolName := coalesceToolName(toolName, step.rawToolName, "invalid_tool_call")
	if resolvedToolCallID != "" {
		appendToolResult(e.history, resolvedToolCallID, resolvedToolName, traceID, "", callErr, nil)
	}
	if err := e.events.toolCallFinished(ctx, traceID, step.turn, step.stepID, resolvedToolName, resolvedToolCallID, "error", callErr); err != nil {
		return toolCallOutcome{}, err
	}
	return toolCallOutcome{}, nil
}

func (e toolCallExecutor) finishMissingToolCall(ctx context.Context, traceID string, step toolCallStep, toolCallID string, toolName string) (toolCallOutcome, error) {
	toolErr := fmt.Errorf("tool %q not found", toolName)
	appendToolResult(e.history, toolCallID, toolName, traceID, "", toolErr, nil)
	if err := e.events.toolCallFinished(ctx, traceID, step.turn, step.stepID, toolName, toolCallID, "error", toolErr); err != nil {
		return toolCallOutcome{}, err
	}
	return toolCallOutcome{}, nil
}

func (e toolCallExecutor) runResolvedToolCall(ctx context.Context, traceID string, resolved resolvedToolCall) (toolCallOutcome, error) {
	toolCtx := tools.WithToolCallID(ctx, resolved.toolCallID)
	if sess := tools.SessionFromContext(ctx); sess != nil {
		sess.NoteDynamicToolCall(resolved.toolName)
	}

	output, execErr := e.executeToolSafely(toolCtx, resolved.tool, resolved.args, traceID, resolved.toolName)
	if execErr != nil {
		toolErr := fmt.Errorf("tool %q error: %w", resolved.toolName, execErr)
		return e.finishToolFailure(ctx, traceID, resolved, toolErr)
	}

	output, meta, postProcessErr := e.postProcessToolResultSafely(resolved.tool, output, traceID, resolved.toolName)
	if postProcessErr != nil {
		toolErr := fmt.Errorf("tool %q post-process error: %w", resolved.toolName, postProcessErr)
		return e.finishToolFailure(ctx, traceID, resolved, toolErr)
	}
	return e.finishSuccessfulToolCall(ctx, traceID, resolved, output, meta)
}

func (e toolCallExecutor) finishToolFailure(ctx context.Context, traceID string, resolved resolvedToolCall, toolErr error) (toolCallOutcome, error) {
	appendToolResult(e.history, resolved.toolCallID, resolved.toolName, traceID, "", toolErr, nil)
	if err := e.events.toolCallFinished(ctx, traceID, resolved.step.turn, resolved.step.stepID, resolved.toolName, resolved.toolCallID, "error", toolErr); err != nil {
		return toolCallOutcome{}, err
	}
	return toolCallOutcome{executed: true}, nil
}

func (e toolCallExecutor) finishSuccessfulToolCall(ctx context.Context, traceID string, resolved resolvedToolCall, output string, meta tools.ExecuteMeta) (toolCallOutcome, error) {
	if err := e.events.toolCallFinished(ctx, traceID, resolved.step.turn, resolved.step.stepID, resolved.toolName, resolved.toolCallID, "success", nil); err != nil {
		return toolCallOutcome{}, err
	}
	if meta.AwaitingHuman != nil {
		return e.finishAwaitingHuman(ctx, traceID, resolved, meta.AwaitingHuman)
	}
	if meta.Iteration != nil {
		return toolCallOutcome{executed: true, stopErr: newIterationHandoffError(meta.Iteration)}, nil
	}

	appendToolResult(e.history, resolved.toolCallID, resolved.toolName, traceID, output, nil, meta.Content)
	return toolCallOutcome{executed: true}, nil
}

func (e toolCallExecutor) finishAwaitingHuman(ctx context.Context, traceID string, resolved resolvedToolCall, awaiting *tools.AwaitingHumanSignal) (toolCallOutcome, error) {
	questionID := strings.TrimSpace(awaiting.QuestionID)
	prompt := strings.TrimSpace(awaiting.Prompt)
	selectionMode := strings.TrimSpace(awaiting.SelectionMode)
	options := append([]tools.AskHumanOption(nil), awaiting.Options...)
	if err := e.events.awaitingHuman(ctx, traceID, resolved.step.turn, resolved.step.stepID, resolved.toolName, resolved.toolCallID, questionID, prompt, selectionMode, options); err != nil {
		return toolCallOutcome{}, err
	}
	return toolCallOutcome{
		executed: true,
		stopErr: &ErrAwaitingHuman{
			QuestionID:    questionID,
			Prompt:        prompt,
			SelectionMode: selectionMode,
			Options:       options,
		},
	}, nil
}

func newIterationHandoffError(signal *tools.IterationHandoffSignal) *ErrIterationHandoff {
	if signal == nil {
		return &ErrIterationHandoff{}
	}
	return &ErrIterationHandoff{
		Did:            strings.TrimSpace(signal.Did),
		Remaining:      strings.TrimSpace(signal.Remaining),
		Completed:      signal.Completed,
		FinalMessage:   strings.TrimSpace(signal.FinalMessage),
		FinalChangeLog: strings.TrimSpace(signal.FinalChangeLog),
	}
}
