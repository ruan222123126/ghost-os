package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestCodexCLIToolRequiresPersistent(t *testing.T) {
	tool := NewCodexCLITool(mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			return map[string]any{}, nil
		},
	}, false)

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"op":"start","prompt":"hello"}`), "trace-1")
	if err == nil {
		t.Fatal("expected native_persistent error")
	}
}

func TestCodexCLIToolStartDefaults(t *testing.T) {
	var gotAction string
	var gotParams map[string]any

	tool := NewCodexCLITool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, _ string) (map[string]any, error) {
			gotAction = action
			gotParams = params
			return map[string]any{
				"status":     "running",
				"command_id": "cmd-1",
			}, nil
		},
	}, true)

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"op":"start","prompt":"hello"}`), "trace-start")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if gotAction != "CODEX_CLI_START" {
		t.Fatalf("unexpected action: got %q", gotAction)
	}
	if gotParams["op"] != "start" {
		t.Fatalf("unexpected op param: %+v", gotParams)
	}
	if gotParams["prompt"] != "hello" {
		t.Fatalf("unexpected prompt param: %+v", gotParams)
	}
	if gotParams["model"] != "gpt-5.4" {
		t.Fatalf("unexpected model default: %+v", gotParams)
	}
	if gotParams["full_auto"] != true {
		t.Fatalf("unexpected full_auto default: %+v", gotParams)
	}
	if gotParams["skip_git_repo_check"] != true {
		t.Fatalf("unexpected skip_git_repo_check default: %+v", gotParams)
	}
	if gotParams["json"] != true {
		t.Fatalf("unexpected json default: %+v", gotParams)
	}
	if gotParams["wait_ms_before_async"] != defaultCodexCLIWaitMSBeforeAsync {
		t.Fatalf("unexpected wait_ms_before_async default: %+v", gotParams)
	}

	var decoded codexCLIResult
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if decoded.Status != "running" || decoded.CommandID != "cmd-1" {
		t.Fatalf("unexpected output: %+v", decoded)
	}
}

func TestCodexCLIToolStatusDefaults(t *testing.T) {
	var gotAction string
	var gotParams map[string]any

	tool := NewCodexCLITool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, _ string) (map[string]any, error) {
			gotAction = action
			gotParams = params
			return map[string]any{
				"status":     "running",
				"command_id": "cmd-2",
			}, nil
		},
	}, true)

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"op":"status","session_id":"cmd-2"}`), "trace-status")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if gotAction != "CODEX_CLI_STATUS" {
		t.Fatalf("unexpected action: got %q", gotAction)
	}
	if gotParams["session_id"] != "cmd-2" {
		t.Fatalf("unexpected status params: %+v", gotParams)
	}
	if _, ok := gotParams["command_id"]; ok {
		t.Fatalf("unexpected command_id param: %+v", gotParams)
	}
	if gotParams["wait_duration_seconds"] != defaultCodexCLIWaitDurationSeconds {
		t.Fatalf("unexpected wait_duration_seconds default: %+v", gotParams)
	}
	if gotParams["output_character_count"] != defaultCodexCLIOutputChars {
		t.Fatalf("unexpected output_character_count default: %+v", gotParams)
	}
}

func TestCodexCLIToolValidatesArgs(t *testing.T) {
	tool := NewCodexCLITool(mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			return map[string]any{}, nil
		},
	}, true)

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
