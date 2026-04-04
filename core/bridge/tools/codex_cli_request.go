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
	FullAuto            bool
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
	op := strings.ToLower(strings.TrimSpace(args.Op))
	if op == "" {
		return normalizedCodexCLIArgs{}, fmt.Errorf("op is required")
	}
	if !isCodexCLIOperation(op) {
		return normalizedCodexCLIArgs{}, fmt.Errorf("unsupported op %q", op)
	}

	normalized := normalizedCodexCLIArgs{
		Op:                  op,
		Prompt:              strings.TrimSpace(args.Prompt),
		SessionID:           strings.TrimSpace(args.SessionID),
		Cwd:                 strings.TrimSpace(args.Cwd),
		OutputPath:          strings.TrimSpace(args.OutputPath),
		Model:               strings.TrimSpace(args.Model),
		FullAuto:            boolOrDefault(args.FullAuto, true),
		SkipGitRepoCheck:    boolOrDefault(args.SkipGitRepoCheck, true),
		JSON:                boolOrDefault(args.JSON, true),
		WaitMSBeforeAsync:   args.WaitMSBeforeAsync,
		WaitDurationSeconds: args.WaitDurationSeconds,
		OutputCharCount:     args.OutputCharacterCount,
	}

	if err := validateCodexCLIOpRequirements(normalized); err != nil {
		return normalizedCodexCLIArgs{}, err
	}
	if normalized.Cwd != "" && filepath.IsAbs(normalized.Cwd) {
		return normalizedCodexCLIArgs{}, fmt.Errorf("cwd must be a relative path")
	}
	if normalized.Model == "" {
		normalized.Model = defaultCodexCLIModel
	}
	waitMSBeforeAsync, err := normalizedNonNegativeInt(
		normalized.WaitMSBeforeAsync,
		defaultCodexCLIWaitMSBeforeAsync,
		"wait_ms_before_async",
	)
	if err != nil {
		return normalizedCodexCLIArgs{}, err
	}
	normalized.WaitMSBeforeAsync = waitMSBeforeAsync
	waitDurationSeconds, err := normalizedNonNegativeInt(
		normalized.WaitDurationSeconds,
		defaultCodexCLIWaitDurationSeconds,
		"wait_duration_seconds",
	)
	if err != nil {
		return normalizedCodexCLIArgs{}, err
	}
	normalized.WaitDurationSeconds = waitDurationSeconds
	outputCharCount, err := normalizedNonNegativeInt(
		normalized.OutputCharCount,
		defaultCodexCLIOutputChars,
		"output_character_count",
	)
	if err != nil {
		return normalizedCodexCLIArgs{}, err
	}
	normalized.OutputCharCount = outputCharCount
	return normalized, nil
}

func validateCodexCLIOpRequirements(args normalizedCodexCLIArgs) error {
	switch args.Op {
	case codexCLIOpStart:
		if args.Prompt == "" {
			return fmt.Errorf("prompt is required for start")
		}
	case codexCLIOpResume, codexCLIOpFork:
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
	params["model"] = args.Model
	params["full_auto"] = args.FullAuto
	params["skip_git_repo_check"] = args.SkipGitRepoCheck
	params["json"] = args.JSON
	params["wait_ms_before_async"] = args.WaitMSBeforeAsync
	return codexCLIRequest{
		Action:     "CODEX_CLI_START",
		Params:     params,
		OutputPath: args.OutputPath,
	}
}
