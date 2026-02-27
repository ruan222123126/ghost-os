package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// BashExecTool 提供最小 shell 执行能力（当前为 stub）。
type BashExecTool struct{}

type bashExecArgs struct {
	Command string `json:"command"`
}

func NewBashExecTool() Tool {
	return BashExecTool{}
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

// Execute 解析参数并返回 stub 结果，保留未来真实执行扩展点。
func (BashExecTool) Execute(_ context.Context, argsJSON json.RawMessage) (string, error) {
	var args bashExecArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}

	command := strings.TrimSpace(args.Command)
	if command == "" {
		return "", fmt.Errorf("command is required")
	}

	return fmt.Sprintf("[stub] would execute: %s", command), nil
}
