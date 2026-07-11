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
			if params["query"] != "needle" || params["path"] != "." || params["max_results"] != 2 {
				t.Fatalf("unexpected params: %+v", params)
			}
			return map[string]any{
				"matches": []any{
					map[string]any{"path": "a.txt", "line": 1, "text": "needle alpha"},
					map[string]any{"path": "b.txt", "line": 2, "text": "needle beta"},
				},
			}, nil
		},
	})

	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"query":"needle","path":" . ","max_results":2}`),
		"trace-search-1",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	for _, snippet := range []string{"Query: needle", "Path: .", "Matches (2)", "a.txt:1: needle alpha"} {
		if !strings.Contains(output, snippet) {
			t.Fatalf("expected output to contain %q, got %q", snippet, output)
		}
	}
}

func TestSearchFilesToolExecuteTruncatesLongMatchText(t *testing.T) {
	longLine := "needle " + strings.Repeat("x", 500)
	tool := NewSearchFilesTool(mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			return map[string]any{
				"matches": []any{
					map[string]any{"path": "audit.log", "line": 1, "text": longLine},
				},
			}, nil
		},
	})

	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"query":"needle"}`),
		"trace-search-long",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if strings.Contains(output, strings.Repeat("x", 400)) {
		t.Fatalf("expected long match text to be truncated, got %q", output)
	}
	if !strings.Contains(output, "... [truncated]") {
		t.Fatalf("expected truncation marker, got %q", output)
	}
}

func TestSearchFilesToolExecuteRequiresQuery(t *testing.T) {
	tool := NewSearchFilesTool(mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			t.Fatal("execution client should not be called")
			return nil, nil
		},
	})

	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"query":" "}`), "trace-search-2"); err == nil {
		t.Fatal("expected query validation error")
	}
}

func TestSearchFilesToolExecuteRejectsZeroMaxResults(t *testing.T) {
	tool := NewSearchFilesTool(mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			t.Fatal("execution client should not be called")
			return nil, nil
		},
	})

	if _, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"query":"needle","max_results":0}`),
		"trace-search-3",
	); err == nil {
		t.Fatal("expected max_results validation error")
	}
}
