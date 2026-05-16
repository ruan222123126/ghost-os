package tools

import (
	"fmt"
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
	codexExecutablePath, err := optionalStringParam(params, "codex_executable_path")
	if err != nil {
		return codexCLIStartRequest{}, err
	}
	nodeExecutablePath, err := optionalStringParam(params, "node_executable_path")
	if err != nil {
		return codexCLIStartRequest{}, err
	}
	sandbox, err := optionalStringParam(params, "sandbox")
	if err != nil {
		return codexCLIStartRequest{}, err
	}
	return codexCLIStartRequest{
		op:                  op,
		prompt:              prompt,
		sessionID:           sessionID,
		cwd:                 cwd,
		outputPath:          outputPath,
		model:               model,
		codexExecutablePath: codexExecutablePath,
		nodeExecutablePath:  nodeExecutablePath,
		sandbox:             sandbox,
	}, nil
}

func decodeCodexCLIStartFlags(params map[string]any, request *codexCLIStartRequest) error {
	if request == nil {
		return fmt.Errorf("start request is nil")
	}
	fullAuto, err := optionalBoolParam(params, "full_auto")
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

func optionalBoolParam(params map[string]any, field string) (*bool, error) {
	raw, ok := params[field]
	if !ok || raw == nil {
		return nil, nil
	}
	value, ok := raw.(bool)
	if !ok {
		return nil, fmt.Errorf("%s must be a boolean", field)
	}
	return &value, nil
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
