package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"ghost-os/bridge/tools/internal/websearch"
)

func TestWebSearchToolParsesDuckDuckGoHTML(t *testing.T) {
	client := &http.Client{
		Transport: webSearchRoundTripper(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodGet {
				t.Fatalf("unexpected method: got %s want %s", req.Method, http.MethodGet)
			}
			if req.URL.Path != "/html" {
				t.Fatalf("unexpected path: %q", req.URL.Path)
			}
			if got := req.URL.Query().Get("q"); got != "ghost os" {
				t.Fatalf("unexpected query: %q", got)
			}
			return newWebSearchResponse(http.StatusOK, "text/html; charset=utf-8", `
<html><body>
  <a class="result__a" href="https://example.com/a">First Result</a>
  <div class="result__snippet">First snippet text.</div>
  <a class="result__a" href="https://example.com/b">Second Result</a>
  <div class="result__snippet">Second snippet text.</div>
</body></html>`), nil
		}),
	}

	tool := &WebSearchTool{
		httpClient: client,
		endpoint:   "https://search.test/html",
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
	client := &http.Client{
		Transport: webSearchRoundTripper(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodGet {
				t.Fatalf("unexpected method: got %s want %s", req.Method, http.MethodGet)
			}
			if got := req.URL.Query().Get("q"); got != "ghost os" {
				t.Fatalf("unexpected query: %q", got)
			}

			switch req.URL.Path {
			case "/duck":
				return newWebSearchResponse(http.StatusOK, "text/html; charset=utf-8", `<html><body><p>temporary upstream layout drift</p></body></html>`), nil
			case "/bing":
				if got := req.URL.Query().Get("format"); got != "rss" {
					t.Fatalf("unexpected format: %q", got)
				}
				return newWebSearchResponse(http.StatusOK, "application/rss+xml; charset=utf-8", `<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0"><channel>
  <item><title>Ghost OS Search</title><link>https://example.com/search</link><description>Fallback result</description></item>
</channel></rss>`), nil
			default:
				t.Fatalf("unexpected path: %q", req.URL.Path)
				return nil, nil
			}
		}),
	}

	tool := &WebSearchTool{
		httpClient: client,
		userAgents: []string{"test-agent"},
		providers: []webSearchProvider{
			{name: "duckduckgo_html", endpoint: "https://search.test/duck", format: webSearchFormatDuckDuckGoHTML},
			{name: "bing_rss", endpoint: "https://search.test/bing", format: webSearchFormatBingRSS},
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
	client := &http.Client{
		Transport: webSearchRoundTripper(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Fatalf("unexpected method: got %s want %s", req.Method, http.MethodPost)
			}
			if req.URL.Path != "/tavily" {
				t.Fatalf("unexpected path: %q", req.URL.Path)
			}
			if got := req.Header.Get("Authorization"); got != "Bearer test-tavily-key" {
				t.Fatalf("unexpected authorization header: %q", got)
			}
			if got := req.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("unexpected content type: %q", got)
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			defer req.Body.Close()

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

			return newWebSearchResponse(http.StatusOK, "application/json; charset=utf-8", `{
  "results": [
    {"title":"Ghost Tavily Result","url":"https://example.com/tavily","content":"Tavily content"}
  ]
}`), nil
		}),
	}

	tool := &WebSearchTool{
		httpClient: client,
		userAgents: []string{"test-agent"},
		providers: []webSearchProvider{{
			name:     "tavily",
			endpoint: "https://search.test/tavily",
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

func TestWebSearchToolUsesExaWhenConfigured(t *testing.T) {
	client := &http.Client{
		Transport: webSearchRoundTripper(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Fatalf("unexpected method: got %s want %s", req.Method, http.MethodPost)
			}
			if req.URL.Path != "/exa" {
				t.Fatalf("unexpected path: %q", req.URL.Path)
			}
			if got := req.Header.Get("x-api-key"); got != "test-exa-key" {
				t.Fatalf("unexpected x-api-key header: %q", got)
			}
			if got := req.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("unexpected content type: %q", got)
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			defer req.Body.Close()

			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if payload["query"] != "ghost os" {
				t.Fatalf("unexpected query payload: %#v", payload["query"])
			}
			if payload["numResults"] != float64(2) {
				t.Fatalf("unexpected numResults payload: %#v", payload["numResults"])
			}

			return newWebSearchResponse(http.StatusOK, "application/json; charset=utf-8", `{
  "results": [
    {
      "title": "Ghost Exa Result",
      "url": "https://example.com/exa",
      "highlights": ["Exa highlight snippet."]
    }
  ]
}`), nil
		}),
	}

	tool := &WebSearchTool{
		httpClient: client,
		userAgents: []string{"test-agent"},
		providers: []webSearchProvider{{
			name:     "exa",
			endpoint: "https://search.test/exa",
			format:   webSearchFormatExaJSON,
			apiKey:   "test-exa-key",
		}},
	}

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"ghost os","max_results":2}`), "trace-web-5")
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
	if results[0].Title != "Ghost Exa Result" || results[0].URL != "https://example.com/exa" {
		t.Fatalf("unexpected exa result: %+v", results[0])
	}
}

func TestWebSearchToolUsesCustomTavilyURLWhenConfigured(t *testing.T) {
	client := &http.Client{
		Transport: webSearchRoundTripper(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != "https://proxy.example/internal/tavily" {
				t.Fatalf("unexpected url: %q", req.URL.String())
			}
			return newWebSearchResponse(http.StatusOK, "application/json; charset=utf-8", `{
  "results": [
    {"title":"Ghost Tavily Result","url":"https://example.com/tavily","content":"Tavily content"}
  ]
}`), nil
		}),
	}

	tool := NewWebSearchTool(WebSearchConfig{
		TavilyURL:    "https://proxy.example/internal/tavily",
		TavilyAPIKey: "test-tavily-key",
	}).(*WebSearchTool)
	tool.httpClient = client
	tool.userAgents = []string{"test-agent"}

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"ghost os"}`), "trace-web-custom-tavily")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	var results []websearch.Result
	if err := json.Unmarshal([]byte(output), &results); err != nil {
		t.Fatalf("decode results: %v", err)
	}
	if len(results) != 1 || results[0].URL != "https://example.com/tavily" {
		t.Fatalf("unexpected tavily result: %+v", results)
	}
}

func TestWebSearchToolUsesCustomExaURLWhenConfigured(t *testing.T) {
	client := &http.Client{
		Transport: webSearchRoundTripper(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != "https://proxy.example/internal/exa" {
				t.Fatalf("unexpected url: %q", req.URL.String())
			}
			return newWebSearchResponse(http.StatusOK, "application/json; charset=utf-8", `{
  "results": [
    {"title":"Ghost Exa Result","url":"https://example.com/exa","highlights":["Exa highlight snippet."]}
  ]
}`), nil
		}),
	}

	tool := NewWebSearchTool(WebSearchConfig{
		ExaURL:    "https://proxy.example/internal/exa",
		ExaAPIKey: "test-exa-key",
	}).(*WebSearchTool)
	tool.httpClient = client
	tool.userAgents = []string{"test-agent"}

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"ghost os"}`), "trace-web-custom-exa")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	var results []websearch.Result
	if err := json.Unmarshal([]byte(output), &results); err != nil {
		t.Fatalf("decode results: %v", err)
	}
	if len(results) != 1 || results[0].URL != "https://example.com/exa" {
		t.Fatalf("unexpected exa result: %+v", results)
	}
}

func TestWebSearchToolRequiresProviderWhenBothAPISearchKeysExist(t *testing.T) {
	tool := NewWebSearchTool(WebSearchConfig{
		TavilyAPIKey: "test-tavily-key",
		ExaAPIKey:    "test-exa-key",
	})

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"ghost os"}`), "trace-web-6")
	if err == nil {
		t.Fatal("expected error when provider is omitted and both api providers are configured")
	}
	if !strings.Contains(err.Error(), "provider is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWebSearchToolSelectsRequestedProvider(t *testing.T) {
	client := &http.Client{
		Transport: webSearchRoundTripper(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/exa" {
				t.Fatalf("unexpected path: %q", req.URL.Path)
			}
			return newWebSearchResponse(http.StatusOK, "application/json; charset=utf-8", `{
  "results": [
    {"title":"Ghost Exa Result","url":"https://example.com/exa","highlights":["Exa highlight snippet."]}
  ]
}`), nil
		}),
	}

	tool := &WebSearchTool{
		httpClient: client,
		userAgents: []string{"test-agent"},
		config: WebSearchConfig{
			TavilyAPIKey: "test-tavily-key",
			ExaAPIKey:    "test-exa-key",
		},
		providers: []webSearchProvider{
			{name: webSearchProviderTavily, endpoint: "https://search.test/tavily", format: webSearchFormatTavilyJSON, apiKey: "test-tavily-key"},
			{name: webSearchProviderExa, endpoint: "https://search.test/exa", format: webSearchFormatExaJSON, apiKey: "test-exa-key"},
		},
	}

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"provider":"exa","query":"ghost os"}`), "trace-web-7")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	var results []websearch.Result
	if err := json.Unmarshal([]byte(output), &results); err != nil {
		t.Fatalf("decode results: %v", err)
	}
	if len(results) != 1 || results[0].URL != "https://example.com/exa" {
		t.Fatalf("unexpected selected provider result: %+v", results)
	}
}

type webSearchRoundTripper func(*http.Request) (*http.Response, error)

func (rt webSearchRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return rt(req)
}

func newWebSearchResponse(status int, contentType, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{contentType}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
