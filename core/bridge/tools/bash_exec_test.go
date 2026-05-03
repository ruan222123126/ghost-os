package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestBashExecToolExecuteSuccess(t *testing.T) {
	tool := NewBashExecTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, traceID string) (map[string]any, error) {
			if action != "BASH_EXEC" {
				t.Fatalf("unexpected action: got %q want %q", action, "BASH_EXEC")
			}
			if traceID != "trace-bash-1" {
				t.Fatalf("unexpected trace id: got %q want %q", traceID, "trace-bash-1")
			}
			if params["command"] != "pwd" || params["max_output_chars"] != 512 {
				t.Fatalf("unexpected params: %+v", params)
			}
			return map[string]any{
				"stdout": "/tmp\n",
				"stderr": "",
			}, nil
		},
	})

	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"command":" pwd ","max_output_chars":512}`),
		"trace-bash-1",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if output != "/tmp\n" {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestBashExecToolExecuteRequiresCommand(t *testing.T) {
	tool := NewBashExecTool(mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			t.Fatal("execution client should not be called")
			return nil, nil
		},
	})

	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"command":" "}`), "trace-bash-2"); err == nil {
		t.Fatal("expected command validation error")
	}
}

func TestBashExecToolExecuteRejectsZeroOutputLimit(t *testing.T) {
	tool := NewBashExecTool(mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			t.Fatal("execution client should not be called")
			return nil, nil
		},
	})

	if _, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"command":"pwd","max_output_chars":0}`),
		"trace-bash-3",
	); err == nil {
		t.Fatal("expected max_output_chars validation error")
	}
}
