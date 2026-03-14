package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestTextInputToolName(t *testing.T) {
	tool := NewTextInputTool(nil)
	if tool.Name() != "text_input" {
		t.Fatalf("unexpected tool name: got %q want %q", tool.Name(), "text_input")
	}
}

func TestTextInputToolExecuteMapsToExecution(t *testing.T) {
	var gotAction string
	var gotParams map[string]any
	tool := NewTextInputTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, traceID string) (map[string]any, error) {
			gotAction = action
			gotParams = params
			if traceID != "trace-text-1" {
				t.Fatalf("unexpected trace id: got %q want %q", traceID, "trace-text-1")
			}
			return map[string]any{
				"typed":      true,
				"submitted":  true,
				"characters": 11,
				"lines":      2,
			}, nil
		},
	})

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"text":"hello\nworld","submit":true}`), "trace-text-1")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if gotAction != "TEXT_INPUT" {
		t.Fatalf("unexpected native action: got %q want %q", gotAction, "TEXT_INPUT")
	}
	if gotParams["text"] != "hello\nworld" {
		t.Fatalf("unexpected text forwarding: %#v", gotParams["text"])
	}
	if gotParams["submit"] != true {
		t.Fatalf("unexpected submit forwarding: %#v", gotParams["submit"])
	}
	if !strings.Contains(output, `"typed":true`) || !strings.Contains(output, `"lines":2`) {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestTextInputToolRejectsEmptyText(t *testing.T) {
	tool := NewTextInputTool(mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			t.Fatal("execution should not be called")
			return nil, nil
		},
	})

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"text":""}`), "trace-text-2")
	if err == nil || !strings.Contains(err.Error(), "text is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
