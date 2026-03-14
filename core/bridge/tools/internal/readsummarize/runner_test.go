package readsummarize

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"ghost-os/bridge/llm"
)

type mockExecutionClient struct {
	callFunc func(context.Context, string, map[string]any, string) (map[string]any, error)
}

func (m mockExecutionClient) Call(ctx context.Context, action string, params map[string]any, traceID string) (map[string]any, error) {
	return m.callFunc(ctx, action, params, traceID)
}

type mockWorker struct {
	mu        sync.Mutex
	responses []string
	requests  []llm.CompletionRequest
}

func (m *mockWorker) Complete(_ context.Context, request llm.CompletionRequest) (*llm.CompletionResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = append(m.requests, request)
	idx := len(m.requests) - 1
	text := fmt.Sprintf("summary-%d", idx+1)
	if idx < len(m.responses) && strings.TrimSpace(m.responses[idx]) != "" {
		text = m.responses[idx]
	}
	return &llm.CompletionResponse{Message: llm.Message{Role: llm.RoleAssistant, Text: text}}, nil
}

func TestChunkReaderMarksTruncatedWhenBudgetExhausted(t *testing.T) {
	reader := NewChunkReader(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, traceID string) (map[string]any, error) {
			if action != "READ_FILE" {
				t.Fatalf("unexpected action: got %q want %q", action, "READ_FILE")
			}
			if traceID != "trace-rs-3" {
				t.Fatalf("unexpected trace id: got %q want %q", traceID, "trace-rs-3")
			}
			start, _ := params["start_line"].(int)
			switch start {
			case 1:
				return map[string]any{
					"path":                 "large.go",
					"requested_start_line": 1,
					"requested_end_line":   200,
					"returned_start_line":  1,
					"returned_end_line":    200,
					"total_lines":          450,
					"content":              "chunk-1",
				}, nil
			case 201:
				return map[string]any{
					"path":                 "large.go",
					"requested_start_line": 201,
					"requested_end_line":   400,
					"returned_start_line":  201,
					"returned_end_line":    400,
					"total_lines":          450,
					"content":              "chunk-2",
				}, nil
			default:
				t.Fatalf("unexpected start line: %d", start)
				return nil, nil
			}
		},
	})

	chunks, truncated, err := reader.Read(context.Background(), "large.go", 2, "trace-rs-3")
	if err != nil {
		t.Fatalf("read returned error: %v", err)
	}
	if !truncated {
		t.Fatal("expected truncated result when chunk budget is exhausted")
	}
	if len(chunks) != 2 {
		t.Fatalf("unexpected chunk count: got %d want %d", len(chunks), 2)
	}
	if chunks[1].EndLine != 400 || chunks[1].TotalLines != 450 {
		t.Fatalf("unexpected final chunk metadata: %+v", chunks[1])
	}
}

func TestRunnerSummarizeFileUsesChunkAggregation(t *testing.T) {
	worker := &mockWorker{responses: []string{
		"Chunk one summary",
		"Chunk two summary",
		"Merged file summary",
	}}
	runner := NewRunner(
		NewChunkReader(mockExecutionClient{
			callFunc: func(_ context.Context, action string, params map[string]any, _ string) (map[string]any, error) {
				if action != "READ_FILE" {
					t.Fatalf("unexpected action: got %q want %q", action, "READ_FILE")
				}
				start, _ := params["start_line"].(int)
				switch start {
				case 1:
					return map[string]any{
						"path":                 "chunked.go",
						"requested_start_line": 1,
						"requested_end_line":   200,
						"returned_start_line":  1,
						"returned_end_line":    200,
						"total_lines":          450,
						"content":              "chunk-1 body",
					}, nil
				case 201:
					return map[string]any{
						"path":                 "chunked.go",
						"requested_start_line": 201,
						"requested_end_line":   400,
						"returned_start_line":  201,
						"returned_end_line":    400,
						"total_lines":          450,
						"content":              "chunk-2 body",
					}, nil
				default:
					t.Fatalf("unexpected start line: %d", start)
					return nil, nil
				}
			},
		}),
		NewWorkerClient(worker),
		1,
	)

	result := runner.SummarizeFile(context.Background(), "chunked.go", "Trace flow", 2, "trace-rs-4")
	if result.Error != "" {
		t.Fatalf("unexpected summarize error: %s", result.Error)
	}
	if result.Summary != "Merged file summary" {
		t.Fatalf("unexpected final summary: got %q want %q", result.Summary, "Merged file summary")
	}
	if !result.Truncated {
		t.Fatal("expected truncated summary metadata for partial coverage")
	}
	if result.ChunksRead != 2 || result.LinesCovered != 400 {
		t.Fatalf("unexpected coverage metadata: %+v", result)
	}
	if len(worker.requests) != 3 {
		t.Fatalf("unexpected worker request count: got %d want %d", len(worker.requests), 3)
	}
	mergedPrompt := worker.requests[2].Messages[1].Text
	for _, snippet := range []string{"Chunk 1:", "Chunk one summary", "Chunk 2:", "Chunk two summary"} {
		if !strings.Contains(mergedPrompt, snippet) {
			t.Fatalf("merged prompt missing %q:\n%s", snippet, mergedPrompt)
		}
	}
}
