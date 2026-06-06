package llm

import (
	"encoding/json"
	"strings"
)

// ToolResultEnvelope is the stable JSON envelope stored in tool messages.
type ToolResultEnvelope struct {
	Status  string `json:"status"`
	Tool    string `json:"tool"`
	TraceID string `json:"trace_id"`
	Output  string `json:"output"`
	Error   string `json:"error"`
}

func FormatToolResult(toolName string, traceID string, output string, toolErr error) string {
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
