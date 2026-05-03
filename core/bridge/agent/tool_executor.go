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
var debugToolCallLogs = isTruthyEnv("GHOST_BRIDGE_DEBUG") || isTruthyEnv("GHOST_DEBUG") || isTruthyEnv("DEBUG")

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

func (e toolCallExecutor) reportInvalidCalls(ctx context.Context, traceID string, turn int, issues []invalidToolCallIssue) error {
	for _, issue := range issues {
		stepID, err := streaming.ToolStepID(turn, issue.index)
		if err != nil {
			return err
		}
		rawToolCallID := strings.TrimSpace(issue.call.ID)
		rawToolName := strings.TrimSpace(issue.call.Name)
		if err := e.events.toolCallStarted(ctx, traceID, turn, stepID, rawToolName, rawToolCallID, string(issue.call.Arguments)); err != nil {
			return err
		}
		fmt.Fprintf(e.stderr, "[%s] invalid_tool_call: id=%q name=%q error=%v\n", traceID, rawToolCallID, rawToolName, issue.err)
		if err := e.events.toolCallFinished(ctx, traceID, turn, stepID, rawToolName, rawToolCallID, "error", issue.err, ""); err != nil {
			return err
		}
	}
	return nil
}

func (e toolCallExecutor) executeToolSafely(ctx context.Context, tool tools.Tool, args json.RawMessage, traceID string, toolName string) (output string, err error) {
	if tool == nil {
		return "", errors.New("tool is nil")
	}
	if cancelErr := contextCanceled(ctx); cancelErr != nil {
		return "", cancelErr
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("execute panic: %v", recovered)
			fmt.Fprintf(e.stderr, "[%s] tool_panic: tool=%s phase=execute error=%v\n", traceID, toolName, err)
		}
	}()

	return tool.Execute(ctx, args, traceID)
}

func (e toolCallExecutor) logToolCall(traceID string, toolName string, args json.RawMessage) {
	if !debugToolCallLogs {
		return
	}
	fmt.Fprintf(e.stderr, "[%s] tool_call: %s %s\n", traceID, toolName, summarizeToolArgs(args))
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

func contextCanceled(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func isTruthyEnv(key string) bool {
	value := strings.TrimSpace(os.Getenv(strings.TrimSpace(key)))
	if value == "" {
		return false
	}

	switch strings.ToLower(value) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
