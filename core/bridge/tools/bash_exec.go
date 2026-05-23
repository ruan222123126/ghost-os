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
	Login          *bool  `json:"login,omitempty"`
	Interactive    *bool  `json:"interactive,omitempty"`
	SessionID      string `json:"session_id,omitempty"`
	TTY            *bool  `json:"tty,omitempty"`
	YieldTimeMs    *int   `json:"yield_time_ms,omitempty"`
	TimeoutMs      *int   `json:"timeout_ms,omitempty"`
	MaxOutputChars *int   `json:"max_output_chars,omitempty"`
}

type bashExecRequest struct {
	Params      map[string]any
	Interactive bool
}

func NewBashExecTool(client ExecutionClient) Tool {
	return BashExecTool{execution: client}
}

func (BashExecTool) Name() string {
	return "bash_exec"
}

func (BashExecTool) Description() string {
	return "Run a bash shell command in the sandbox. Use one-shot mode by default; use interactive=true only when you need a persistent shell session (reuse session_id across calls) such as setting env vars, changing directories, or running multi-step scripts. interactive=true does not support tty=true. Non-zero exit fails."
}

func (BashExecTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"command":{"type":"string","description":"Shell command to run."},
			"login":{"type":"boolean","description":"One-shot only. true uses bash -lc (default), false uses bash -c."},
			"interactive":{"type":"boolean","description":"Enable persistent shell session mode. Use this only when state must persist across calls (cd/export/temporary files). Do not pass timeout_ms or login in interactive mode."},
			"session_id":{"type":"string","description":"Interactive only. Reuse an existing shell session_id from a previous interactive call. Omit to create a new session_id."},
			"tty":{"type":"boolean","description":"Interactive only. Not supported yet: do not set tty=true."},
			"yield_time_ms":{"type":"integer","minimum":1,"description":"Interactive only. Wait window before collecting incremental output (default ~100ms, max 60000ms)."},
			"timeout_ms":{"type":"integer","minimum":1,"description":"One-shot only. Per-call timeout in milliseconds. Do not pass this when interactive=true."},
			"max_output_chars":{"type":"integer","minimum":1,"description":"Optional stdout/stderr preview limit override."}
		},
		"required":["command"],
		"additionalProperties":false,
		"allOf":[
			{
				"if":{"required":["interactive"],"properties":{"interactive":{"const":true}}},
				"then":{"not":{"anyOf":[{"required":["login"]},{"required":["timeout_ms"]}]}}
			},
			{
				"if":{"properties":{"interactive":{"const":false}}},
				"then":{"not":{"anyOf":[{"required":["session_id"]},{"required":["tty"]},{"required":["yield_time_ms"]}]}}
			}
		]
	}`)
}

func (t BashExecTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	if t.execution == nil {
		return "", fmt.Errorf("execution client is not configured")
	}

	request, err := decodeBashExecParams(argsJSON)
	if err != nil {
		return "", err
	}
	if request.Interactive && !supportsPersistentBashSessions(t.execution) {
		return "", fmt.Errorf("interactive bash_exec requires native_persistent=true")
	}

	payload, err := t.execution.Call(ctx, "BASH_EXEC", request.Params, traceID)
	if err != nil {
		return "", fmt.Errorf("execution BASH_EXEC failed: %w", err)
	}
	if request.Interactive {
		return encodeInteractiveBashExecPayload(payload)
	}

	stdout, err := payloadutil.String(payload, "stdout")
	if err != nil {
		return "", fmt.Errorf("invalid BASH_EXEC payload: %w", err)
	}
	return stdout, nil
}

func decodeBashExecParams(argsJSON json.RawMessage) (bashExecRequest, error) {
	var args bashExecArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return bashExecRequest{}, fmt.Errorf("decode args: %w", err)
	}

	command := strings.TrimSpace(args.Command)
	if command == "" {
		return bashExecRequest{}, fmt.Errorf("command is required")
	}

	interactive := args.Interactive != nil && *args.Interactive
	if err := validateBashExecArgs(args, interactive); err != nil {
		return bashExecRequest{}, err
	}

	return bashExecRequest{
		Params:      buildBashExecParams(command, args, interactive),
		Interactive: interactive,
	}, nil
}

func validateBashExecArgs(args bashExecArgs, interactive bool) error {
	if args.MaxOutputChars != nil && *args.MaxOutputChars < 1 {
		return fmt.Errorf("max_output_chars must be >= 1")
	}
	if args.YieldTimeMs != nil && *args.YieldTimeMs < 1 {
		return fmt.Errorf("yield_time_ms must be >= 1")
	}
	if args.TimeoutMs != nil && *args.TimeoutMs < 1 {
		return fmt.Errorf("timeout_ms must be >= 1")
	}

	sessionID := strings.TrimSpace(args.SessionID)
	if !interactive && sessionID != "" {
		return fmt.Errorf("session_id is only allowed when interactive=true")
	}
	if interactive {
		if args.Login != nil {
			return fmt.Errorf("login is only allowed when interactive=false")
		}
		if args.TimeoutMs != nil {
			return fmt.Errorf("timeout_ms is only allowed when interactive=false")
		}
		return nil
	}
	if args.YieldTimeMs != nil || args.TTY != nil {
		return fmt.Errorf("yield_time_ms and tty are only allowed when interactive=true")
	}
	return nil
}

func buildBashExecParams(command string, args bashExecArgs, interactive bool) map[string]any {
	params := map[string]any{"command": command}
	sessionID := strings.TrimSpace(args.SessionID)

	if args.Login != nil {
		params["login"] = *args.Login
	}
	if interactive {
		params["interactive"] = true
	}
	if sessionID != "" {
		params["session_id"] = sessionID
	}
	if args.TTY != nil {
		params["tty"] = *args.TTY
	}
	if args.YieldTimeMs != nil {
		params["yield_time_ms"] = *args.YieldTimeMs
	}
	if args.TimeoutMs != nil {
		params["timeout_ms"] = *args.TimeoutMs
	}
	if args.MaxOutputChars != nil {
		params["max_output_chars"] = *args.MaxOutputChars
	}
	return params
}

func supportsPersistentBashSessions(client ExecutionClient) bool {
	type persistentSessionSupport interface {
		SupportsPersistentSessions() bool
	}
	support, ok := client.(persistentSessionSupport)
	return ok && support.SupportsPersistentSessions()
}

func encodeInteractiveBashExecPayload(payload map[string]any) (string, error) {
	sessionID, err := payloadutil.String(payload, "session_id")
	if err != nil {
		return "", fmt.Errorf("invalid BASH_EXEC payload: %w", err)
	}
	stdout, err := payloadutil.String(payload, "stdout")
	if err != nil {
		return "", fmt.Errorf("invalid BASH_EXEC payload: %w", err)
	}
	stderr, err := payloadutil.String(payload, "stderr")
	if err != nil {
		return "", fmt.Errorf("invalid BASH_EXEC payload: %w", err)
	}
	running, err := readInteractiveRunning(payload)
	if err != nil {
		return "", err
	}

	result := map[string]any{
		"session_id": sessionID,
		"stdout":     stdout,
		"stderr":     stderr,
		"running":    running,
	}
	if err := attachExitCodeIfExited(payload, running, result); err != nil {
		return "", err
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("encode interactive BASH_EXEC payload: %w", err)
	}
	return string(encoded), nil
}

func readInteractiveRunning(payload map[string]any) (bool, error) {
	value, ok := payload["running"]
	if !ok {
		return false, fmt.Errorf("invalid BASH_EXEC payload: missing running")
	}
	running, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("invalid BASH_EXEC payload: running must be a boolean")
	}
	return running, nil
}

func attachExitCodeIfExited(payload map[string]any, running bool, result map[string]any) error {
	if running {
		return nil
	}
	value, exists := payload["exit_code"]
	if !exists {
		return fmt.Errorf("invalid BASH_EXEC payload: missing exit_code for exited session")
	}
	exitCode, ok := payloadutil.NumericToInt(value)
	if !ok {
		return fmt.Errorf("invalid BASH_EXEC payload: exit_code must be an integer")
	}
	result["exit_code"] = exitCode
	return nil
}
