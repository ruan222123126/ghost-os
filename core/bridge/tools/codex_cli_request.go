package tools

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

type codexCLIRequest struct {
	Action     string
	Params     map[string]any
	OutputPath string
}

type normalizedCodexCLIArgs struct {
	Op                  string
	Prompt              string
	SessionID           string
	Cwd                 string
	OutputPath          string
	Model               string
	Sandbox             string
	FullAuto            *bool
	SkipGitRepoCheck    bool
	JSON                bool
	WaitMSBeforeAsync   int
	WaitDurationSeconds int
	OutputCharCount     int
}

func parseCodexCLIRequest(argsJSON json.RawMessage) (codexCLIRequest, error) {
	var args codexCLIArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return codexCLIRequest{}, fmt.Errorf("decode args: %w", err)
	}
	normalized, err := normalizeCodexCLIArgs(args)
	if err != nil {
		return codexCLIRequest{}, err
	}
	return buildCodexCLIRequest(normalized), nil
}

func normalizeCodexCLIArgs(args codexCLIArgs) (normalizedCodexCLIArgs, error) {
	op, err := normalizeCodexCLIOp(args.Op)
	if err != nil {
		return normalizedCodexCLIArgs{}, err
	}

	normalized := normalizedCodexCLIArgsFromDecoded(op, args)

	if err := validateCodexCLIOpRequirements(normalized); err != nil {
		return normalizedCodexCLIArgs{}, err
	}
	if err := validateCodexCLICommonArgs(normalized); err != nil {
		return normalizedCodexCLIArgs{}, err
	}
	if err := normalizeCodexCLITimingArgs(&normalized); err != nil {
		return normalizedCodexCLIArgs{}, err
	}
	return normalized, nil
}

func normalizeCodexCLIOp(raw string) (string, error) {
	op := strings.ToLower(strings.TrimSpace(raw))
	if op == "" {
		return "", fmt.Errorf("op is required")
	}
	if !isCodexCLIOperation(op) {
		return "", fmt.Errorf("unsupported op %q", op)
	}
	if op == codexCLIOpFork {
		return "", fmt.Errorf("fork is interactive-only in Codex CLI 0.130.0")
	}
	return op, nil
}

func normalizedCodexCLIArgsFromDecoded(op string, args codexCLIArgs) normalizedCodexCLIArgs {
	return normalizedCodexCLIArgs{
		Op:                  op,
		Prompt:              strings.TrimSpace(args.Prompt),
		SessionID:           strings.TrimSpace(args.SessionID),
		Cwd:                 strings.TrimSpace(args.Cwd),
		OutputPath:          strings.TrimSpace(args.OutputPath),
		Model:               strings.TrimSpace(args.Model),
		Sandbox:             strings.TrimSpace(args.Sandbox),
		FullAuto:            args.FullAuto,
		SkipGitRepoCheck:    boolOrDefault(args.SkipGitRepoCheck, true),
		JSON:                boolOrDefault(args.JSON, true),
		WaitMSBeforeAsync:   args.WaitMSBeforeAsync,
		WaitDurationSeconds: args.WaitDurationSeconds,
		OutputCharCount:     args.OutputCharacterCount,
	}
}

func validateCodexCLICommonArgs(args normalizedCodexCLIArgs) error {
	if args.Cwd != "" && filepath.IsAbs(args.Cwd) {
		return fmt.Errorf("cwd must be a relative path")
	}
	if args.Sandbox != "" && !isCodexCLISandboxMode(args.Sandbox) {
		return fmt.Errorf("unsupported sandbox %q", args.Sandbox)
	}
	if args.FullAuto != nil {
		return fmt.Errorf("full_auto is removed; use sandbox=workspace-write")
	}
	return nil
}

func normalizeCodexCLITimingArgs(args *normalizedCodexCLIArgs) error {
	waitMSBeforeAsync, err := normalizedNonNegativeInt(
		args.WaitMSBeforeAsync,
		defaultCodexCLIWaitMSBeforeAsync,
		"wait_ms_before_async",
	)
	if err != nil {
		return err
	}
	args.WaitMSBeforeAsync = waitMSBeforeAsync

	waitDurationSeconds, err := normalizedNonNegativeInt(
		args.WaitDurationSeconds,
		defaultCodexCLIWaitDurationSeconds,
		"wait_duration_seconds",
	)
	if err != nil {
		return err
	}
	args.WaitDurationSeconds = waitDurationSeconds

	outputCharCount, err := normalizedNonNegativeInt(
		args.OutputCharCount,
		defaultCodexCLIOutputChars,
		"output_character_count",
	)
	if err != nil {
		return err
	}
	args.OutputCharCount = outputCharCount
	return nil
}

func validateCodexCLIOpRequirements(args normalizedCodexCLIArgs) error {
	switch args.Op {
	case codexCLIOpStart:
		if args.Prompt == "" {
			return fmt.Errorf("prompt is required for start")
		}
	case codexCLIOpResume:
		if args.SessionID == "" {
			return fmt.Errorf("session_id is required for %s", args.Op)
		}
		if args.Prompt == "" {
			return fmt.Errorf("prompt is required for %s", args.Op)
		}
	case codexCLIOpStatus:
		if args.SessionID == "" {
			return fmt.Errorf("session_id is required for status")
		}
	}
	return nil
}

func buildCodexCLIRequest(args normalizedCodexCLIArgs) codexCLIRequest {
	params := map[string]any{"op": args.Op}
	if args.Op == codexCLIOpStatus {
		params["session_id"] = args.SessionID
		params["wait_duration_seconds"] = args.WaitDurationSeconds
		params["output_character_count"] = args.OutputCharCount
		return codexCLIRequest{
			Action:     "CODEX_CLI_STATUS",
			Params:     params,
			OutputPath: args.OutputPath,
		}
	}
	params["prompt"] = args.Prompt
	if args.SessionID != "" {
		params["session_id"] = args.SessionID
	}
	if args.Cwd != "" {
		params["cwd"] = args.Cwd
	}
	if args.OutputPath != "" {
		params["output_path"] = args.OutputPath
	}
	if args.Model != "" {
		params["model"] = args.Model
	}
	if args.Sandbox != "" {
		params["sandbox"] = args.Sandbox
	}
	params["skip_git_repo_check"] = args.SkipGitRepoCheck
	params["json"] = args.JSON
	params["wait_ms_before_async"] = args.WaitMSBeforeAsync
	return codexCLIRequest{
		Action:     "CODEX_CLI_START",
		Params:     params,
		OutputPath: args.OutputPath,
	}
}

func isCodexCLISandboxMode(value string) bool {
	switch value {
	case "read-only", "workspace-write", "danger-full-access":
		return true
	default:
		return false
	}
}
