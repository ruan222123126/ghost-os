package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func decodeCodexCLIStartRequest(params map[string]any) (codexCLIStartRequest, error) {
	request, err := decodeCodexCLIStartIdentity(params)
	if err != nil {
		return codexCLIStartRequest{}, err
	}
	if err := decodeCodexCLIStartFlags(params, &request); err != nil {
		return codexCLIStartRequest{}, err
	}
	return request, nil
}

func decodeCodexCLIStartIdentity(params map[string]any) (codexCLIStartRequest, error) {
	op, err := requiredStringParam(params, "op")
	if err != nil {
		return codexCLIStartRequest{}, err
	}
	prompt, err := requiredStringParam(params, "prompt")
	if err != nil {
		return codexCLIStartRequest{}, err
	}
	sessionID, err := optionalStringParam(params, "session_id")
	if err != nil {
		return codexCLIStartRequest{}, err
	}
	cwd, err := optionalStringParam(params, "cwd")
	if err != nil {
		return codexCLIStartRequest{}, err
	}
	outputPath, err := optionalStringParam(params, "output_path")
	if err != nil {
		return codexCLIStartRequest{}, err
	}
	model, err := optionalStringParam(params, "model")
	if err != nil {
		return codexCLIStartRequest{}, err
	}
	return codexCLIStartRequest{
		op:         op,
		prompt:     prompt,
		sessionID:  sessionID,
		cwd:        cwd,
		outputPath: outputPath,
		model:      model,
	}, nil
}

func decodeCodexCLIStartFlags(params map[string]any, request *codexCLIStartRequest) error {
	if request == nil {
		return fmt.Errorf("start request is nil")
	}
	fullAuto, err := boolWithDefault(params, "full_auto", true)
	if err != nil {
		return err
	}
	skipRepo, err := boolWithDefault(params, "skip_git_repo_check", true)
	if err != nil {
		return err
	}
	jsonFlag, err := boolWithDefault(params, "json", true)
	if err != nil {
		return err
	}
	waitMS, err := intWithDefault(params, "wait_ms_before_async", defaultCodexCLIWaitMSBeforeAsync)
	if err != nil {
		return err
	}
	outputChars, err := intWithDefault(params, "output_character_count", defaultCodexCLIOutputChars)
	if err != nil {
		return err
	}
	request.fullAuto = fullAuto
	request.skipGitRepoCheck = skipRepo
	request.jsonFlag = jsonFlag
	request.waitMSBeforeAsync = waitMS
	request.outputCharCount = outputChars
	return nil
}

func decodeCodexCLIStatusRequest(params map[string]any) (codexCLIStatusRequest, error) {
	commandID, err := optionalStringParam(params, "command_id")
	if err != nil {
		return codexCLIStatusRequest{}, err
	}
	sessionID, err := optionalStringParam(params, "session_id")
	if err != nil {
		return codexCLIStatusRequest{}, err
	}
	if commandID == "" && sessionID == "" {
		return codexCLIStatusRequest{}, fmt.Errorf("command_id or session_id is required")
	}
	waitSecs, err := intWithDefault(params, "wait_duration_seconds", defaultCodexCLIWaitDurationSeconds)
	if err != nil {
		return codexCLIStatusRequest{}, err
	}
	outputChars, err := intWithDefault(params, "output_character_count", defaultCodexCLIOutputChars)
	if err != nil {
		return codexCLIStatusRequest{}, err
	}
	return codexCLIStatusRequest{
		commandID:           commandID,
		sessionID:           sessionID,
		waitDurationSeconds: waitSecs,
		outputCharCount:     outputChars,
	}, nil
}

func requiredStringParam(params map[string]any, field string) (string, error) {
	value, err := optionalStringParam(params, field)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	return value, nil
}

func optionalStringParam(params map[string]any, field string) (string, error) {
	raw, ok := params[field]
	if !ok || raw == nil {
		return "", nil
	}
	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", field)
	}
	return strings.TrimSpace(value), nil
}

func boolWithDefault(params map[string]any, field string, fallback bool) (bool, error) {
	raw, ok := params[field]
	if !ok || raw == nil {
		return fallback, nil
	}
	value, ok := raw.(bool)
	if !ok {
		return false, fmt.Errorf("%s must be a boolean", field)
	}
	return value, nil
}

func intWithDefault(params map[string]any, field string, fallback int) (int, error) {
	raw, ok := params[field]
	if !ok || raw == nil {
		return fallback, nil
	}
	value, err := asNonNegativeInt(raw, field)
	if err != nil {
		return 0, err
	}
	if value == 0 {
		return fallback, nil
	}
	return value, nil
}

func asNonNegativeInt(raw any, field string) (int, error) {
	switch value := raw.(type) {
	case int:
		if value < 0 {
			return 0, fmt.Errorf("%s must be >= 0", field)
		}
		return value, nil
	case int32:
		if value < 0 {
			return 0, fmt.Errorf("%s must be >= 0", field)
		}
		return int(value), nil
	case int64:
		if value < 0 {
			return 0, fmt.Errorf("%s must be >= 0", field)
		}
		return int(value), nil
	case float64:
		if value < 0 {
			return 0, fmt.Errorf("%s must be >= 0", field)
		}
		return int(value), nil
	default:
		return 0, fmt.Errorf("%s must be a non-negative integer", field)
	}
}

func (t *CodexCLITool) resolveStartPaths(cwd string, outputPath string) (string, string, error) {
	baseDir, err := t.resolveBaseWorkingDir()
	if err != nil {
		return "", "", err
	}
	workingDir, err := resolveStartWorkingDir(cwd, baseDir)
	if err != nil {
		return "", "", err
	}
	if err := t.validateReadPath(workingDir); err != nil {
		return "", "", err
	}
	resolvedOutputPath, err := resolveOptionalOutputPath(outputPath, baseDir)
	if err != nil {
		return "", "", err
	}
	if err := t.validateWritePath(resolvedOutputPath); err != nil {
		return "", "", err
	}
	return workingDir, resolvedOutputPath, nil
}

func resolveStartWorkingDir(rawCwd string, baseDir string) (string, error) {
	trimmed := strings.TrimSpace(rawCwd)
	if trimmed == "" {
		if err := ensureDirExists(baseDir); err != nil {
			return "", err
		}
		return baseDir, nil
	}
	resolved, err := resolvePathFromBase(trimmed, baseDir)
	if err != nil {
		return "", err
	}
	if err := ensureDirExists(resolved); err != nil {
		return "", err
	}
	return resolved, nil
}

func resolveOptionalOutputPath(rawOutputPath string, baseDir string) (string, error) {
	trimmed := strings.TrimSpace(rawOutputPath)
	if trimmed == "" {
		return "", nil
	}
	return resolvePathFromBase(trimmed, baseDir)
}

func (t *CodexCLITool) resolveBaseWorkingDir() (string, error) {
	if t != nil && t.resolveProjectRoot != nil {
		root, err := t.resolveProjectRoot()
		if err != nil {
			return "", fmt.Errorf("resolve project root failed: %w", err)
		}
		if strings.TrimSpace(root) != "" {
			resolved, err := resolveAbsolutePath(root)
			if err != nil {
				return "", err
			}
			if err := ensureDirExists(resolved); err != nil {
				return "", err
			}
			return resolved, nil
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve cwd: %w", err)
	}
	resolved, err := resolveAbsolutePath(cwd)
	if err != nil {
		return "", err
	}
	if err := ensureDirExists(resolved); err != nil {
		return "", err
	}
	return resolved, nil
}

func resolvePathFromBase(pathValue string, baseDir string) (string, error) {
	if filepath.IsAbs(pathValue) {
		return resolveAbsolutePath(pathValue)
	}
	return resolveAbsolutePath(filepath.Join(baseDir, pathValue))
}

func (t *CodexCLITool) validateReadPath(path string) error {
	roots, err := t.allowedReadRoots()
	if err != nil {
		return err
	}
	if pathWithinAnyRoot(path, roots) {
		return nil
	}
	return fmt.Errorf("cwd is outside allowed read paths")
}

func (t *CodexCLITool) validateWritePath(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	roots, err := t.allowedWriteRoots()
	if err != nil {
		return err
	}
	if pathWithinAnyRoot(path, roots) {
		return nil
	}
	return fmt.Errorf("output_path is outside allowed write paths")
}

func (t *CodexCLITool) allowedReadRoots() ([]string, error) {
	if t == nil || len(t.allowedReadPaths) == 0 {
		return defaultAllowedRoots()
	}
	return normalizeAllowedRoots(t.allowedReadPaths)
}

func (t *CodexCLITool) allowedWriteRoots() ([]string, error) {
	if t == nil || len(t.allowedWritePaths) == 0 {
		return defaultAllowedRoots()
	}
	return normalizeAllowedRoots(t.allowedWritePaths)
}
