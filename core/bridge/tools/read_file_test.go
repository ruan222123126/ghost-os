package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestReadFileToolExecuteSuccess(t *testing.T) {
	tool := NewReadFileTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, traceID string) (map[string]any, error) {
			if action != "READ_FILE" {
				t.Fatalf("unexpected action: got %q want %q", action, "READ_FILE")
			}
			if traceID != "trace-read-1" {
				t.Fatalf("unexpected trace id: got %q want %q", traceID, "trace-read-1")
			}
			if params["path"] != "README.md" || params["start_line"] != 1 || params["end_line"] != 2 {
				t.Fatalf("unexpected params: %+v", params)
			}
			return map[string]any{
				"path":                 "README.md",
				"requested_start_line": 1,
				"requested_end_line":   2,
				"returned_start_line":  1,
				"returned_end_line":    2,
				"total_lines":          8,
				"content":              "1 | alpha\n2 | beta",
			}, nil
		},
	})

	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"path":"README.md","start_line":1,"end_line":2}`),
		"trace-read-1",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if !strings.Contains(output, "File: README.md") || !strings.Contains(output, "2 | beta") {
		t.Fatalf("unexpected output: %s", output)
	}
	if !strings.Contains(output, "Requested lines: 1-2") || !strings.Contains(output, "Returned lines: 1-2 of 8 total") {
		t.Fatalf("missing line metadata: %s", output)
	}
}

func TestReadFileToolExecuteRequiresPath(t *testing.T) {
	tool := NewReadFileTool(mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			return map[string]any{}, nil
		},
	})

	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"path":" "}`), "trace-read-2"); err == nil {
		t.Fatal("expected path validation error")
	}
}

func TestReadFileToolExecuteRejectsOversizedRange(t *testing.T) {
	tool := NewReadFileTool(mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			t.Fatal("execution client should not be called")
			return nil, nil
		},
	})

	if _, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"path":"README.md","start_line":1,"end_line":250}`),
		"trace-read-3",
	); err == nil {
		t.Fatal("expected range validation error")
	}
}
