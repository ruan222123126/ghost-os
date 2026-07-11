package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestListFilesToolExecuteSuccess(t *testing.T) {
	tool := NewListFilesTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, traceID string) (map[string]any, error) {
			if action != "LIST_FILES" {
				t.Fatalf("unexpected action: got %q want %q", action, "LIST_FILES")
			}
			if traceID != "trace-list-1" {
				t.Fatalf("unexpected trace id: got %q want %q", traceID, "trace-list-1")
			}
			if params["path"] != "." {
				t.Fatalf("unexpected params: %+v", params)
			}
			return map[string]any{
				"path":    ".",
				"entries": []any{"README.md", "core/"},
			}, nil
		},
	})

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"."}`), "trace-list-1")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if !strings.Contains(output, "README.md") || !strings.Contains(output, "core/") {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestListFilesToolExecuteRequiresClient(t *testing.T) {
	tool := NewListFilesTool(nil)
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{}`), "trace-list-2"); err == nil {
		t.Fatal("expected execution client error")
	}
}
