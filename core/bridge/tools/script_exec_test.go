package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type mockExecutionClient struct {
	callFunc func(ctx context.Context, action string, params map[string]any, traceID string) (map[string]any, error)
}

func (m mockExecutionClient) Call(
	ctx context.Context,
	action string,
	params map[string]any,
	traceID string,
) (map[string]any, error) {
	return m.callFunc(ctx, action, params, traceID)
}

func TestScriptExecToolName(t *testing.T) {
	tool := NewScriptExecTool(nil)
	if tool.Name() != "script_exec" {
		t.Fatalf("unexpected tool name: got %q want %q", tool.Name(), "script_exec")
	}
}

func TestScriptExecToolDescriptionListsEnhancedMethods(t *testing.T) {
	tool := NewScriptExecTool(nil)
	description := tool.Description()

	expectedSnippets := []string{
		"primary local workspace tool",
		"Allowed helpers:",
		"tools.bash_exec(",
		"tools.list_files(",
		"tools.read_file(",
		"tools.write_file(",
		"tools.apply_diff(",
		"tools.search_files(",
		"tools.fetch_webpage(",
		"Sandbox limits are enforced",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(description, snippet) {
			t.Fatalf("description missing %q", snippet)
		}
	}
}

func TestScriptExecToolExecuteEmptyScript(t *testing.T) {
	tool := NewScriptExecTool(mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			return map[string]any{}, nil
		},
	})

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"script": ""}`), "trace-test")
	if err == nil {
		t.Fatal("expected error for empty script")
	}
}

func TestScriptExecToolExecuteScriptTooLong(t *testing.T) {
	tool := NewScriptExecTool(mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			return map[string]any{}, nil
		},
	})

	longScript := make([]byte, 11_000)
	for i := range longScript {
		longScript[i] = 'x'
	}
	args, _ := json.Marshal(map[string]string{"script": string(longScript)})

	_, err := tool.Execute(context.Background(), args, "trace-test")
	if err == nil {
		t.Fatal("expected error for oversized script")
	}
}

func TestScriptExecToolExecuteSuccess(t *testing.T) {
	mockClient := mockExecutionClient{
		callFunc: func(_ context.Context, action string, _ map[string]any, traceID string) (map[string]any, error) {
			if action != "SCRIPT_EXEC" {
				t.Fatalf("unexpected action: got %q want %q", action, "SCRIPT_EXEC")
			}
			if traceID != "trace-123" {
				t.Fatalf("unexpected trace id: got %q want %q", traceID, "trace-123")
			}
			return map[string]any{
				"output": "Hello from script",
				"tool_calls_log": []any{
					map[string]any{
						"tool":   "bash_exec",
						"args":   map[string]any{"command": "echo hi"},
						"result": "hi\n",
						"error":  nil,
					},
				},
			}, nil
		},
	}

	tool := NewScriptExecTool(mockClient)
	output, err := tool.Execute(context.Background(), json.RawMessage(`{"script":"print('ok')"}`), "trace-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output == "" {
		t.Fatal("expected non-empty output")
	}
	var report scriptExecReport
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("expected JSON report output: %v", err)
	}
	if report.ScriptOutput != "Hello from script" {
		t.Fatalf("unexpected script output: %q", report.ScriptOutput)
	}
	if report.Summary.StepCount != 1 || report.Summary.FailedSteps != 0 {
		t.Fatalf("unexpected summary: %+v", report.Summary)
	}
	if len(report.Steps) != 1 {
		t.Fatalf("unexpected step count: %d", len(report.Steps))
	}
	if report.Steps[0].Tool != "bash_exec" || report.Steps[0].Status != "success" {
		t.Fatalf("unexpected first step: %+v", report.Steps[0])
	}
	if report.Steps[0].ResultSummary != "hi" {
		t.Fatalf("unexpected step result summary: %q", report.Steps[0].ResultSummary)
	}
}

func TestScriptExecToolExecuteCapturesApplyDiffWriteSummary(t *testing.T) {
	mockClient := mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			return map[string]any{
				"output": "",
				"tool_calls_log": []any{
					map[string]any{
						"tool": "apply_diff",
						"args": map[string]any{
							"path": "/tmp/a.txt",
							"diff_summary": map[string]any{
								"hunk_count":    1,
								"added_lines":   3,
								"removed_lines": 1,
								"hunk_ranges": []any{
									map[string]any{
										"old_start": 10,
										"old_count": 2,
										"new_start": 10,
										"new_count": 4,
									},
								},
							},
						},
						"result": "applied 1 hunks to /tmp/a.txt",
						"error":  nil,
					},
				},
			}, nil
		},
	}

	tool := NewScriptExecTool(mockClient)
	output, err := tool.Execute(context.Background(), json.RawMessage(`{"script":"print('ok')"}`), "trace-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var report scriptExecReport
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("expected JSON report output: %v", err)
	}
	if report.Summary.WriteSteps != 1 {
		t.Fatalf("unexpected write step count: %d", report.Summary.WriteSteps)
	}
	if len(report.Steps) != 1 {
		t.Fatalf("unexpected step count: %d", len(report.Steps))
	}
	change := report.Steps[0].WriteChange
	if change == nil {
		t.Fatal("expected write_change summary")
	}
	if change.Operation != "apply_diff" || change.Path != "/tmp/a.txt" {
		t.Fatalf("unexpected write_change metadata: %+v", change)
	}
	if change.AddedLines != 3 || change.RemovedLines != 1 {
		t.Fatalf("unexpected line delta: %+v", change)
	}
	if len(change.HunkRanges) != 1 || change.HunkRanges[0].OldStart != 10 || change.HunkRanges[0].NewCount != 4 {
		t.Fatalf("unexpected hunk ranges: %+v", change.HunkRanges)
	}
}

func TestScriptExecToolExecuteOmitsUnsetResourceLimits(t *testing.T) {
	var capturedParams map[string]any
	mockClient := mockExecutionClient{
		callFunc: func(_ context.Context, _ string, params map[string]any, _ string) (map[string]any, error) {
			capturedParams = params
			return map[string]any{
				"output":         "",
				"tool_calls_log": []any{},
			}, nil
		},
	}

	tool := NewScriptExecTool(mockClient)
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"script":"print('ok')"}`), "trace-test"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := capturedParams["timeout_ms"]; ok {
		t.Fatal("timeout_ms should be omitted when unset")
	}
	if _, ok := capturedParams["max_memory_mb"]; ok {
		t.Fatal("max_memory_mb should be omitted when unset")
	}
}

func TestScriptExecToolExecuteForwardsResourceLimitsWithoutClamping(t *testing.T) {
	var capturedParams map[string]any
	mockClient := mockExecutionClient{
		callFunc: func(_ context.Context, _ string, params map[string]any, _ string) (map[string]any, error) {
			capturedParams = params
			return map[string]any{
				"output":         "",
				"tool_calls_log": []any{},
			}, nil
		},
	}

	tool := NewScriptExecTool(mockClient)
	if _, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"script":"print('ok')","timeout_ms":120000,"max_memory_mb":2048}`),
		"trace-test",
	); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := capturedParams["timeout_ms"], 120000; got != want {
		t.Fatalf("unexpected timeout_ms forwarding: got %v want %v", got, want)
	}
	if got, want := capturedParams["max_memory_mb"], 2048; got != want {
		t.Fatalf("unexpected max_memory_mb forwarding: got %v want %v", got, want)
	}
}

func TestScriptExecToolExecuteInvalidOutputType(t *testing.T) {
	tool := NewScriptExecTool(mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			return map[string]any{
				"output":         123,
				"tool_calls_log": []any{},
			}, nil
		},
	})

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"script":"print('ok')"}`), "trace-test")
	if err == nil {
		t.Fatal("expected payload validation error")
	}
}

func TestScriptExecToolExecuteMissingToolCallsLog(t *testing.T) {
	tool := NewScriptExecTool(mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			return map[string]any{
				"output": "ok",
			}, nil
		},
	})

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"script":"print('ok')"}`), "trace-test")
	if err == nil {
		t.Fatal("expected payload validation error")
	}
}
