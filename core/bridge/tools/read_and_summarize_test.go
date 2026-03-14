package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"ghost-os/bridge/llm"
)

type mockReadAndSummarizeWorker struct {
	mu        sync.Mutex
	responses []string
	requests  []llm.CompletionRequest
	err       error
}

func (m *mockReadAndSummarizeWorker) Complete(ctx context.Context, request llm.CompletionRequest) (*llm.CompletionResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requests = append(m.requests, request)
	if m.err != nil {
		return nil, m.err
	}
	index := len(m.requests) - 1
	text := fmt.Sprintf("summary-%d", index+1)
	if index < len(m.responses) && strings.TrimSpace(m.responses[index]) != "" {
		text = m.responses[index]
	}
	return &llm.CompletionResponse{
		Message: llm.Message{Role: llm.RoleAssistant, Text: text},
	}, nil
}

func TestReadAndSummarizeToolName(t *testing.T) {
	tool := NewReadAndSummarizeTool(nil, nil, ReadAndSummarizeConfig{})
	if tool.Name() != "read_and_summarize" {
		t.Fatalf("unexpected tool name: got %q want %q", tool.Name(), "read_and_summarize")
	}
}

func TestReadAndSummarizeToolExecuteSummarizesMultipleFiles(t *testing.T) {
	worker := &mockReadAndSummarizeWorker{responses: []string{
		"File one summary",
		"File two summary",
		"Cross-file summary",
	}}
	tool := NewReadAndSummarizeTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, traceID string) (map[string]any, error) {
			if action != "READ_FILE" {
				t.Fatalf("unexpected action: got %q want %q", action, "READ_FILE")
			}
			if traceID != "trace-rs-1" {
				t.Fatalf("unexpected trace id: got %q want %q", traceID, "trace-rs-1")
			}
			path, _ := params["path"].(string)
			start, _ := params["start_line"].(int)
			end, _ := params["end_line"].(int)
			if start != 1 || end != 200 {
				t.Fatalf("unexpected line window for %q: %d-%d", path, start, end)
			}
			switch path {
			case "alpha.go":
				return map[string]any{
					"path":                 "alpha.go",
					"requested_start_line": 1,
					"requested_end_line":   200,
					"returned_start_line":  1,
					"returned_end_line":    18,
					"total_lines":          18,
					"content":              "1 package alpha\n2 func A() {}",
				}, nil
			case "beta.go":
				return map[string]any{
					"path":                 "beta.go",
					"requested_start_line": 1,
					"requested_end_line":   200,
					"returned_start_line":  1,
					"returned_end_line":    24,
					"total_lines":          24,
					"content":              "1 package beta\n2 func B() {}",
				}, nil
			default:
				return nil, fmt.Errorf("unexpected path %q", path)
			}
		},
	}, worker, ReadAndSummarizeConfig{WorkerModel: "gpt-4.1-mini"})

	output, err := tool.Execute(context.Background(), json.RawMessage(`{
		"paths": ["alpha.go", "beta.go"],
		"task": "Find ownership"
	}`), "trace-rs-1")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	for _, snippet := range []string{"Worker model: gpt-4.1-mini", "File: alpha.go", "File one summary", "File: beta.go", "File two summary", "Cross-file synthesis:", "Cross-file summary"} {
		if !strings.Contains(output, snippet) {
			t.Fatalf("output missing %q:\n%s", snippet, output)
		}
	}
	if len(worker.requests) != 3 {
		t.Fatalf("unexpected worker request count: got %d want %d", len(worker.requests), 3)
	}
	matched := false
	for _, request := range worker.requests {
		if len(request.Messages) < 2 {
			continue
		}
		got := request.Messages[1].Text
		if strings.Contains(got, "Find ownership") && strings.Contains(got, "alpha.go") {
			matched = true
			break
		}
	}
	if !matched {
		t.Fatalf("worker prompts missing expected task/path: %+v", worker.requests)
	}
}

func TestReadAndSummarizeToolExecuteHandlesPartialReadFailure(t *testing.T) {
	worker := &mockReadAndSummarizeWorker{responses: []string{"Good file summary"}}
	tool := NewReadAndSummarizeTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, _ string) (map[string]any, error) {
			if action != "READ_FILE" {
				t.Fatalf("unexpected action: got %q want %q", action, "READ_FILE")
			}
			path, _ := params["path"].(string)
			if path == "broken.go" {
				return nil, fmt.Errorf("permission denied")
			}
			return map[string]any{
				"path":                 path,
				"requested_start_line": 1,
				"requested_end_line":   200,
				"returned_start_line":  1,
				"returned_end_line":    12,
				"total_lines":          12,
				"content":              "1 package ok",
			}, nil
		},
	}, worker, ReadAndSummarizeConfig{})

	output, err := tool.Execute(context.Background(), json.RawMessage(`{
		"paths": ["good.go", "broken.go"],
		"task": "Map the flow"
	}`), "trace-rs-2")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if !strings.Contains(output, "File: good.go") || !strings.Contains(output, "Good file summary") {
		t.Fatalf("output missing good file summary:\n%s", output)
	}
	if !strings.Contains(output, "File: broken.go") || !strings.Contains(output, "Status: error") || !strings.Contains(output, "permission denied") {
		t.Fatalf("output missing failure details:\n%s", output)
	}
	if strings.Contains(output, "Cross-file synthesis:") {
		t.Fatalf("single successful file should not produce cross-file synthesis:\n%s", output)
	}
}
