package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
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
	if history == nil {
		history = NewHistory("")
	}
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
		stepID, err := streaming.ToolStepID(turn, toolIndex)
		if err != nil {
			return stats, err
		}
		rawToolCallID := strings.TrimSpace(call.ID)
		rawToolName := strings.TrimSpace(call.Name)
		if err := e.events.toolCallStarted(ctx, traceID, turn, stepID, rawToolName, rawToolCallID); err != nil {
			return stats, err
		}

		toolCallID, toolName, args, err := validateToolCall(call)
		if err != nil {
			fmt.Fprintf(e.stderr, "[%s] invalid_tool_call: id=%q name=%q error=%v\n", traceID, strings.TrimSpace(call.ID), strings.TrimSpace(call.Name), err)
			resolvedToolCallID := coalesceToolCallID(toolCallID, rawToolCallID)
			if resolvedToolCallID != "" {
				resolvedToolName := coalesceToolName(toolName, rawToolName, "invalid_tool_call")
				appendToolResult(e.history, resolvedToolCallID, resolvedToolName, traceID, "", err, nil)
			}
			if emitErr := e.events.toolCallFinished(ctx, traceID, turn, stepID, coalesceToolName(toolName, rawToolName, "invalid_tool_call"), resolvedToolCallID, "error", err); emitErr != nil {
				return stats, emitErr
			}
			continue
		}

		fmt.Fprintf(e.stderr, "[%s] tool_call: %s %s\n", traceID, toolName, summarizeToolArgs(args))

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
		if sess := tools.SessionFromContext(ctx); sess != nil {
			sess.NoteDynamicToolCall(toolName)
		}
		stats.executed++
		output, execErr := e.executeToolSafely(toolCtx, tool, args, traceID, toolName)
		if execErr != nil {
			toolErr := fmt.Errorf("tool %q error: %w", toolName, execErr)
			appendToolResult(e.history, toolCallID, toolName, traceID, "", toolErr, nil)
			if emitErr := e.events.toolCallFinished(ctx, traceID, turn, stepID, toolName, toolCallID, "error", toolErr); emitErr != nil {
				return stats, emitErr
			}
			continue
		}

		output, meta, postProcessErr := e.postProcessToolResultSafely(tool, output, traceID, toolName)
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
		if meta.Iteration != nil {
			return stats, &ErrIterationHandoff{
				Did:            strings.TrimSpace(meta.Iteration.Did),
				Remaining:      strings.TrimSpace(meta.Iteration.Remaining),
				Completed:      meta.Iteration.Completed,
				FinalMessage:   strings.TrimSpace(meta.Iteration.FinalMessage),
				FinalChangeLog: strings.TrimSpace(meta.Iteration.FinalChangeLog),
			}
		}

		appendToolResult(e.history, toolCallID, toolName, traceID, output, nil, meta.Content)
	}

	return stats, nil
}

func (e toolCallExecutor) reportInvalidCalls(ctx context.Context, traceID string, turn int, issues []invalidToolCallIssue) error {
	for _, issue := range issues {
		stepID, err := streaming.ToolStepID(turn, issue.index)
		if err != nil {
			return err
		}
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

func (e toolCallExecutor) executeToolSafely(ctx context.Context, tool tools.Tool, args json.RawMessage, traceID string, toolName string) (output string, err error) {
	if tool == nil {
		return "", errors.New("tool is nil")
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("execute panic: %v", recovered)
			fmt.Fprintf(e.stderr, "[%s] tool_panic: tool=%s phase=execute error=%v\n", traceID, toolName, err)
		}
	}()

	return tool.Execute(ctx, args, traceID)
}

func (e toolCallExecutor) postProcessToolResultSafely(tool tools.Tool, output string, traceID string, toolName string) (processed string, meta tools.ExecuteMeta, err error) {
	if tool == nil {
		return "", tools.ExecuteMeta{}, errors.New("tool is nil")
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("post-process panic: %v", recovered)
			processed = ""
			meta = tools.ExecuteMeta{}
			fmt.Fprintf(e.stderr, "[%s] tool_panic: tool=%s phase=post_process error=%v\n", traceID, toolName, err)
		}
	}()

	return tools.PostProcessExecuteResult(tool, output, traceID)
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

func summarizeToolArgs(args json.RawMessage) string {
	var decoded map[string]any
	if err := json.Unmarshal(args, &decoded); err != nil {
		return "args_keys=<invalid>"
	}
	if len(decoded) == 0 {
		return "args_keys=[]"
	}

	keys := make([]string, 0, len(decoded))
	for key := range decoded {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return fmt.Sprintf("args_keys=[%s]", strings.Join(keys, ","))
}
