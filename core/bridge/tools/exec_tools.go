package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/tools/internal/payloadutil"
)

type ScriptExecTool struct {
	execution          ExecutionClient
	defaultMaxMemoryMB int
}

type scriptExecArgs struct {
	Script      string `json:"script"`
	TimeoutMs   int    `json:"timeout_ms,omitempty"`
	MaxMemoryMB int    `json:"max_memory_mb,omitempty"`
}

const scriptExecDescription = "Python sandbox. Each call runs a fresh script. " +
	"No state carries across calls; re-import modules and recreate variables every time. " +
	"Helpers are top-level functions and accept positional or named parameters (no import, do not use tools.*): " +
	"list_files(path='.'), read_file(path, start_line=None, end_line=None), " +
	"search_files(query, path='.', max_results=50), write_file(path, content, mode='write'), " +
	"apply_diff(path, diff_text), bash_exec(command, max_output_chars=None), fetch_webpage(url). " +
	"write_file creates missing parent directories automatically. " +
	"list_files returns a string array and directory entries end with '/'. " +
	"For longer shell stdout, request bash_exec(..., max_output_chars=N) explicitly. " +
	"open(path, mode) supports UTF-8 text r/w/a only; prefer read_file/search_files for text access. " +
	"Common modules json/os/sys are preloaded for each call. " +
	"subprocess.run/check_output and os.popen/os.system are compatibility shims backed by the sandbox shell path; non-whitelisted modules like pathlib remain unavailable. " +
	"Output concise JSON/text."

// NewScriptExecTool 创建 script_exec 工具并绑定 execution 客户端。
func NewScriptExecTool(client ExecutionClient, defaultMaxMemoryMB int) Tool {
	return ScriptExecTool{
		execution:          client,
		defaultMaxMemoryMB: defaultMaxMemoryMB,
	}
}

func (ScriptExecTool) Name() string {
	return "script_exec"
}

func (ScriptExecTool) Description() string {
	return scriptExecDescription
}

func (ScriptExecTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"script": {
				"type": "string"
			},
			"timeout_ms": {
				"type": "integer"
			},
			"max_memory_mb": {
				"type": "integer"
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
	} else if t.defaultMaxMemoryMB > 0 {
		params["max_memory_mb"] = t.defaultMaxMemoryMB
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
