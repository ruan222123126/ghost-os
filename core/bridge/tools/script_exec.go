package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ScriptExecTool 在受限 Python 沙盒中执行脚本，并透传 execution layer 能力。
type ScriptExecTool struct {
	execution ExecutionClient
}

type scriptExecArgs struct {
	Script      string `json:"script"`
	TimeoutMs   int    `json:"timeout_ms,omitempty"`
	MaxMemoryMB int    `json:"max_memory_mb,omitempty"`
}

// NewScriptExecTool 创建 script_exec 工具并绑定 execution 客户端。
func NewScriptExecTool(client ExecutionClient) Tool {
	return ScriptExecTool{execution: client}
}

func (ScriptExecTool) Name() string {
	return "script_exec"
}

func (ScriptExecTool) Description() string {
	return `Execute a Python script in a sandboxed environment with access to tools.

Available tools in the script:
- tools.bash_exec(command: str) -> str
- tools.list_files(path: str = ".") -> list[str]
- tools.read_file(path: str, start_line: int | None = None, end_line: int | None = None) -> str
- tools.write_file(path: str, content: str, mode: str = "write") -> str
- tools.apply_diff(path: str, diff_text: str) -> str
- tools.search_files(keyword: str, dir_path: str = ".", case_sensitive: bool = True) -> list[str]
- tools.fetch_webpage(url: str) -> str`
}

func (ScriptExecTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"script": {
				"type": "string",
				"description": "Python script to execute. Use tools.* helpers for shell/file/search/web operations."
			},
			"timeout_ms": {
				"type": "integer",
				"description": "Execution timeout in milliseconds. Limits are enforced by the execution layer."
			},
			"max_memory_mb": {
				"type": "integer",
				"description": "Maximum memory usage in MB. Limits are enforced by the execution layer."
			}
		},
		"required": ["script"],
		"additionalProperties": false
	}`)
}

// Execute 校验脚本参数后调用 execution 层，资源限制由 native 统一裁剪。
func (t ScriptExecTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	if t.execution == nil {
		return "", fmt.Errorf("execution client is not configured")
	}

	var args scriptExecArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}

	script := strings.TrimSpace(args.Script)
	if script == "" {
		return "", fmt.Errorf("script is required")
	}
	if len(script) > 10*1024 {
		return "", fmt.Errorf("script too long (max 10KB)")
	}

	// 资源约束在 execution layer 统一裁剪，避免与 native 规则重复漂移。
	params := map[string]any{
		"script": script,
	}
	if args.TimeoutMs > 0 {
		params["timeout_ms"] = args.TimeoutMs
	}
	if args.MaxMemoryMB > 0 {
		params["max_memory_mb"] = args.MaxMemoryMB
	}

	payload, err := t.execution.Call(ctx, "SCRIPT_EXEC", params, traceID)
	if err != nil {
		return "", fmt.Errorf("execution SCRIPT_EXEC failed: %w", err)
	}

	outputValue, ok := payload["output"]
	if !ok {
		return "", fmt.Errorf("invalid SCRIPT_EXEC payload: missing output")
	}
	output, ok := outputValue.(string)
	if !ok {
		return "", fmt.Errorf("invalid SCRIPT_EXEC payload: output must be a string")
	}

	toolCallsValue, ok := payload["tool_calls_log"]
	if !ok {
		return "", fmt.Errorf("invalid SCRIPT_EXEC payload: missing tool_calls_log")
	}
	toolCallsLog, ok := toolCallsValue.([]any)
	if !ok {
		return "", fmt.Errorf("invalid SCRIPT_EXEC payload: tool_calls_log must be an array")
	}

	var result strings.Builder
	if strings.TrimSpace(output) != "" {
		result.WriteString("Script Output:\n")
		result.WriteString(output)
		if !strings.HasSuffix(output, "\n") {
			result.WriteByte('\n')
		}
	}

	if len(toolCallsLog) > 0 {
		if result.Len() > 0 {
			result.WriteByte('\n')
		}
		result.WriteString("Tool Calls:\n")
		for i, rawCall := range toolCallsLog {
			callMap, ok := rawCall.(map[string]any)
			if !ok {
				continue
			}

			toolName, _ := callMap["tool"].(string)
			result.WriteString(fmt.Sprintf("%d. %s\n", i+1, toolName))

			if callErr, ok := callMap["error"].(string); ok && strings.TrimSpace(callErr) != "" {
				result.WriteString(fmt.Sprintf("   Error: %s\n", callErr))
			}
		}
	}

	if result.Len() == 0 {
		return "(no output)", nil
	}
	return strings.TrimRight(result.String(), "\n"), nil
}
