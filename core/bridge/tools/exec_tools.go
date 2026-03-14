package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"ghost-os/bridge/tools/internal/payloadutil"
)

type ScriptExecTool struct {
	execution ExecutionClient
}

type scriptExecArgs struct {
	Script      string `json:"script"`
	TimeoutMs   int    `json:"timeout_ms,omitempty"`
	MaxMemoryMB int    `json:"max_memory_mb,omitempty"`
}

var scriptExecAllowedHelpers = []string{
	"tools.apply_diff(path, diff_text)",
	"tools.bash_exec(command)",
	"tools.fetch_webpage(url)",
	"tools.list_files(path='.')",
	"tools.read_file(path, start_line=None, end_line=None)",
	"tools.search_files(keyword, dir_path='.', case_sensitive=True)",
	"tools.write_file(path, content, mode='write')",
}

// NewScriptExecTool 创建 script_exec 工具并绑定 execution 客户端。
func NewScriptExecTool(client ExecutionClient) Tool {
	return ScriptExecTool{execution: client}
}

func (ScriptExecTool) Name() string {
	return "script_exec"
}

func (ScriptExecTool) Description() string {
	allowedHelpers := append([]string(nil), scriptExecAllowedHelpers...)
	sort.Strings(allowedHelpers)

	return "Execute a Python script in the fallback sandbox as the primary local workspace tool. For deterministic edits, prefer tools.list_files, tools.read_file, tools.search_files, tools.apply_diff, and tools.write_file inside the script. Use tools.bash_exec only when a shell command is required.\n\nAllowed helpers: " + strings.Join(allowedHelpers, ", ") + ".\nSandbox limits are enforced by the execution layer."
}

func (ScriptExecTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
				"script": {
					"type": "string",
					"description": "Python script for fallback sandbox execution."
				},
				"timeout_ms": {
					"type": "integer",
					"description": "Optional timeout in milliseconds."
				},
				"max_memory_mb": {
					"type": "integer",
					"description": "Optional memory limit in MB."
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

	params := map[string]any{"script": script}
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

	output, err := payloadutil.String(payload, "output")
	if err != nil {
		return "", fmt.Errorf("invalid SCRIPT_EXEC payload: %w", err)
	}
	toolCallsLog, err := payloadutil.AnySlice(payload, "tool_calls_log")
	if err != nil {
		return "", fmt.Errorf("invalid SCRIPT_EXEC payload: %w", err)
	}
	return formatScriptExecReport(output, toolCallsLog)
}
