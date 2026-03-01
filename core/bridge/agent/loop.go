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

	initialHistoryLen int
}

var errToolCallIDRequired = errors.New("tool_call.id is empty")

const maxConsecutiveNonExecutableToolCallTurns = 3

// ErrAwaitingHuman 表示 ask_human 已发起问题，当前回合需要等待用户输入。
type ErrAwaitingHuman struct {
	QuestionID string
	Prompt     string
}

func (e *ErrAwaitingHuman) Error() string {
	return fmt.Sprintf("awaiting human input: question_id=%s", strings.TrimSpace(e.QuestionID))
}

func NewAgent(completer Completer, toolCatalog ToolCatalog, systemPrompt string, maxTurns int) *Agent {
	return NewAgentWithHistory(completer, toolCatalog, NewHistory(systemPrompt), maxTurns)
}

// NewAgentWithHistory 使用预加载历史初始化 Agent，适用于跨请求会话。
func NewAgentWithHistory(completer Completer, toolCatalog ToolCatalog, history *History, maxTurns int) *Agent {
	if maxTurns <= 0 {
		maxTurns = 20
	}
	if history == nil {
		history = NewHistory("")
	}

	return &Agent{
		completer:         completer,
		tools:             toolCatalog,
		history:           history,
		maxTurns:          maxTurns,
		initialHistoryLen: history.Len(),
	}
}

// Run 负责循环与退出条件；单步逻辑拆到私有方法里，便于测试与扩展。
func (a *Agent) Run(ctx context.Context, userMessage string) (string, error) {
	return a.RunWithTraceID(ctx, userMessage, "")
}

// RunWithTraceID 允许调用方注入请求级 trace_id，保障跨层链路追踪一致。
func (a *Agent) RunWithTraceID(ctx context.Context, userMessage string, traceID string) (string, error) {
	traceID = strings.TrimSpace(traceID)
	if traceID == "" {
		traceID = fmt.Sprintf("agent-%d", time.Now().UnixNano())
	}
	a.appendUserMessage(userMessage)
	consecutiveNonExecutableToolCallTurns := 0

	for turn := 0; turn < a.maxTurns; turn++ {
		msg, finishReason, err := a.completeOnce(ctx)
		if err != nil {
			return "", fmt.Errorf("trace_id=%s turn=%d complete_once: %w", traceID, turn, err)
		}

		switch finishReason {
		case llm.FinishStop:
			return a.handleAssistantStop(msg), nil
		case llm.FinishToolCalls:
			stats, err := a.handleToolCalls(ctx, traceID, msg.ToolCalls)
			if err != nil {
				return "", fmt.Errorf("trace_id=%s turn=%d handle_tool_calls: %w", traceID, turn, err)
			}
			if stats.nonExecutable() {
				consecutiveNonExecutableToolCallTurns++
			} else {
				consecutiveNonExecutableToolCallTurns = 0
			}
			// 防止模型反复生成不可执行的 tool_call（空参数/缺失工具）导致无效循环。
			if consecutiveNonExecutableToolCallTurns >= maxConsecutiveNonExecutableToolCallTurns {
				return "", fmt.Errorf("trace_id=%s turn=%d repeated non-executable tool_calls; aborting tool-call loop", traceID, turn)
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

type toolCallTurnStats struct {
	totalCalls int
	executed   int
}

func (s toolCallTurnStats) nonExecutable() bool {
	return s.totalCalls > 0 && s.executed == 0
}

// handleToolCalls 负责执行工具并把结果统一写回历史，供下一轮模型继续推理。
func (a *Agent) handleToolCalls(ctx context.Context, traceID string, calls []llm.ToolCall) (toolCallTurnStats, error) {
	stats := toolCallTurnStats{totalCalls: len(calls)}
	if len(calls) == 0 {
		return stats, errors.New("finish_reason=tool_calls but tool_calls is empty")
	}

	for _, call := range calls {
		toolCallID, toolName, args, err := validateToolCall(call)
		if err != nil {
			// 统一降级处理：记录错误并继续，让模型有机会自我纠正。
			fmt.Fprintf(os.Stderr, "[%s] invalid_tool_call: id=%q name=%q error=%v\n", traceID, strings.TrimSpace(call.ID), strings.TrimSpace(call.Name), err)
			a.appendToolResult(toolCallID, toolName, traceID, "", err, nil)
			continue
		}

		fmt.Fprintf(os.Stderr, "[%s] tool_call: %s args=%s\n", traceID, toolName, string(args))

		tool := a.tools.Get(toolName)
		if tool == nil {
			a.appendToolResult(toolCallID, toolName, traceID, "", fmt.Errorf("tool %q not found", toolName), nil)
			continue
		}

		toolCtx := tools.WithToolCallID(ctx, toolCallID)
		stats.executed++
		output, execErr := tool.Execute(toolCtx, args, traceID)
		if execErr != nil {
			a.appendToolResult(toolCallID, toolName, traceID, "", fmt.Errorf("tool %q error: %w", toolName, execErr), nil)
			continue
		}

		meta := tools.InterpretExecuteResult(tool, output)
		if meta.AwaitingHuman != nil {
			// 暂停态通过专用错误向上抛，让上层保存现场并等待用户输入。
			return stats, &ErrAwaitingHuman{
				QuestionID: strings.TrimSpace(meta.AwaitingHuman.QuestionID),
				Prompt:     strings.TrimSpace(meta.AwaitingHuman.Prompt),
			}
		}

		a.appendToolResult(toolCallID, toolName, traceID, output, nil, meta.Content)
	}

	return stats, nil
}

// appendToolResult 统一写入 tool 消息，确保所有路径输出格式一致。
func (a *Agent) appendToolResult(toolCallID, toolName, traceID, output string, toolErr error, content []llm.ContentPart) {
	message := llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: toolCallID,
		Text:       formatToolResult(toolName, traceID, output, toolErr),
	}
	if len(content) > 0 && toolErr == nil {
		message.Content = content
	}
	a.history.Append(message)
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

// FormatToolResult 对外暴露统一 tool result 编码，便于跨请求恢复工具结果。
func FormatToolResult(toolName, traceID, output string, toolErr error) string {
	return formatToolResult(toolName, traceID, output, toolErr)
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

// GetNewMessages 返回 Agent 初始化后新增的会话消息。
func (a *Agent) GetNewMessages() []llm.Message {
	if a == nil || a.history == nil {
		return nil
	}

	messages := a.history.Messages()
	if a.initialHistoryLen <= 0 {
		return messages
	}
	if a.initialHistoryLen >= len(messages) {
		return nil
	}

	return llm.CloneMessages(messages[a.initialHistoryLen:])
}
