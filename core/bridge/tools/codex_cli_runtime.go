package tools

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

const (
	codexCLIUnknownExitCode      = -1
	codexCLIMaxOutputBufferChars = 4000
	codexCLIStatusPollInterval   = 500 * time.Millisecond
)

type codexCLIStartRequest struct {
	op                string
	prompt            string
	sessionID         string
	cwd               string
	outputPath        string
	model             string
	fullAuto          bool
	skipGitRepoCheck  bool
	jsonFlag          bool
	waitMSBeforeAsync int
	outputCharCount   int
}

type codexCLIStatusRequest struct {
	commandID           string
	sessionID           string
	waitDurationSeconds int
	outputCharCount     int
}

func (t *CodexCLITool) executeStart(ctx context.Context, request codexCLIRequest) codexCLIResult {
	startReq, err := decodeCodexCLIStartRequest(request.Params)
	if err != nil {
		return codexCLIErrorResult(err, request.OutputPath)
	}
	workingDir, outputPath, err := t.resolveStartPaths(startReq.cwd, startReq.outputPath)
	if err != nil {
		return codexCLIErrorResult(err, request.OutputPath)
	}
	command, err := t.startCodexCommand(workingDir, outputPath, startReq)
	if err != nil {
		return codexCLIErrorResult(err, outputPath)
	}
	if err := waitBeforeAsync(ctx, startReq.waitMSBeforeAsync); err != nil {
		return codexCLIErrorResult(err, outputPath)
	}
	status, snapshot := codexCLIStatusFromSnapshot(command, startReq.outputCharCount)
	return codexCLIResult{
		Status:     status,
		CommandID:  command.id,
		SessionID:  snapshot.sessionID,
		ExitCode:   snapshot.exitCode,
		OutputTail: snapshot.outputTail,
		OutputPath: snapshot.outputPath,
	}
}

func (t *CodexCLITool) executeStatus(ctx context.Context, request codexCLIRequest) codexCLIResult {
	statusReq, err := decodeCodexCLIStatusRequest(request.Params)
	if err != nil {
		return codexCLIErrorResult(err, request.OutputPath)
	}
	command := t.manager.find(statusReq.commandID, statusReq.sessionID)
	if command == nil {
		return codexCLIResult{Status: "error", Message: "command not found"}
	}
	result, err := pollCodexCLIStatus(ctx, command, statusReq)
	if err != nil {
		return codexCLIErrorResult(err, command.outputPathSnapshot())
	}
	return result
}

func (t *CodexCLITool) startCodexCommand(
	workingDir string,
	outputPath string,
	startReq codexCLIStartRequest,
) (*codexCLICommand, error) {
	seq, commandID := t.manager.nextCommandID()
	args := buildCodexCLICommandArgs(startReq, workingDir, outputPath)
	commandExec := t.newCodexCommand(args)
	commandExec.Dir = workingDir
	stdout, stderr, err := openCommandPipes(commandExec)
	if err != nil {
		return nil, err
	}
	if err := commandExec.Start(); err != nil {
		return nil, fmt.Errorf("spawn codex failed: %w", err)
	}
	command := newCodexCLICommand(seq, commandID, outputPath, commandExec)
	command.startOutputReaders(stdout, stderr)
	command.startWaiter()
	t.manager.insert(command)
	return command, nil
}

func openCommandPipes(command *exec.Cmd) (io.ReadCloser, io.ReadCloser, error) {
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("attach stdout failed: %w", err)
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("attach stderr failed: %w", err)
	}
	return stdout, stderr, nil
}

func (t *CodexCLITool) newCodexCommand(args []string) *exec.Cmd {
	if t != nil && t.commandFactory != nil {
		return t.commandFactory(codexCLIExecutable, args...)
	}
	return exec.Command(codexCLIExecutable, args...)
}

func buildCodexCLICommandArgs(
	request codexCLIStartRequest,
	workingDir string,
	outputPath string,
) []string {
	args := codexCLIOperationArgs(request)
	if request.fullAuto {
		args = append(args, "--full-auto")
	}
	if request.skipGitRepoCheck {
		args = append(args, "--skip-git-repo-check")
	}
	if request.jsonFlag {
		args = append(args, "--json")
	}
	if request.model != "" {
		args = append(args, "-m", request.model)
	}
	if request.cwd != "" {
		args = append(args, "-C", workingDir)
	}
	if request.outputPath != "" {
		args = append(args, "-o", outputPath)
	}
	return args
}

func codexCLIOperationArgs(request codexCLIStartRequest) []string {
	switch request.op {
	case codexCLIOpResume:
		return []string{"exec", "resume", "--session-id", request.sessionID, request.prompt}
	case codexCLIOpFork:
		return []string{"fork", "--session-id", request.sessionID, request.prompt}
	default:
		return []string{"exec", request.prompt}
	}
}

func codexCLIStatusFromSnapshot(
	command *codexCLICommand,
	outputCharCount int,
) (string, codexCLICommandSnapshot) {
	snapshot := command.snapshot(outputCharCount)
	if snapshot.exitCode != nil {
		return "done", snapshot
	}
	return "running", snapshot
}

func waitBeforeAsync(ctx context.Context, waitMS int) error {
	if waitMS <= 0 {
		return nil
	}
	timer := time.NewTimer(time.Duration(waitMS) * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func pollCodexCLIStatus(
	ctx context.Context,
	command *codexCLICommand,
	request codexCLIStatusRequest,
) (codexCLIResult, error) {
	deadline := time.Now().Add(time.Duration(request.waitDurationSeconds) * time.Second)
	ticker := time.NewTicker(codexCLIStatusPollInterval)
	defer ticker.Stop()
	for {
		status, snapshot := codexCLIStatusFromSnapshot(command, request.outputCharCount)
		if status == "done" || !time.Now().Before(deadline) {
			return codexCLIResult{
				Status:     status,
				CommandID:  command.id,
				SessionID:  snapshot.sessionID,
				ExitCode:   snapshot.exitCode,
				OutputTail: snapshot.outputTail,
				OutputPath: snapshot.outputPath,
			}, nil
		}
		select {
		case <-ctx.Done():
			return codexCLIResult{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func codexCLIErrorResult(err error, outputPath string) codexCLIResult {
	return codexCLIResult{
		Status:     "error",
		Message:    strings.TrimSpace(err.Error()),
		OutputPath: strings.TrimSpace(outputPath),
	}
}
