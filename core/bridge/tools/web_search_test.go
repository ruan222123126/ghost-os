package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"ghost-os/bridge/tools/internal/websearch"
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

	var results []websearch.Result
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
	tool := NewWebSearchTool(WebSearchConfig{})
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"query":" "}`), "trace-web-2")
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestWebSearchToolFallsBackToBingRSS(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/duck":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<html><body><p>temporary upstream layout drift</p></body></html>`))
		case "/bing":
			w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0"><channel>
  <item><title>Ghost OS Search</title><link>https://example.com/search</link><description>Fallback result</description></item>
</channel></rss>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tool := &WebSearchTool{
		httpClient: server.Client(),
		userAgents: []string{"test-agent"},
		providers: []webSearchProvider{
			{name: "duckduckgo_html", endpoint: server.URL + "/duck", format: webSearchFormatDuckDuckGoHTML},
			{name: "bing_rss", endpoint: server.URL + "/bing", format: webSearchFormatBingRSS},
		},
	}

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"ghost os","max_results":2}`), "trace-web-3")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	var results []websearch.Result
	if err := json.Unmarshal([]byte(output), &results); err != nil {
		t.Fatalf("decode results: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("unexpected result count: got %d want %d", len(results), 1)
	}
	if results[0].Title != "Ghost OS Search" || results[0].URL != "https://example.com/search" {
		t.Fatalf("unexpected fallback result: %+v", results[0])
	}
}

func TestWebSearchToolUsesTavilyWhenConfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: got %s want %s", r.Method, http.MethodPost)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-tavily-key" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("unexpected content type: %q", got)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		defer r.Body.Close()

		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if payload["query"] != "ghost os" {
			t.Fatalf("unexpected query payload: %#v", payload["query"])
		}
		if payload["max_results"] != float64(2) {
			t.Fatalf("unexpected max_results payload: %#v", payload["max_results"])
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(`{
  "results": [
    {"title":"Ghost Tavily Result","url":"https://example.com/tavily","content":"Tavily content"}
  ]
}`))
	}))
	defer server.Close()

	tool := &WebSearchTool{
		httpClient: server.Client(),
		userAgents: []string{"test-agent"},
		providers: []webSearchProvider{{
			name:     "tavily",
			endpoint: server.URL,
			format:   webSearchFormatTavilyJSON,
			apiKey:   "test-tavily-key",
		}},
	}

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"ghost os","max_results":2}`), "trace-web-4")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	var results []websearch.Result
	if err := json.Unmarshal([]byte(output), &results); err != nil {
		t.Fatalf("decode results: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("unexpected result count: got %d want %d", len(results), 1)
	}
	if results[0].Title != "Ghost Tavily Result" || results[0].URL != "https://example.com/tavily" {
		t.Fatalf("unexpected tavily result: %+v", results[0])
	}
}
