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
		"tools.read_file(",
		"tools.write_file(",
		"tools.apply_diff(",
		"tools.search_files(",
		"tools.fetch_webpage(",
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
