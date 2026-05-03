package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/tools/internal/payloadutil"
)

type BashExecTool struct {
	execution ExecutionClient
}

type bashExecArgs struct {
	Command        string `json:"command"`
	MaxOutputChars *int   `json:"max_output_chars,omitempty"`
}

func NewBashExecTool(client ExecutionClient) Tool {
	return BashExecTool{execution: client}
}

func (BashExecTool) Name() string {
	return "bash_exec"
}

func (BashExecTool) Description() string {
	return "Run a shell command in the sandbox bash shell. Returns stdout only; non-zero exit fails."
}

func (BashExecTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"command":{"type":"string","description":"Shell command to run."},
			"max_output_chars":{"type":"integer","minimum":1,"description":"Optional stdout preview limit override."}
		},
		"required":["command"],
		"additionalProperties":false
	}`)
}

func (t BashExecTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	if t.execution == nil {
		return "", fmt.Errorf("execution client is not configured")
	}

	params, err := decodeBashExecParams(argsJSON)
	if err != nil {
		return "", err
	}
	payload, err := t.execution.Call(ctx, "BASH_EXEC", params, traceID)
	if err != nil {
		return "", fmt.Errorf("execution BASH_EXEC failed: %w", err)
	}
	stdout, err := payloadutil.String(payload, "stdout")
	if err != nil {
		return "", fmt.Errorf("invalid BASH_EXEC payload: %w", err)
	}
	return stdout, nil
}

func decodeBashExecParams(argsJSON json.RawMessage) (map[string]any, error) {
	var args bashExecArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return nil, fmt.Errorf("decode args: %w", err)
	}
	command := strings.TrimSpace(args.Command)
	if command == "" {
		return nil, fmt.Errorf("command is required")
	}
	if args.MaxOutputChars != nil && *args.MaxOutputChars < 1 {
		return nil, fmt.Errorf("max_output_chars must be >= 1")
	}

	params := map[string]any{"command": command}
	if args.MaxOutputChars != nil {
		params["max_output_chars"] = *args.MaxOutputChars
	}
	return params, nil
}
