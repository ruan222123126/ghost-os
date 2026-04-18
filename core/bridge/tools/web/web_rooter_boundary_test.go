package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebRooterExecute_ForwardsTraceIDToVersionAndActionRequests(t *testing.T) {
	var versionTraceID string
	var actionTraceID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/":
			versionTraceID = r.Header.Get("X-Trace-ID")
			_, _ = w.Write([]byte(`{"version":"0.2.4"}`))
		case "/research":
			actionTraceID = r.Header.Get("X-Trace-ID")
			_, _ = w.Write([]byte(`{"success":true,"content":"ok","data":{},"urls":[],"error":null,"metadata":{}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	tool := NewWebRooterTool(WebRooterConfig{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})
	traceID := "trace-web-rooter-sidecar"
	_, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"research","params":{"topic":"trace contract","max_pages":2}}`),
		traceID,
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if versionTraceID != traceID {
		t.Fatalf("unexpected version trace header: got %q want %q", versionTraceID, traceID)
	}
	if actionTraceID != traceID {
		t.Fatalf("unexpected action trace header: got %q want %q", actionTraceID, traceID)
	}
}

func TestWebRooterExecute_RejectsStateTrackingActions(t *testing.T) {
	tool := NewWebRooterTool(WebRooterConfig{BaseURL: "http://127.0.0.1:8765"})

	_, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"knowledge","params":{}}`),
		"trace-web-rooter-stateful",
	)
	if err == nil || !strings.Contains(err.Error(), `unsupported web_rooter action "knowledge"`) {
		t.Fatalf("expected unsupported action error, got %v", err)
	}
}

func TestWebRooterExecute_RejectsOversizedJSONResponse(t *testing.T) {
	oversized := strings.Repeat("a", (16<<20)+64)

	tool, server := newWebRooterVersionedTool(t, "/fetch", func(w http.ResponseWriter, _ *http.Request) {
		payload, err := json.Marshal(map[string]any{
			"success": true,
			"content": oversized,
			"data":    map[string]any{},
			"urls":    []string{},
			"error":   nil,
			"metadata": map[string]any{
				"source": "web-rooter",
			},
		})
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		_, _ = w.Write(payload)
	})
	defer server.Close()

	_, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"fetch","params":{"url":"https://example.com","use_browser":false}}`),
		"trace-web-rooter-oversized",
	)
	if err == nil || !strings.Contains(err.Error(), "response body exceeds 16777216 bytes") {
		t.Fatalf("expected oversized response error, got %v", err)
	}
}
