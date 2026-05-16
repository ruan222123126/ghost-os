package tools

import (
	"context"
	"fmt"
	"strings"
)

type codexCLIStartRequest struct {
	op                  string
	prompt              string
	sessionID           string
	cwd                 string
	outputPath          string
	model               string
	codexExecutablePath string
	nodeExecutablePath  string
	sandbox             string
	fullAuto            *bool
	skipGitRepoCheck    bool
	jsonFlag            bool
	waitMSBeforeAsync   int
	outputCharCount     int
}

type codexCLIStatusRequest struct {
	commandID           string
	sessionID           string
	waitDurationSeconds int
	outputCharCount     int
}

type codexCLIStartExecutionPayload struct {
	outputPath   string
	exitCodePath string
	outputTail   string
	finalMessage string
	exitCode     *int
}

type codexCLIStatusExecutionPayload struct {
	outputTail   string
	finalMessage string
	exitCode     *int
}

type codexCLIStartExecutionInput struct {
	request    codexCLIStartRequest
	workingDir string
	outputPath string
}

type codexCLIStatusExecutionInput struct {
	command *codexCLICommand
	request codexCLIStatusRequest
}

func (t *CodexCLITool) executeStart(ctx context.Context, request codexCLIRequest, traceID string) codexCLIResult {
	startReq, err := decodeCodexCLIStartRequest(request.Params)
	if err != nil {
		return codexCLIErrorResult(err, request.OutputPath)
	}
	if startReq.codexExecutablePath == "" {
		startReq.codexExecutablePath = strings.TrimSpace(t.codexExecutablePath)
	}
	if startReq.nodeExecutablePath == "" {
		startReq.nodeExecutablePath = strings.TrimSpace(t.nodeExecutablePath)
	}
	workingDir, outputPath, err := t.resolveStartPaths(startReq.cwd, startReq.outputPath)
	if err != nil {
		return codexCLIErrorResult(err, request.OutputPath)
	}
	payload, err := t.startWithExecution(ctx, traceID, codexCLIStartExecutionInput{
		request:    startReq,
		workingDir: workingDir,
		outputPath: outputPath,
	})
	if err != nil {
		return codexCLIErrorResult(err, outputPath)
	}
	seq, commandID := t.manager.nextCommandID()
	command := newCodexCLICommand(seq, commandID, payload.outputPath, payload.exitCodePath)
	command.applyStatus(payload.outputTail, payload.exitCode)
	t.manager.insert(command)
	status, snapshot := codexCLIStatusFromSnapshot(command)
	return codexCLIResult{
		Status:       status,
		CommandID:    command.id,
		SessionID:    snapshot.sessionID,
		ExitCode:     snapshot.exitCode,
		OutputTail:   strings.TrimSpace(payload.outputTail),
		FinalMessage: strings.TrimSpace(payload.finalMessage),
		OutputPath:   snapshot.outputPath,
	}
}

func (t *CodexCLITool) executeStatus(ctx context.Context, request codexCLIRequest, traceID string) codexCLIResult {
	statusReq, err := decodeCodexCLIStatusRequest(request.Params)
	if err != nil {
		return codexCLIErrorResult(err, request.OutputPath)
	}
	command := t.manager.find(statusReq.commandID, statusReq.sessionID)
	if command == nil {
		return codexCLIResult{Status: "error", Message: "command not found"}
	}
	payload, err := t.statusWithExecution(ctx, traceID, codexCLIStatusExecutionInput{
		command: command,
		request: statusReq,
	})
	if err != nil {
		return codexCLIErrorResult(err, command.outputPathSnapshot())
	}
	command.applyStatus(payload.outputTail, payload.exitCode)
	status, snapshot := codexCLIStatusFromSnapshot(command)
	return codexCLIResult{
		Status:       status,
		CommandID:    command.id,
		SessionID:    snapshot.sessionID,
		ExitCode:     snapshot.exitCode,
		OutputTail:   strings.TrimSpace(payload.outputTail),
		FinalMessage: strings.TrimSpace(payload.finalMessage),
		OutputPath:   snapshot.outputPath,
	}
}

func (t *CodexCLITool) startWithExecution(
	ctx context.Context,
	traceID string,
	input codexCLIStartExecutionInput,
) (codexCLIStartExecutionPayload, error) {
	if t == nil || t.execution == nil {
		return codexCLIStartExecutionPayload{}, fmt.Errorf("execution client is not configured")
	}
	params := buildCodexCLIStartExecutionParams(input.request, input.workingDir, input.outputPath)
	payload, err := t.execution.Call(ctx, "CODEX_CLI_START", params, traceID)
	if err != nil {
		return codexCLIStartExecutionPayload{}, fmt.Errorf("execution CODEX_CLI_START failed: %w", err)
	}
	return decodeCodexCLIStartExecutionPayload(payload)
}

func (t *CodexCLITool) statusWithExecution(
	ctx context.Context,
	traceID string,
	input codexCLIStatusExecutionInput,
) (codexCLIStatusExecutionPayload, error) {
	if t == nil || t.execution == nil {
		return codexCLIStatusExecutionPayload{}, fmt.Errorf("execution client is not configured")
	}
	params, err := buildCodexCLIStatusExecutionParams(input.command, input.request)
	if err != nil {
		return codexCLIStatusExecutionPayload{}, err
	}
	payload, err := t.execution.Call(ctx, "CODEX_CLI_STATUS", params, traceID)
	if err != nil {
		return codexCLIStatusExecutionPayload{}, fmt.Errorf("execution CODEX_CLI_STATUS failed: %w", err)
	}
	return decodeCodexCLIStatusExecutionPayload(payload)
}

func buildCodexCLIStartExecutionParams(
	request codexCLIStartRequest,
	workingDir string,
	outputPath string,
) map[string]any {
	params := map[string]any{
		"op":                     request.op,
		"prompt":                 request.prompt,
		"working_dir":            workingDir,
		"use_cwd_flag":           request.cwd != "",
		"skip_git_repo_check":    request.skipGitRepoCheck,
		"json":                   request.jsonFlag,
		"wait_ms_before_async":   request.waitMSBeforeAsync,
		"output_character_count": request.outputCharCount,
	}
	if request.model != "" {
		params["model"] = request.model
	}
	if request.codexExecutablePath != "" {
		params["codex_executable_path"] = request.codexExecutablePath
	}
	if request.nodeExecutablePath != "" {
		params["node_executable_path"] = request.nodeExecutablePath
	}
	if request.sandbox != "" {
		params["sandbox"] = request.sandbox
	}
	if request.fullAuto != nil {
		params["full_auto"] = *request.fullAuto
	}
	if request.sessionID != "" {
		params["session_id"] = request.sessionID
	}
	if outputPath != "" {
		params["output_path"] = outputPath
	}
	return params
}

func buildCodexCLIStatusExecutionParams(
	command *codexCLICommand,
	request codexCLIStatusRequest,
) (map[string]any, error) {
	if command == nil {
		return nil, fmt.Errorf("command is nil")
	}
	snapshot := command.snapshot()
	if strings.TrimSpace(snapshot.outputPath) == "" {
		return nil, fmt.Errorf("command output path is empty")
	}
	if strings.TrimSpace(snapshot.exitCodePath) == "" {
		return nil, fmt.Errorf("command exit code path is empty")
	}
	return map[string]any{
		"output_path":            snapshot.outputPath,
		"exit_code_path":         snapshot.exitCodePath,
		"wait_duration_seconds":  request.waitDurationSeconds,
		"output_character_count": request.outputCharCount,
	}, nil
}

func decodeCodexCLIStartExecutionPayload(payload map[string]any) (codexCLIStartExecutionPayload, error) {
	outputPath, err := requiredPayloadString(payload, "output_path")
	if err != nil {
		return codexCLIStartExecutionPayload{}, err
	}
	exitCodePath, err := requiredPayloadString(payload, "exit_code_path")
	if err != nil {
		return codexCLIStartExecutionPayload{}, err
	}
	outputTail, err := optionalPayloadString(payload, "output_tail")
	if err != nil {
		return codexCLIStartExecutionPayload{}, err
	}
	exitCode, err := optionalPayloadInt(payload, "exit_code")
	if err != nil {
		return codexCLIStartExecutionPayload{}, err
	}
	finalMessage, err := optionalPayloadString(payload, "final_message")
	if err != nil {
		return codexCLIStartExecutionPayload{}, err
	}
	return codexCLIStartExecutionPayload{
		outputPath:   outputPath,
		exitCodePath: exitCodePath,
		outputTail:   outputTail,
		finalMessage: finalMessage,
		exitCode:     exitCode,
	}, nil
}

func decodeCodexCLIStatusExecutionPayload(payload map[string]any) (codexCLIStatusExecutionPayload, error) {
	outputTail, err := optionalPayloadString(payload, "output_tail")
	if err != nil {
		return codexCLIStatusExecutionPayload{}, err
	}
	exitCode, err := optionalPayloadInt(payload, "exit_code")
	if err != nil {
		return codexCLIStatusExecutionPayload{}, err
	}
	finalMessage, err := optionalPayloadString(payload, "final_message")
	if err != nil {
		return codexCLIStatusExecutionPayload{}, err
	}
	return codexCLIStatusExecutionPayload{
		outputTail:   outputTail,
		finalMessage: finalMessage,
		exitCode:     exitCode,
	}, nil
}

func codexCLIStatusFromSnapshot(command *codexCLICommand) (string, codexCLICommandSnapshot) {
	snapshot := command.snapshot()
	if snapshot.exitCode != nil {
		return "done", snapshot
	}
	return "running", snapshot
}

func codexCLIErrorResult(err error, outputPath string) codexCLIResult {
	return codexCLIResult{
		Status:     "error",
		Message:    strings.TrimSpace(err.Error()),
		OutputPath: strings.TrimSpace(outputPath),
	}
}
