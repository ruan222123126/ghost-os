package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
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
	codexCLIExecutable                 = "codex"
)

type CodexProjectRootResolver func() (string, error)

type CodexCLITool struct {
	manager            *codexCLICommandManager
	allowedReadPaths   []string
	allowedWritePaths  []string
	resolveProjectRoot CodexProjectRootResolver
	commandFactory     func(name string, args ...string) *exec.Cmd
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

func NewCodexCLITool(_ ExecutionClient, _ bool) Tool {
	return &CodexCLITool{
		manager:        newCodexCLICommandManager(),
		commandFactory: exec.Command,
	}
}

func (CodexCLITool) Name() string {
	return "codex_cli"
}

func (CodexCLITool) Description() string {
	return "Run codex start/resume/fork asynchronously and poll status. For status, pass the command_id (or session_id) via session_id."
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

func (t *CodexCLITool) Execute(ctx context.Context, argsJSON json.RawMessage, _ string) (string, error) {
	request, err := parseCodexCLIRequest(argsJSON)
	if err != nil {
		return "", err
	}
	result := t.executeRequest(ctx, request)
	return marshalCodexCLIResult(result)
}

func (t *CodexCLITool) executeRequest(ctx context.Context, request codexCLIRequest) codexCLIResult {
	switch request.Action {
	case "CODEX_CLI_START":
		return t.executeStart(ctx, request)
	case "CODEX_CLI_STATUS":
		return t.executeStatus(ctx, request)
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
