package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestCodexCLIToolStartDefaults(t *testing.T) {
	tool := newTestCodexCLITool(t)
	var gotName string
	var gotArgs []string
	tool.commandFactory = func(name string, args ...string) *exec.Cmd {
		gotName = name
		gotArgs = append([]string(nil), args...)
		return codexCLIHelperCommand("session_done")
	}

	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"op":"start","prompt":"hello","wait_ms_before_async":1}`),
		"trace-start",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if gotName != codexCLIExecutable {
		t.Fatalf("unexpected command name: got %q", gotName)
	}
	assertContainsArgSequence(t, gotArgs, []string{"exec", "hello"})
	assertContainsArgSequence(t, gotArgs, []string{"--full-auto"})
	assertContainsArgSequence(t, gotArgs, []string{"--skip-git-repo-check"})
	assertContainsArgSequence(t, gotArgs, []string{"--json"})
	assertContainsArgSequence(t, gotArgs, []string{"-m", defaultCodexCLIModel})
	if containsArg(gotArgs, "-C") {
		t.Fatalf("unexpected -C in default args: %v", gotArgs)
	}
	if containsArg(gotArgs, "-o") {
		t.Fatalf("unexpected -o in default args: %v", gotArgs)
	}

	result := decodeCodexCLIResult(t, output)
	if result.CommandID == "" {
		t.Fatalf("expected command_id in output: %+v", result)
	}
	if result.Status != "running" && result.Status != "done" {
		t.Fatalf("unexpected status: %+v", result)
	}
}

func TestCodexCLIToolStartAndStatusFlow(t *testing.T) {
	tool := newTestCodexCLITool(t)
	tool.commandFactory = func(_ string, _ ...string) *exec.Cmd {
		return codexCLIHelperCommand("session_done")
	}

	startOutput, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"op":"start","prompt":"hi","wait_ms_before_async":1}`),
		"trace-start",
	)
	if err != nil {
		t.Fatalf("start returned error: %v", err)
	}
	startResult := decodeCodexCLIResult(t, startOutput)
	if startResult.CommandID == "" {
		t.Fatalf("missing command_id: %+v", startResult)
	}

	statusOutput, err := tool.Execute(
		context.Background(),
		json.RawMessage(fmt.Sprintf(`{"op":"status","session_id":"%s","wait_duration_seconds":1}`, startResult.CommandID)),
		"trace-status",
	)
	if err != nil {
		t.Fatalf("status returned error: %v", err)
	}
	statusResult := decodeCodexCLIResult(t, statusOutput)
	if statusResult.Status != "done" {
		t.Fatalf("expected done status, got %+v", statusResult)
	}
	if statusResult.CommandID != startResult.CommandID {
		t.Fatalf("command_id mismatch: start=%q status=%q", startResult.CommandID, statusResult.CommandID)
	}
	if statusResult.ExitCode == nil || *statusResult.ExitCode != 0 {
		t.Fatalf("expected exit_code=0, got %+v", statusResult)
	}
}

func TestCodexCLIToolStatusUnknownCommand(t *testing.T) {
	tool := newTestCodexCLITool(t)
	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"op":"status","session_id":"missing","wait_duration_seconds":1}`),
		"trace-status",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	result := decodeCodexCLIResult(t, output)
	if result.Status != "error" || result.Message != "command not found" {
		t.Fatalf("unexpected status output: %+v", result)
	}
}

func TestCodexCLIToolValidatesArgs(t *testing.T) {
	tool := NewCodexCLITool(nil, false)
	cases := []string{
		`{"op":"start"}`,
		`{"op":"resume","prompt":"hi"}`,
		`{"op":"fork","session_id":"sess"}`,
		`{"op":"status"}`,
		`{"op":"unknown"}`,
		`{"op":"start","prompt":"hi","cwd":"/abs/path"}`,
	}

	for _, raw := range cases {
		if _, err := tool.Execute(context.Background(), json.RawMessage(raw), "trace-args"); err == nil {
			t.Fatalf("expected error for args: %s", raw)
		}
	}
}

func TestCodexCLIHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_CODEX_CLI_HELPER") != "1" {
		return
	}
	switch os.Getenv("GO_CODEX_CLI_HELPER_MODE") {
	case "session_done":
		_, _ = fmt.Fprintln(os.Stdout, `{"session_id":"sess-helper-1"}`)
		_, _ = fmt.Fprintln(os.Stdout, "helper output")
		os.Exit(0)
	default:
		_, _ = fmt.Fprintln(os.Stderr, "unknown helper mode")
		os.Exit(2)
	}
}

func newTestCodexCLITool(t *testing.T) *CodexCLITool {
	t.Helper()
	tool, ok := NewCodexCLITool(nil, false).(*CodexCLITool)
	if !ok {
		t.Fatal("expected *CodexCLITool")
	}
	return tool
}

func decodeCodexCLIResult(t *testing.T, output string) codexCLIResult {
	t.Helper()
	var result codexCLIResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode output failed: %v", err)
	}
	return result
}

func codexCLIHelperCommand(mode string) *exec.Cmd {
	cmd := exec.Command(os.Args[0], "-test.run=^TestCodexCLIHelperProcess$")
	cmd.Env = append(os.Environ(),
		"GO_WANT_CODEX_CLI_HELPER=1",
		"GO_CODEX_CLI_HELPER_MODE="+mode,
	)
	return cmd
}

func assertContainsArgSequence(t *testing.T, args []string, sequence []string) {
	t.Helper()
	if len(sequence) == 0 {
		return
	}
	for index := 0; index <= len(args)-len(sequence); index++ {
		if matchesArgSequence(args[index:index+len(sequence)], sequence) {
			return
		}
	}
	t.Fatalf("sequence %v not found in args: %v", sequence, args)
}

func matchesArgSequence(actual []string, expected []string) bool {
	for index := range expected {
		if strings.TrimSpace(actual[index]) != strings.TrimSpace(expected[index]) {
			return false
		}
	}
	return true
}

func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if strings.TrimSpace(arg) == target {
			return true
		}
	}
	return false
}
