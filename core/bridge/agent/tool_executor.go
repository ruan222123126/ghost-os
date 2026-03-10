package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

var errToolCallIDRequired = errors.New("tool_call.id is empty")

type toolCallTurnStats struct {
	totalCalls int
	executed   int
}

func (s toolCallTurnStats) nonExecutable() bool {
	return s.totalCalls > 0 && s.executed == 0
}

// toolCallExecutor 负责单回合工具执行、stderr 记录、事件发射与历史回写。
type toolCallExecutor struct {
	tools   ToolCatalog
	history *History
	stderr  io.Writer
	events  agentEventEmitter
}

func newToolCallExecutor(toolCatalog ToolCatalog, history *History, stderr io.Writer, events agentEventEmitter) toolCallExecutor {
	if stderr == nil {
		stderr = os.Stderr
	}
	return toolCallExecutor{
		tools:   toolCatalog,
		history: history,
		stderr:  stderr,
		events:  events,
	}
}

func (e toolCallExecutor) execute(ctx context.Context, traceID string, turn int, calls []llm.ToolCall) (toolCallTurnStats, error) {
	stats := toolCallTurnStats{totalCalls: len(calls)}
	if len(calls) == 0 {
		return stats, errors.New("finish_reason=tool_calls but tool_calls is empty")
	}

	for toolIndex, call := range calls {
		stepID := ToolStepID(turn, toolIndex)
		rawToolCallID := strings.TrimSpace(call.ID)
		rawToolName := strings.TrimSpace(call.Name)
		if err := e.events.toolCallStarted(ctx, traceID, turn, stepID, rawToolName, rawToolCallID); err != nil {
			return stats, err
		}

		toolCallID, toolName, args, err := validateToolCall(call)
		if err != nil {
			fmt.Fprintf(e.stderr, "[%s] invalid_tool_call: id=%q name=%q error=%v\n", traceID, strings.TrimSpace(call.ID), strings.TrimSpace(call.Name), err)
			appendToolResult(e.history, toolCallID, toolName, traceID, "", err, nil)
			if emitErr := e.events.toolCallFinished(ctx, traceID, turn, stepID, coalesceToolName(toolName, rawToolName), coalesceToolCallID(toolCallID, rawToolCallID), "error", err); emitErr != nil {
				return stats, emitErr
			}
			continue
		}

		fmt.Fprintf(e.stderr, "[%s] tool_call: %s args=%s\n", traceID, toolName, string(args))

		tool := e.tools.Get(toolName)
		if tool == nil {
			toolErr := fmt.Errorf("tool %q not found", toolName)
			appendToolResult(e.history, toolCallID, toolName, traceID, "", toolErr, nil)
			if emitErr := e.events.toolCallFinished(ctx, traceID, turn, stepID, toolName, toolCallID, "error", toolErr); emitErr != nil {
				return stats, emitErr
			}
			continue
		}

		toolCtx := tools.WithToolCallID(ctx, toolCallID)
		stats.executed++
		output, execErr := tool.Execute(toolCtx, args, traceID)
		if execErr != nil {
			toolErr := fmt.Errorf("tool %q error: %w", toolName, execErr)
			appendToolResult(e.history, toolCallID, toolName, traceID, "", toolErr, nil)
			if emitErr := e.events.toolCallFinished(ctx, traceID, turn, stepID, toolName, toolCallID, "error", toolErr); emitErr != nil {
				return stats, emitErr
			}
			continue
		}

		output, meta, postProcessErr := tools.PostProcessExecuteResult(tool, output, traceID)
		if postProcessErr != nil {
			toolErr := fmt.Errorf("tool %q post-process error: %w", toolName, postProcessErr)
			appendToolResult(e.history, toolCallID, toolName, traceID, "", toolErr, nil)
			if emitErr := e.events.toolCallFinished(ctx, traceID, turn, stepID, toolName, toolCallID, "error", toolErr); emitErr != nil {
				return stats, emitErr
			}
			continue
		}
		if emitErr := e.events.toolCallFinished(ctx, traceID, turn, stepID, toolName, toolCallID, "success", nil); emitErr != nil {
			return stats, emitErr
		}
		if meta.AwaitingHuman != nil {
			questionID := strings.TrimSpace(meta.AwaitingHuman.QuestionID)
			prompt := strings.TrimSpace(meta.AwaitingHuman.Prompt)
			selectionMode := strings.TrimSpace(meta.AwaitingHuman.SelectionMode)
			options := append([]tools.AskHumanOption(nil), meta.AwaitingHuman.Options...)
			if emitErr := e.events.awaitingHuman(ctx, traceID, turn, stepID, toolName, toolCallID, questionID, prompt, selectionMode, options); emitErr != nil {
				return stats, emitErr
			}
			return stats, &ErrAwaitingHuman{
				QuestionID:    questionID,
				Prompt:        prompt,
				SelectionMode: selectionMode,
				Options:       options,
			}
		}

		appendToolResult(e.history, toolCallID, toolName, traceID, output, nil, meta.Content)
	}

	return stats, nil
}

func (e toolCallExecutor) reportInvalidCalls(ctx context.Context, traceID string, turn int, issues []invalidToolCallIssue) error {
	for _, issue := range issues {
		stepID := ToolStepID(turn, issue.index)
		rawToolCallID := strings.TrimSpace(issue.call.ID)
		rawToolName := strings.TrimSpace(issue.call.Name)
		if err := e.events.toolCallStarted(ctx, traceID, turn, stepID, rawToolName, rawToolCallID); err != nil {
			return err
		}
		fmt.Fprintf(e.stderr, "[%s] invalid_tool_call: id=%q name=%q error=%v\n", traceID, rawToolCallID, rawToolName, issue.err)
		if err := e.events.toolCallFinished(ctx, traceID, turn, stepID, rawToolName, rawToolCallID, "error", issue.err); err != nil {
			return err
		}
	}
	return nil
}

// validateToolCall 做最小输入护栏：id/name 必填，arguments 必须是 JSON object。
func validateToolCall(call llm.ToolCall) (toolCallID string, toolName string, args json.RawMessage, err error) {
	toolCallID = strings.TrimSpace(call.ID)
	if toolCallID == "" {
		return "", "", nil, errToolCallIDRequired
	}

	toolName = strings.TrimSpace(call.Name)
	if toolName == "" {
		return toolCallID, "", nil, errors.New("tool_call.name is empty")
	}

	args, err = normalizedToolArguments(call.Arguments)
	if err != nil {
		return toolCallID, toolName, nil, err
	}

	return toolCallID, toolName, args, nil
}

// normalizedToolArguments 只接受非空 JSON object。
func normalizedToolArguments(raw json.RawMessage) (json.RawMessage, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, errors.New("tool_call.arguments is empty")
	}

	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("tool_call.arguments must be valid JSON object: %w", err)
	}
	objectValue, ok := decoded.(map[string]any)
	if !ok {
		return nil, errors.New("tool_call.arguments must be a JSON object")
	}
	normalized, err := json.Marshal(objectValue)
	if err != nil {
		return nil, fmt.Errorf("normalize tool_call.arguments: %w", err)
	}
	return normalized, nil
}

func coalesceToolName(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func coalesceToolCallID(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
