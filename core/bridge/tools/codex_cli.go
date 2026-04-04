package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/tools/internal/payloadutil"
)

const (
	codexCLIOpStart  = "start"
	codexCLIOpResume = "resume"
	codexCLIOpFork   = "fork"
	codexCLIOpStatus = "status"

	defaultCodexCLIModel               = "gpt-5.4"
	defaultCodexCLIWaitMSBeforeAsync   = 3000
	defaultCodexCLIWaitDurationSeconds = 300
	defaultCodexCLIOutputChars         = 200
)

type CodexCLITool struct {
	execution        ExecutionClient
	nativePersistent bool
}

type codexCLIArgs struct {
	Op                   string `json:"op"`
	Prompt               string `json:"prompt,omitempty"`
	SessionID            string `json:"session_id,omitempty"`
	Cwd                  string `json:"cwd,omitempty"`
	OutputPath           string `json:"output_path,omitempty"`
	Model                string `json:"model,omitempty"`
	FullAuto             *bool  `json:"full_auto,omitempty"`
	SkipGitRepoCheck     *bool  `json:"skip_git_repo_check,omitempty"`
	JSON                 *bool  `json:"json,omitempty"`
	WaitMSBeforeAsync    int    `json:"wait_ms_before_async,omitempty"`
	WaitDurationSeconds  int    `json:"wait_duration_seconds,omitempty"`
	OutputCharacterCount int    `json:"output_character_count,omitempty"`
}

type codexCLIResult struct {
	Status     string `json:"status"`
	CommandID  string `json:"command_id,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
	ExitCode   *int   `json:"exit_code,omitempty"`
	OutputTail string `json:"output_tail,omitempty"`
	OutputPath string `json:"output_path,omitempty"`
	Message    string `json:"message,omitempty"`
}

// NewCodexCLITool creates the codex_cli tool. It requires native_persistent=true.
func NewCodexCLITool(client ExecutionClient, nativePersistent bool) Tool {
	return &CodexCLITool{execution: client, nativePersistent: nativePersistent}
}

func (CodexCLITool) Name() string {
	return "codex_cli"
}

func (CodexCLITool) Description() string {
	return "Run codex start/resume/fork asynchronously and poll status. Requires native_persistent=true. For status, pass the command_id (or session_id) via session_id."
}

func (CodexCLITool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"op":{"type":"string","enum":["start","resume","fork","status"]},
			"prompt":{"type":"string","description":"Prompt for start/resume/fork."},
			"session_id":{"type":"string","description":"Session ID for resume/fork/status. For status, you may pass command_id here."},
			"cwd":{"type":"string","description":"Optional relative working directory for -C and process cwd."},
			"output_path":{"type":"string","description":"Optional output path for -o."},
			"model":{"type":"string","description":"Model name. Defaults to gpt-5.4."},
			"full_auto":{"type":"boolean","description":"Whether to add --full-auto. Defaults to true."},
			"skip_git_repo_check":{"type":"boolean","description":"Whether to add --skip-git-repo-check. Defaults to true."},
			"json":{"type":"boolean","description":"Whether to add --json. Defaults to true."},
			"wait_ms_before_async":{"type":"integer","minimum":0,"description":"Start wait in ms before returning. Defaults to 3000."},
			"wait_duration_seconds":{"type":"integer","minimum":0,"description":"Status polling window in seconds. Defaults to 300."},
			"output_character_count":{"type":"integer","minimum":1,"description":"Max output tail size for status. Defaults to 200."}
		},
		"required":["op"],
		"additionalProperties":false
	}`)
}

func (t *CodexCLITool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	if err := t.validateExecutionConfig(); err != nil {
		return "", err
	}
	request, err := parseCodexCLIRequest(argsJSON)
	if err != nil {
		return "", err
	}

	payload, err := t.execution.Call(ctx, request.Action, request.Params, traceID)
	if err != nil {
		return marshalCodexCLIResult(codexCLIResult{
			Status:     "error",
			Message:    err.Error(),
			OutputPath: request.OutputPath,
		})
	}

	result, err := codexCLIResultFromPayload(payload, request.OutputPath)
	if err != nil {
		return "", err
	}
	return marshalCodexCLIResult(result)
}

func (t *CodexCLITool) validateExecutionConfig() error {
	if t == nil || t.execution == nil {
		return fmt.Errorf("execution client is not configured")
	}
	if !t.nativePersistent {
		return fmt.Errorf("codex_cli requires native_persistent=true")
	}
	return nil
}

func isCodexCLIOperation(op string) bool {
	switch op {
	case codexCLIOpStart, codexCLIOpResume, codexCLIOpFork, codexCLIOpStatus:
		return true
	default:
		return false
	}
}

func boolOrDefault(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func normalizedNonNegativeInt(value int, fallback int, field string) (int, error) {
	if value < 0 {
		return 0, fmt.Errorf("%s must be >= 0", field)
	}
	if value == 0 {
		return fallback, nil
	}
	return value, nil
}

func codexCLIResultFromPayload(payload map[string]any, fallbackOutputPath string) (codexCLIResult, error) {
	status, err := payloadutil.String(payload, "status")
	if err != nil {
		return codexCLIResult{}, err
	}
	commandID, err := optionalStringPayload(payload, "command_id")
	if err != nil {
		return codexCLIResult{}, err
	}
	sessionID, err := optionalStringPayload(payload, "session_id")
	if err != nil {
		return codexCLIResult{}, err
	}
	outputTail, err := optionalStringPayload(payload, "output_tail")
	if err != nil {
		return codexCLIResult{}, err
	}
	outputPath, err := optionalStringPayload(payload, "output_path")
	if err != nil {
		return codexCLIResult{}, err
	}
	message, err := optionalStringPayload(payload, "message")
	if err != nil {
		return codexCLIResult{}, err
	}
	var exitCode *int
	if raw, ok := payload["exit_code"]; ok {
		if raw == nil {
			exitCode = nil
		} else if value, ok := payloadutil.NumericToInt(raw); ok {
			exitCode = &value
		} else {
			return codexCLIResult{}, fmt.Errorf("invalid payload: exit_code must be an integer")
		}
	}

	if outputPath == "" {
		outputPath = strings.TrimSpace(fallbackOutputPath)
	}

	return codexCLIResult{
		Status:     strings.TrimSpace(status),
		CommandID:  strings.TrimSpace(commandID),
		SessionID:  strings.TrimSpace(sessionID),
		ExitCode:   exitCode,
		OutputTail: outputTail,
		OutputPath: outputPath,
		Message:    strings.TrimSpace(message),
	}, nil
}

func optionalStringPayload(payload map[string]any, field string) (string, error) {
	raw, ok := payload[field]
	if !ok {
		return "", nil
	}
	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("invalid payload: %s must be a string", field)
	}
	return value, nil
}

func marshalCodexCLIResult(result codexCLIResult) (string, error) {
	encoded, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("encode result: %w", err)
	}
	return string(encoded), nil
}
