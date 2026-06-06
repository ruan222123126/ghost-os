package agent

import "ghost-os/bridge/llm"

// ToolResultEnvelope 是 Central 内部持久化 tool 消息时使用的稳定 envelope。
type ToolResultEnvelope = llm.ToolResultEnvelope

func appendToolResult(history *History, toolCallID string, toolName string, traceID string, output string, toolErr error, content []llm.ContentPart) {
	if history == nil {
		return
	}

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
	return llm.FormatToolResult(toolName, traceID, output, toolErr)
}

// FormatToolResult 对外暴露统一 tool result 编码，便于跨请求恢复工具结果。
func FormatToolResult(toolName string, traceID string, output string, toolErr error) string {
	return formatToolResult(toolName, traceID, output, toolErr)
}

// ParseToolResultEnvelope 解析 Agent 内部 tool result envelope，供上层投影为稳定外部 DTO。
func ParseToolResultEnvelope(raw string) (ToolResultEnvelope, bool) {
	return llm.ParseToolResultEnvelope(raw)
}
