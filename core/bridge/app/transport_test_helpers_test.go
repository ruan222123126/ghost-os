package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/session"
)

func decodeResponseBody(t *testing.T, recorder *httptest.ResponseRecorder) apiResponse {
	t.Helper()

	var body apiResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v, body=%s", err, recorder.Body.String())
	}
	return body
}

func newTestHandler(t *testing.T, executor agentExecutorFunc) http.Handler {
	t.Helper()
	handler, _ := newTestHandlerWithStore(t, executor)
	return handler
}

func newTestHandlerWithStore(t *testing.T, executor agentExecutorFunc) (http.Handler, *session.Store) {
	handler, _, sessionStore := newTestHandlerWithService(t, executor, nil)
	return handler, sessionStore
}

func newTestHandlerWithStreamExecutor(t *testing.T, executor agentExecutorFunc, streamExecutor agentStreamExecutorFunc) (http.Handler, *session.Store) {
	handler, _, sessionStore := newTestHandlerWithService(t, executor, streamExecutor)
	return handler, sessionStore
}

func newTestHandlerWithService(t *testing.T, executor agentExecutorFunc, streamExecutor agentStreamExecutorFunc) (http.Handler, *bridgeService, *session.Store) {
	t.Helper()
	tempDir := t.TempDir()
	t.Setenv("GHOST_CONFIG_PATH", tempDir+"/config.toml")
	t.Setenv("GHOST_API_KEY", "test-key")
	t.Setenv("GHOST_TASKS_PATH", tempDir+"/tasks")
	t.Setenv("GHOST_RSS_POLL_ENABLED", "false")
	t.Setenv("GHOST_RSS_INBOX_PATH", tempDir+"/rss/inbox.json")
	t.Setenv("GHOST_RSS_FEEDS_PATH", tempDir+"/rss/feeds.json")
	t.Setenv("GHOST_MEMORY_WARM_PATH", tempDir+"/memory/warm.json")
	t.Setenv("GHOST_MEMORY_COLD_PATH", tempDir+"/memory/cold")
	t.Setenv("GHOST_MEMORY_DECISION_PATH", tempDir+"/memory/decision")
	t.Setenv("GHOST_MEMORY_HYGIENE_PATH", tempDir+"/memory/hygiene")
	if executor == nil {
		executor = func(_ context.Context, _ string, _ string, _ string, _ *ConfigStore, _ *session.Store) (string, string, error) {
			return "ok", "session-test", nil
		}
	}

	store, err := NewConfigStoreFromEnv()
	if err != nil {
		t.Fatalf("new config store: %v", err)
	}

	sessionStore, err := session.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}

	service := newBridgeServiceWithStreamExecutor(store, sessionStore, executor, streamExecutor)
	t.Cleanup(service.Close)
	options := newServerOptionsFromEnv(8080)
	options.maxBodyBytes = defaultMaxRequestBodyBytes
	return newHTTPHandler(service, options), service, sessionStore
}

func decodeSSEEvents(t *testing.T, recorder *httptest.ResponseRecorder) []agent.AgentEvent {
	t.Helper()

	trimmed := strings.TrimSpace(recorder.Body.String())
	if trimmed == "" {
		return nil
	}

	blocks := strings.Split(trimmed, "\n\n")
	events := make([]agent.AgentEvent, 0, len(blocks))
	for _, block := range blocks {
		if strings.TrimSpace(block) == "" {
			continue
		}

		var eventName string
		var dataLine string
		for _, line := range strings.Split(block, "\n") {
			switch {
			case strings.HasPrefix(line, "event: "):
				eventName = strings.TrimSpace(strings.TrimPrefix(line, "event: "))
			case strings.HasPrefix(line, "data: "):
				dataLine = strings.TrimSpace(strings.TrimPrefix(line, "data: "))
			}
		}
		if dataLine == "" {
			t.Fatalf("missing data line in block: %q", block)
		}

		var event agent.AgentEvent
		if err := json.Unmarshal([]byte(dataLine), &event); err != nil {
			t.Fatalf("decode sse event: %v", err)
		}
		if eventName != "" && string(event.Type) != eventName {
			t.Fatalf("event header mismatch: header=%q payload=%q", eventName, event.Type)
		}
		events = append(events, event)
	}
	return events
}

func serveRequest(handler http.Handler, method string, path string, body string, headers map[string]string) *httptest.ResponseRecorder {
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}

	request := httptest.NewRequest(method, path, reader)
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}
