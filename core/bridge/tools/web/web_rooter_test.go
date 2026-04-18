package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newWebRooterVersionedTool(
	t *testing.T,
	actionPath string,
	actionHandler http.HandlerFunc,
) (Tool, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/" {
			_, _ = w.Write([]byte(`{"name":"Web-Rooter API","version":"0.2.4","status":"running"}`))
			return
		}
		if r.URL.Path != actionPath {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		actionHandler(w, r)
	}))

	tool := NewWebRooterTool(WebRooterConfig{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})
	return tool, server
}

func TestWebRooterExecute_WrapsPinnedHTTPResponse(t *testing.T) {
	t.Helper()

	requests := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/":
			_, _ = w.Write([]byte(`{"name":"Web-Rooter API","version":"0.2.4","status":"running"}`))
		case "/search/internet":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("ReadAll: %v", err)
			}
			if strings.TrimSpace(string(body)) != `{"auto_crawl":false,"num_results":5,"query":"OpenAI API docs"}` {
				t.Fatalf("unexpected request body: %s", body)
			}
			if got := r.Header.Get("X-Trace-ID"); got != "trace-web-rooter-1" {
				t.Fatalf("unexpected trace header: %q", got)
			}
			_, _ = w.Write([]byte(`{
				"success": true,
				"content": "ok",
				"data": {
					"results": [{"title":"OpenAI","url":"https://openai.com/docs"}],
					"citations": [{"id":"W1","url":"https://openai.com/docs"}],
					"references_text": "[W1] https://openai.com/docs",
					"comparison": {"summary":"kept"}
				},
				"urls": ["https://openai.com/docs"],
				"error": null,
				"metadata": {"source":"web-rooter"}
			}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	tool := NewWebRooterTool(WebRooterConfig{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})
	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"internet_search","params":{"query":"OpenAI API docs","num_results":5,"auto_crawl":false}}`),
		"trace-web-rooter-1",
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Join(requests, ",") != "GET /,POST /search/internet" {
		t.Fatalf("unexpected request sequence: %v", requests)
	}

	var envelope map[string]any
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got := envelope["provider"]; got != webRooterToolName {
		t.Fatalf("unexpected provider: %+v", envelope)
	}
	if got := envelope["action"]; got != webRooterActionInternetSearch {
		t.Fatalf("unexpected action: %+v", envelope)
	}
	if got := envelope["trace_id"]; got != "trace-web-rooter-1" {
		t.Fatalf("unexpected trace_id: %+v", envelope)
	}
	citations, ok := envelope["citations"].([]any)
	if !ok || len(citations) != 1 {
		t.Fatalf("expected wrapped citations, got %+v", envelope["citations"])
	}
	if got := envelope["references_text"]; got != "[W1] https://openai.com/docs" {
		t.Fatalf("unexpected references_text: %+v", envelope)
	}
	payload, ok := envelope["payload"].(map[string]any)
	if !ok {
		t.Fatalf("expected payload object, got %+v", envelope["payload"])
	}
	data, ok := payload["data"].(map[string]any)
	if !ok || data["comparison"] == nil {
		t.Fatalf("expected raw payload to preserve comparison, got %+v", payload)
	}
}

func TestWebRooterExecute_SendsBearerTokenWhenConfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/" {
			_, _ = w.Write([]byte(`{"name":"Web-Rooter API","version":"0.2.4","status":"running"}`))
			return
		}
		if r.URL.Path != "/fetch" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer rooter-token" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		_, _ = w.Write([]byte(`{"success":true,"content":"ok","data":{},"urls":[],"error":null,"metadata":{}}`))
	}))
	defer server.Close()

	tool := NewWebRooterTool(WebRooterConfig{
		BaseURL:    server.URL,
		APIToken:   "rooter-token",
		HTTPClient: server.Client(),
	})

	if _, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"fetch","params":{"url":"https://example.com","use_browser":false}}`),
		"trace-web-rooter-auth",
	); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestWebRooterExecute_RejectsVersionDrift(t *testing.T) {
	t.Helper()

	postCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/" {
			_, _ = w.Write([]byte(`{"version":"0.2.5"}`))
			return
		}
		postCalled = true
		http.Error(w, "should not execute action", http.StatusInternalServerError)
	}))
	defer server.Close()

	tool := NewWebRooterTool(WebRooterConfig{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})
	_, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"fetch","params":{"url":"https://example.com","use_browser":false}}`),
		"trace-web-rooter-version",
	)
	if err == nil || !strings.Contains(err.Error(), "pinned to v0.2.4") {
		t.Fatalf("expected version drift error, got %v", err)
	}
	if postCalled {
		t.Fatal("expected action request to stay blocked after version mismatch")
	}
}

func TestWebRooterExecute_RequiresExplicitActionParams(t *testing.T) {
	tool := NewWebRooterTool(WebRooterConfig{BaseURL: "http://127.0.0.1:8765"})

	_, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"fetch","params":{"url":"https://example.com"}}`),
		"trace-web-rooter-params",
	)
	if err == nil || !strings.Contains(err.Error(), "fetch.use_browser is required") {
		t.Fatalf("expected explicit fetch.use_browser error, got %v", err)
	}
}

func TestWebRooterExecute_RejectsUpstreamSuccessFalse(t *testing.T) {
	tool, server := newWebRooterVersionedTool(t, "/fetch", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":false,"content":"访问失败：https://example.com","data":null,"urls":[],"error":"connection reset","metadata":{}}`))
	})
	defer server.Close()

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"fetch","params":{"url":"https://example.com","use_browser":false}}`), "trace-web-rooter-fail")
	if err == nil || !strings.Contains(err.Error(), `upstream returned success=false`) {
		t.Fatalf("expected upstream failure error, got %v", err)
	}
}

func TestWebRooterExecute_RejectsHTTP500(t *testing.T) {
	tool, server := newWebRooterVersionedTool(t, "/research", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream boom", http.StatusInternalServerError)
	})
	defer server.Close()

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"research","params":{"topic":"tool contract","max_pages":2}}`), "trace-web-rooter-500")
	if err == nil || !strings.Contains(err.Error(), "status 500: upstream boom") {
		t.Fatalf("expected explicit 500 error, got %v", err)
	}
}

func TestWebRooterExecute_RejectsMissingResponseField(t *testing.T) {
	tool, server := newWebRooterVersionedTool(t, "/research", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"content":"ok","urls":[],"error":null,"metadata":{}}`))
	})
	defer server.Close()

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"research","params":{"topic":"tool contract","max_pages":2}}`), "trace-web-rooter-missing")
	if err == nil || !strings.Contains(err.Error(), `response field "data" is required`) {
		t.Fatalf("expected missing field error, got %v", err)
	}
}

func TestWebRooterExecute_RejectsMalformedJSONResponse(t *testing.T) {
	tool, server := newWebRooterVersionedTool(t, "/research", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":true`))
	})
	defer server.Close()

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"research","params":{"topic":"tool contract","max_pages":2}}`), "trace-web-rooter-json")
	if err == nil || !strings.Contains(err.Error(), "decode response") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestWebRooterExecute_RejectsMalformedCitationsField(t *testing.T) {
	tool, server := newWebRooterVersionedTool(t, "/research", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"content":"ok","data":{"citations":"bad"},"urls":[],"error":null,"metadata":{}}`))
	})
	defer server.Close()

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"research","params":{"topic":"tool contract","max_pages":2}}`), "trace-web-rooter-citations")
	if err == nil || !strings.Contains(err.Error(), `response field "citations" must be an array`) {
		t.Fatalf("expected citations type error, got %v", err)
	}
}

func TestWebRooterExecute_RejectsRequestTimeout(t *testing.T) {
	tool, server := newWebRooterVersionedTool(t, "/research", func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte(`{"success":true,"content":"ok","data":{"citations":[],"references_text":""},"urls":[],"error":null,"metadata":{}}`))
	})
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := tool.Execute(ctx, json.RawMessage(`{"action":"research","params":{"topic":"tool contract","max_pages":2}}`), "trace-web-rooter-timeout")
	if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}
