package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// BashExecTool 提供 shell 执行入口，具体执行下沉到 execution layer。
type BashExecTool struct {
	execution ExecutionClient
}

type bashExecArgs struct {
	Command string `json:"command"`
}

func NewBashExecTool(client ExecutionClient) Tool {
	return BashExecTool{execution: client}
}

func (BashExecTool) Name() string {
	return "bash_exec"
}

func (BashExecTool) Description() string {
	return "Execute a shell command on the host machine."
}

func (BashExecTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {
				"type": "string",
				"description": "Shell command to run."
			}
		},
		"required": ["command"],
		"additionalProperties": false
	}`)
}

func (t BashExecTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	if t.execution == nil {
		return "", fmt.Errorf("execution client is not configured")
	}

	var args bashExecArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}

	command := strings.TrimSpace(args.Command)
	if command == "" {
		return "", fmt.Errorf("command is required")
	}

	payload, err := t.execution.Call(ctx, "BASH_EXEC", map[string]any{"command": command}, traceID)
	if err != nil {
		return "", fmt.Errorf("execution BASH_EXEC failed: %w", err)
	}

	output, ok := payload["output"].(string)
	if !ok {
		return "", fmt.Errorf("invalid BASH_EXEC payload: output must be a string")
	}
	return output, nil
}
