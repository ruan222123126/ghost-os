package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestSearchFilesToolExecuteSuccess(t *testing.T) {
	tool := NewSearchFilesTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, traceID string) (map[string]any, error) {
			if action != "SEARCH_FILES" {
				t.Fatalf("unexpected action: got %q want %q", action, "SEARCH_FILES")
			}
			if traceID != "trace-search-1" {
				t.Fatalf("unexpected trace id: got %q want %q", traceID, "trace-search-1")
			}
			if params["keyword"] != "TODO" || params["dir_path"] != "." {
				t.Fatalf("unexpected params: %+v", params)
			}
			return map[string]any{
				"dir_path": ".",
				"matches": []any{
					"src/a.go:10:// TODO: tighten this",
					"src/b.go:22:// TODO: add tests",
				},
			}, nil
		},
	})

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"keyword":"TODO","dir_path":"."}`), "trace-search-1")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if !strings.Contains(output, "src/a.go:10") || !strings.Contains(output, "src/b.go:22") {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestSearchFilesToolExecuteRequiresKeyword(t *testing.T) {
	tool := NewSearchFilesTool(mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			return map[string]any{}, nil
		},
	})

	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"keyword":" "}`), "trace-search-2"); err == nil {
		t.Fatal("expected keyword validation error")
	}
}
