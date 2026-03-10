package agent

import (
	"encoding/json"
	"strings"

	"ghost-os/bridge/llm"
)

// ToolResultEnvelope 是 Central 内部持久化 tool 消息时使用的稳定 envelope。
type ToolResultEnvelope struct {
	Status  string `json:"status"`
	Tool    string `json:"tool"`
	TraceID string `json:"trace_id"`
	Output  string `json:"output"`
	Error   string `json:"error"`
}

func appendToolResult(history *History, toolCallID string, toolName string, traceID string, output string, toolErr error, content []llm.ContentPart) {
	message := llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: toolCallID,
		Text:       formatToolResult(toolName, traceID, output, toolErr),
	}
	if len(content) > 0 && toolErr == nil {
		message.Content = content
	}
	history.Append(message)
}

// formatToolResult 把 tool 执行结果规范化为稳定 JSON envelope。
func formatToolResult(toolName string, traceID string, output string, toolErr error) string {
	result := ToolResultEnvelope{
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
func FormatToolResult(toolName string, traceID string, output string, toolErr error) string {
	return formatToolResult(toolName, traceID, output, toolErr)
}

// ParseToolResultEnvelope 解析 Agent 内部 tool result envelope，供上层投影为稳定外部 DTO。
func ParseToolResultEnvelope(raw string) (ToolResultEnvelope, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ToolResultEnvelope{}, false
	}

	var result ToolResultEnvelope
	if err := json.Unmarshal([]byte(trimmed), &result); err != nil {
		return ToolResultEnvelope{}, false
	}
	if strings.TrimSpace(result.Status) == "" || strings.TrimSpace(result.Tool) == "" {
		return ToolResultEnvelope{}, false
	}
	return result, true
}
