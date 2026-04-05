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
	executed                  bool
	stopErr                   error
	output                    string
	meta                      tools.ExecuteMeta
	browserSessionInvalid     bool
	browserSessionInvalidInfo string
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

func (e toolCallExecutor) startExplicitToolCall(
	ctx context.Context,
	traceID string,
	turn int,
	stepIndex int,
	toolName string,
	toolCallID string,
) (toolCallStep, error) {
	stepID, err := streaming.ToolStepID(turn, stepIndex)
	if err != nil {
		return toolCallStep{}, err
	}
	step := toolCallStep{
		turn:          turn,
		stepID:        stepID,
		rawToolCallID: strings.TrimSpace(toolCallID),
		rawToolName:   strings.TrimSpace(toolName),
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

func (e toolCallExecutor) resolveExplicitToolCall(
	ctx context.Context,
	traceID string,
	step toolCallStep,
	args json.RawMessage,
) (resolvedToolCall, toolCallOutcome, bool, error) {
	toolCallID := strings.TrimSpace(step.rawToolCallID)
	if toolCallID == "" {
		return e.finishExplicitToolCallValidationError(ctx, traceID, step, "", "", errToolCallIDRequired)
	}
	toolName := strings.TrimSpace(step.rawToolName)
	if toolName == "" {
		return e.finishExplicitToolCallValidationError(ctx, traceID, step, toolCallID, "", errors.New("tool_call.name is empty"))
	}
	normalizedArgs, err := normalizedToolArguments(args)
	if err != nil {
		return e.finishExplicitToolCallValidationError(ctx, traceID, step, toolCallID, toolName, err)
	}
	fmt.Fprintf(e.stderr, "[%s] tool_call: %s %s\n", traceID, toolName, summarizeToolArgs(normalizedArgs))
	tool := e.tools.Get(toolName)
	if tool == nil {
		outcome, finishErr := e.finishMissingToolCall(ctx, traceID, step, toolCallID, toolName)
		return resolvedToolCall{}, outcome, true, finishErr
	}
	return resolvedToolCall{
		step:       step,
		toolCallID: toolCallID,
		toolName:   toolName,
		args:       normalizedArgs,
		tool:       tool,
	}, toolCallOutcome{}, false, nil
}

func (e toolCallExecutor) finishExplicitToolCallValidationError(
	ctx context.Context,
	traceID string,
	step toolCallStep,
	toolCallID string,
	toolName string,
	callErr error,
) (resolvedToolCall, toolCallOutcome, bool, error) {
	fmt.Fprintf(e.stderr, "[%s] invalid_tool_call: id=%q name=%q error=%v\n", traceID, step.rawToolCallID, step.rawToolName, callErr)

	resolvedToolCallID := coalesceToolCallID(toolCallID, step.rawToolCallID)
	resolvedToolName := coalesceToolName(toolName, step.rawToolName, "invalid_tool_call")
	if err := e.events.toolCallFinished(ctx, traceID, step.turn, step.stepID, resolvedToolName, resolvedToolCallID, "error", callErr); err != nil {
		return resolvedToolCall{}, toolCallOutcome{}, true, err
	}
	return resolvedToolCall{}, toolCallOutcome{}, true, callErr
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
