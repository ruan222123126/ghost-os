package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/tools"
)

func (e toolCallExecutor) execute(ctx context.Context, traceID string, turn int, calls []indexedToolCall) (toolCallTurnStats, error) {
	stats := toolCallTurnStats{totalCalls: len(calls)}
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

func (e toolCallExecutor) executeSingle(
	ctx context.Context,
	traceID string,
	turn int,
	toolName string,
	toolCallID string,
	args json.RawMessage,
) (toolCallOutcome, error) {
	return e.executeSingleWithStepIndex(ctx, traceID, turn, 0, toolName, toolCallID, args)
}

func (e toolCallExecutor) executeSingleWithStepIndex(
	ctx context.Context,
	traceID string,
	turn int,
	stepIndex int,
	toolName string,
	toolCallID string,
	args json.RawMessage,
) (toolCallOutcome, error) {
	step, err := e.startExplicitToolCall(ctx, traceID, turn, stepIndex, toolName, toolCallID)
	if err != nil {
		return toolCallOutcome{}, err
	}
	resolved, outcome, handled, err := e.resolveExplicitToolCall(ctx, traceID, step, args)
	if err != nil || handled {
		return outcome, err
	}
	return e.runResolvedToolCall(ctx, traceID, resolved)
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
		outcome, err := e.finishAwaitingHuman(ctx, traceID, resolved, meta.AwaitingHuman)
		outcome.output = output
		outcome.meta = meta
		return outcome, err
	}
	if meta.Iteration != nil {
		return toolCallOutcome{
			executed: true,
			stopErr:  newIterationHandoffError(meta.Iteration),
			output:   output,
			meta:     meta,
		}, nil
	}

	appendToolResult(e.history, resolved.toolCallID, resolved.toolName, traceID, output, nil, meta.Content)
	return toolCallOutcome{executed: true, output: output, meta: meta}, nil
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
