package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

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

func (BashExecTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "Shell command to run.",
			},
		},
		"required":             []string{"command"},
		"additionalProperties": false,
	}
}

func (BashExecTool) Execute(_ context.Context, argsJSON string) (string, error) {
	var args bashExecArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}

	command := strings.TrimSpace(args.Command)
	if command == "" {
		return "", fmt.Errorf("command is required")
	}

	return fmt.Sprintf("[stub] would execute: %s", command), nil
}
