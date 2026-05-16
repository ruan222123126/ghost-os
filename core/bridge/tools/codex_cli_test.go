package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

type codexCLICall struct {
	action  string
	params  map[string]any
	traceID string
}

type mockCodexCLIExecutionClient struct {
	calls    []codexCLICall
	callFunc func(context.Context, string, map[string]any, string) (map[string]any, error)
}

func (m *mockCodexCLIExecutionClient) Call(
	ctx context.Context,
	action string,
	params map[string]any,
	traceID string,
) (map[string]any, error) {
	cloned := make(map[string]any, len(params))
	for key, value := range params {
		cloned[key] = value
	}
	m.calls = append(m.calls, codexCLICall{
		action:  action,
		params:  cloned,
		traceID: traceID,
	})
	if m.callFunc != nil {
		return m.callFunc(ctx, action, params, traceID)
	}
	return map[string]any{}, nil
}

func TestCodexCLIToolStartDefaults(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolve cwd failed: %v", err)
	}
	client := &mockCodexCLIExecutionClient{
		callFunc: func(context.Context, string, map[string]any, string) (map[string]any, error) {
			return map[string]any{
				"output_path":    "/tmp/codex-cli.log",
				"exit_code_path": "/tmp/codex-cli.log.exit",
				"output_tail":    "{\"session_id\":\"sess-helper-1\"}\nhelper output",
			}, nil
		},
	}
	tool := newTestCodexCLITool(t, client)

	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"op":"start","prompt":"hello","wait_ms_before_async":1}`),
		"trace-start",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if len(client.calls) != 1 {
		t.Fatalf("expected one execution call, got %d", len(client.calls))
	}
	call := client.calls[0]
	if call.action != "CODEX_CLI_START" {
		t.Fatalf("unexpected action: %s", call.action)
	}
	assertCodexCLIParamValue(t, call.params, "op", codexCLIOpStart)
	assertCodexCLIParamValue(t, call.params, "prompt", "hello")
	assertCodexCLIParamValue(t, call.params, "working_dir", cwd)
	assertCodexCLIParamValue(t, call.params, "use_cwd_flag", false)
	assertCodexCLIParamMissing(t, call.params, "model")
	assertCodexCLIParamMissing(t, call.params, "sandbox")
	assertCodexCLIParamMissing(t, call.params, "full_auto")
	assertCodexCLIParamValue(t, call.params, "skip_git_repo_check", true)
	assertCodexCLIParamValue(t, call.params, "json", true)
	assertCodexCLIParamValue(t, call.params, "wait_ms_before_async", 1)
	assertCodexCLIParamValue(t, call.params, "output_character_count", defaultCodexCLIOutputChars)
	if _, exists := call.params["output_path"]; exists {
		t.Fatalf("unexpected output_path in start params: %+v", call.params)
	}

	result := decodeCodexCLIResult(t, output)
	if result.CommandID == "" {
		t.Fatalf("expected command_id in output: %+v", result)
	}
	if result.Status != "running" {
		t.Fatalf("unexpected status: %+v", result)
	}
	if result.SessionID != "sess-helper-1" {
		t.Fatalf("unexpected session_id: %+v", result)
	}
}

func TestCodexCLIToolStartAndStatusFlow(t *testing.T) {
	client := &mockCodexCLIExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, _ string) (map[string]any, error) {
			switch action {
			case "CODEX_CLI_START":
				return map[string]any{
					"output_path":    "/tmp/codex-cli-start.log",
					"exit_code_path": "/tmp/codex-cli-start.log.exit",
					"output_tail":    `{"type":"thread.started","thread_id":"thread-run-1"}`,
				}, nil
			case "CODEX_CLI_STATUS":
				if params["output_path"] != "/tmp/codex-cli-start.log" {
					return nil, fmt.Errorf("unexpected output_path: %v", params["output_path"])
				}
				if params["exit_code_path"] != "/tmp/codex-cli-start.log.exit" {
					return nil, fmt.Errorf("unexpected exit_code_path: %v", params["exit_code_path"])
				}
				return map[string]any{
					"output_tail":   "done",
					"final_message": "created weather script and ran checks",
					"exit_code":     float64(0),
				}, nil
			default:
				return nil, fmt.Errorf("unexpected action: %s", action)
			}
		},
	}
	tool := newTestCodexCLITool(t, client)

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
	if startResult.SessionID != "thread-run-1" {
		t.Fatalf("unexpected parsed thread id: %+v", startResult)
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
	if statusResult.FinalMessage != "created weather script and ran checks" {
		t.Fatalf("unexpected final_message: %+v", statusResult)
	}
}

func TestCodexCLIToolStartPassesExplicitCodexOptions(t *testing.T) {
	client := &mockCodexCLIExecutionClient{
		callFunc: func(context.Context, string, map[string]any, string) (map[string]any, error) {
			return map[string]any{
				"output_path":    "/tmp/codex-cli.log",
				"exit_code_path": "/tmp/codex-cli.log.exit",
			}, nil
		},
	}
	tool := newTestCodexCLITool(t, client)

	_, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"op":"start","prompt":"hello","model":"gpt-test","sandbox":"workspace-write","wait_ms_before_async":1}`),
		"trace-start",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if len(client.calls) != 1 {
		t.Fatalf("expected one execution call, got %d", len(client.calls))
	}
	assertCodexCLIParamValue(t, client.calls[0].params, "model", "gpt-test")
	assertCodexCLIParamValue(t, client.calls[0].params, "sandbox", "workspace-write")
	assertCodexCLIParamMissing(t, client.calls[0].params, "full_auto")
}

func TestCodexCLIToolStatusUnknownCommand(t *testing.T) {
	client := &mockCodexCLIExecutionClient{}
	tool := newTestCodexCLITool(t, client)
	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"op":"status","session_id":"missing","wait_duration_seconds":1}`),
		"trace-status",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if len(client.calls) != 0 {
		t.Fatalf("unexpected execution call count: %d", len(client.calls))
	}
	result := decodeCodexCLIResult(t, output)
	if result.Status != "error" || result.Message != "command not found" {
		t.Fatalf("unexpected status output: %+v", result)
	}
}

func TestCodexCLIToolValidatesArgs(t *testing.T) {
	client := &mockCodexCLIExecutionClient{}
	tool := NewCodexCLITool(client, "", "")
	cases := []string{
		`{"op":"start"}`,
		`{"op":"resume","prompt":"hi"}`,
		`{"op":"fork","session_id":"sess"}`,
		`{"op":"status"}`,
		`{"op":"unknown"}`,
		`{"op":"start","prompt":"hi","cwd":"/abs/path"}`,
		`{"op":"start","prompt":"hi","sandbox":"bad"}`,
		`{"op":"start","prompt":"hi","sandbox":"workspace-write","full_auto":true}`,
	}
	for _, raw := range cases {
		if _, err := tool.Execute(context.Background(), json.RawMessage(raw), "trace-args"); err == nil {
			t.Fatalf("expected error for args: %s", raw)
		}
	}
}

func TestCodexCLIToolRequiresExecutionClient(t *testing.T) {
	tool := NewCodexCLITool(nil, "", "")
	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"op":"start","prompt":"hello"}`),
		"trace-start",
	)
	if err != nil {
		t.Fatalf("execute should return encoded error result: %v", err)
	}
	result := decodeCodexCLIResult(t, output)
	if result.Status != "error" {
		t.Fatalf("expected error status, got %+v", result)
	}
	if result.Message == "" {
		t.Fatalf("expected explicit error message, got %+v", result)
	}
}

func newTestCodexCLITool(t *testing.T, client ExecutionClient) *CodexCLITool {
	t.Helper()
	tool, ok := NewCodexCLITool(client, "", "").(*CodexCLITool)
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

func assertCodexCLIParamValue(t *testing.T, params map[string]any, key string, expected any) {
	t.Helper()
	actual, ok := params[key]
	if !ok {
		t.Fatalf("missing param %q in %+v", key, params)
	}
	if actual != expected {
		t.Fatalf("unexpected param %q: got=%v want=%v", key, actual, expected)
	}
}

func assertCodexCLIParamMissing(t *testing.T, params map[string]any, key string) {
	t.Helper()
	if _, ok := params[key]; ok {
		t.Fatalf("unexpected param %q in %+v", key, params)
	}
}
