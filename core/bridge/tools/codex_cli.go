package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

type CodexProjectRootResolver func() (string, error)

type CodexCLITool struct {
	execution          ExecutionClient
	manager            *codexCLICommandManager
	allowedReadPaths   []string
	allowedWritePaths  []string
	resolveProjectRoot CodexProjectRootResolver
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

func NewCodexCLITool(client ExecutionClient, _ bool) Tool {
	return &CodexCLITool{
		execution: client,
		manager:   newCodexCLICommandManager(),
	}
}

func (CodexCLITool) Name() string {
	return "codex_cli"
}

func (CodexCLITool) Description() string {
	return "Async codex runner. Rules: 'prompt' required for start/resume/fork. 'session_id' required for resume/fork/status (pass command_id here for status). DO NOT use 'exec'."
}

func (CodexCLITool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"op":{"type":"string","enum":["start","resume","fork","status"]},
			"prompt":{"type":"string"},
			"session_id":{"type":"string"},
			"cwd":{"type":"string"},
			"output_path":{"type":"string"},
			"model":{"type":"string","description":"Default: gpt-5.4"},
			"full_auto":{"type":"boolean","description":"Default: true"},
			"skip_git_repo_check":{"type":"boolean","description":"Default: true"},
			"json":{"type":"boolean","description":"Default: true"},
			"wait_ms_before_async":{"type":"integer"},
			"wait_duration_seconds":{"type":"integer"},
			"output_character_count":{"type":"integer"}
		},
		"required":["op"],
		"additionalProperties":false
	}`)
}

func (t *CodexCLITool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	request, err := parseCodexCLIRequest(argsJSON)
	if err != nil {
		return "", err
	}
	result := t.executeRequest(ctx, request, traceID)
	return marshalCodexCLIResult(result)
}

func (t *CodexCLITool) executeRequest(
	ctx context.Context,
	request codexCLIRequest,
	traceID string,
) codexCLIResult {
	switch request.Action {
	case "CODEX_CLI_START":
		return t.executeStart(ctx, request, traceID)
	case "CODEX_CLI_STATUS":
		return t.executeStatus(ctx, request, traceID)
	default:
		return codexCLIResult{
			Status:     "error",
			Message:    fmt.Sprintf("unsupported action: %s", request.Action),
			OutputPath: strings.TrimSpace(request.OutputPath),
		}
	}
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

func marshalCodexCLIResult(result codexCLIResult) (string, error) {
	encoded, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("encode result: %w", err)
	}
	return string(encoded), nil
}
