package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

type Completer interface {
	Complete(context.Context, llm.CompletionRequest) (*llm.CompletionResponse, error)
}

type ToolCatalog interface {
	ToolDefs() []llm.ToolDef
	Get(name string) tools.Tool
}

// Agent 只编排对话循环，不绑定具体 Provider/Tool 实现。
type Agent struct {
	completer Completer
	tools     ToolCatalog
	history   *History
	maxTurns  int
}

func NewAgent(completer Completer, toolCatalog ToolCatalog, systemPrompt string, maxTurns int) *Agent {
	if maxTurns <= 0 {
		maxTurns = 20
	}

	return &Agent{
		completer: completer,
		tools:     toolCatalog,
		history:   NewHistory(systemPrompt),
		maxTurns:  maxTurns,
	}
}

// Run 负责循环与退出条件；单步逻辑拆到私有方法里，便于测试与扩展。
func (a *Agent) Run(ctx context.Context, userMessage string) (string, error) {
	traceID := fmt.Sprintf("agent-%d", time.Now().UnixNano())
	a.appendUserMessage(userMessage)

	for turn := 0; turn < a.maxTurns; turn++ {
		msg, finishReason, err := a.completeOnce(ctx)
		if err != nil {
			return "", fmt.Errorf("trace_id=%s turn=%d complete_once: %w", traceID, turn, err)
		}

		switch finishReason {
		case llm.FinishStop:
			return a.handleAssistantStop(msg), nil
		case llm.FinishToolCalls:
			if err := a.handleToolCalls(ctx, traceID, msg.ToolCalls); err != nil {
				return "", fmt.Errorf("trace_id=%s turn=%d handle_tool_calls: %w", traceID, turn, err)
			}
		case llm.FinishLength:
			content := strings.TrimSpace(msg.Text)
			if content != "" {
				return content, nil
			}
			return "", fmt.Errorf("trace_id=%s turn=%d finish_reason=%q with empty content", traceID, turn, finishReason)
		default:
			return "", fmt.Errorf("trace_id=%s turn=%d unsupported finish_reason: %q", traceID, turn, finishReason)
		}
	}

	return "", fmt.Errorf("trace_id=%s max turns exceeded: %d", traceID, a.maxTurns)
}

// appendUserMessage 只负责把用户输入追加到会话历史。
func (a *Agent) appendUserMessage(userMessage string) {
	a.history.Append(llm.Message{
		Role: llm.RoleUser,
		Text: userMessage,
	})
}

// completeOnce 只做一次模型调用 + assistant 消息落历史。
func (a *Agent) completeOnce(ctx context.Context) (llm.Message, llm.FinishReason, error) {
	resp, err := a.completer.Complete(ctx, llm.CompletionRequest{
		Messages: a.history.Messages(),
		Tools:    a.tools.ToolDefs(),
	})
	if err != nil {
		return llm.Message{}, "", err
	}

	msg := resp.Message
	if msg.Role == "" {
		msg.Role = llm.RoleAssistant
	}
	a.history.Append(msg)

	return msg, resp.FinishReason, nil
}

func (a *Agent) handleAssistantStop(msg llm.Message) string {
	return msg.Text
}

// handleToolCalls 负责执行工具并把结果统一写回历史，供下一轮模型继续推理。
func (a *Agent) handleToolCalls(ctx context.Context, traceID string, calls []llm.ToolCall) error {
	if len(calls) == 0 {
		return errors.New("finish_reason=tool_calls but tool_calls is empty")
	}

	for _, call := range calls {
		toolCallID, toolName, args, err := validateToolCall(call)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[%s] invalid_tool_call: id=%q name=%q error=%v\n", traceID, strings.TrimSpace(call.ID), strings.TrimSpace(call.Name), err)
			a.appendToolResult(toolCallID, toolName, traceID, "", err)
			continue
		}

		fmt.Fprintf(os.Stderr, "[%s] tool_call: %s args=%s\n", traceID, toolName, string(args))

		tool := a.tools.Get(toolName)
		if tool == nil {
			a.appendToolResult(toolCallID, toolName, traceID, "", fmt.Errorf("tool %q not found", toolName))
			continue
		}

		output, execErr := tool.Execute(ctx, args)
		if execErr != nil {
			a.appendToolResult(toolCallID, toolName, traceID, "", fmt.Errorf("tool %q error: %w", toolName, execErr))
			continue
		}

		a.appendToolResult(toolCallID, toolName, traceID, output, nil)
	}

	return nil
}

// appendToolResult 统一写入 tool 消息，确保所有路径输出格式一致。
func (a *Agent) appendToolResult(toolCallID, toolName, traceID, output string, toolErr error) {
	a.history.Append(llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: toolCallID,
		Text:       formatToolResult(toolName, traceID, output, toolErr),
	})
}

type toolResultEnvelope struct {
	Status  string `json:"status"`
	Tool    string `json:"tool"`
	TraceID string `json:"trace_id"`
	Output  string `json:"output"`
	Error   string `json:"error"`
}

// formatToolResult 把 tool 执行结果规范化为稳定 JSON envelope。
func formatToolResult(toolName, traceID, output string, toolErr error) string {
	result := toolResultEnvelope{
		Tool:    toolName,
		TraceID: traceID,
		Output:  output,
	}
	if toolErr != nil {
		result.Status = "error"
		result.Error = toolErr.Error()
		result.Output = ""
	} else {
		result.Status = "success"
		result.Error = ""
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		return `{"status":"error","tool":"internal","trace_id":"","output":"","error":"failed to encode tool result"}`
	}
	return string(encoded)
}

// validateToolCall 做最小输入护栏：id/name 必填，arguments 必须是 JSON object。
func validateToolCall(call llm.ToolCall) (toolCallID string, toolName string, args json.RawMessage, err error) {
	toolCallID = strings.TrimSpace(call.ID)
	if toolCallID == "" {
		return "", "", nil, errors.New("tool_call.id is empty")
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

// normalizedToolArguments 只接受 object；空值归一化为 {}。
func normalizedToolArguments(raw json.RawMessage) (json.RawMessage, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return json.RawMessage(`{}`), nil
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
