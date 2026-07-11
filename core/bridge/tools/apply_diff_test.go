package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestApplyDiffToolExecuteSuccess(t *testing.T) {
	tool := NewApplyDiffTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, traceID string) (map[string]any, error) {
			if action != "APPLY_DIFF" {
				t.Fatalf("unexpected action: got %q want %q", action, "APPLY_DIFF")
			}
			if traceID != "trace-diff-1" {
				t.Fatalf("unexpected trace id: got %q want %q", traceID, "trace-diff-1")
			}
			if params["path"] != "README.md" {
				t.Fatalf("unexpected params: %+v", params)
			}
			return map[string]any{
				"message": "applied 1 hunks to README.md",
			}, nil
		},
	})

	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"path":"README.md","diff_text":"@@ -1 +1 @@\n-old\n+new\n"}`),
		"trace-diff-1",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if output != "applied 1 hunks to README.md" {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestApplyDiffToolExecuteRequiresDiffText(t *testing.T) {
	tool := NewApplyDiffTool(mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			return map[string]any{}, nil
		},
	})

	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"README.md","diff_text":" "}`), "trace-diff-2"); err == nil {
		t.Fatal("expected diff_text validation error")
	}
}
