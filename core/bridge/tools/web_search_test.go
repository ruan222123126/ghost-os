package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebSearchToolParsesDuckDuckGoHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`
<html><body>
  <a class="result__a" href="https://example.com/a">First Result</a>
  <div class="result__snippet">First snippet text.</div>
  <a class="result__a" href="https://example.com/b">Second Result</a>
  <div class="result__snippet">Second snippet text.</div>
</body></html>`))
	}))
	defer server.Close()

	tool := &WebSearchTool{
		httpClient: server.Client(),
		endpoint:   server.URL,
		userAgents: []string{"test-agent"},
	}

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"ghost os","max_results":2}`), "trace-web-1")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	var results []webSearchResult
	if err := json.Unmarshal([]byte(output), &results); err != nil {
		t.Fatalf("decode results: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("unexpected result count: got %d want %d", len(results), 2)
	}
	if results[0].Title != "First Result" || results[0].URL != "https://example.com/a" {
		t.Fatalf("unexpected first result: %+v", results[0])
	}
}

func TestWebSearchToolRequiresQuery(t *testing.T) {
	tool := NewWebSearchTool()
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"query":" "}`), "trace-web-2")
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}
