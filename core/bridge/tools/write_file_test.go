package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestWriteFileToolExecuteSuccess(t *testing.T) {
	tool := NewWriteFileTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, traceID string) (map[string]any, error) {
			if action != "WRITE_FILE" {
				t.Fatalf("unexpected action: got %q want %q", action, "WRITE_FILE")
			}
			if traceID != "trace-write-1" {
				t.Fatalf("unexpected trace id: got %q want %q", traceID, "trace-write-1")
			}
			if params["path"] != "README.md" || params["content"] != "hello" || params["mode"] != "append" {
				t.Fatalf("unexpected params: %+v", params)
			}
			return map[string]any{
				"message": "wrote 5 bytes to README.md",
			}, nil
		},
	})

	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"path":"README.md","content":"hello","mode":"append"}`),
		"trace-write-1",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if output != "wrote 5 bytes to README.md" {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestWriteFileToolExecuteAllowsEmptyContent(t *testing.T) {
	tool := NewWriteFileTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, _ string) (map[string]any, error) {
			if action != "WRITE_FILE" {
				t.Fatalf("unexpected action: got %q want %q", action, "WRITE_FILE")
			}
			if params["content"] != "" {
				t.Fatalf("expected empty content to pass through, got %+v", params)
			}
			return map[string]any{"message": "wrote 0 bytes to empty.txt"}, nil
		},
	})

	if _, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"path":"empty.txt","content":""}`),
		"trace-write-2",
	); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
}

func TestWriteFileToolExecuteRequiresContentField(t *testing.T) {
	tool := NewWriteFileTool(mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			t.Fatal("execution client should not be called")
			return nil, nil
		},
	})

	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"README.md"}`), "trace-write-3"); err == nil {
		t.Fatal("expected content validation error")
	}
}
