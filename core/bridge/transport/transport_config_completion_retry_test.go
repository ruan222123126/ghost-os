package transport

import (
	"net/http"
	"testing"
)

func TestConfigUpdateAllowsZeroLLMCompletionRetrySettings(t *testing.T) {
	handler := newTestHandler(t, nil)

	update := serveRequest(
		handler,
		http.MethodPost,
		"/api/config",
		`{"llm_completion_retry_count":0,"llm_completion_retry_interval_ms":0}`,
		nil,
	)
	if update.Code != http.StatusOK {
		t.Fatalf("unexpected update status: got %d want %d", update.Code, http.StatusOK)
	}

	get := serveRequest(handler, http.MethodGet, "/api/config", "", nil)
	if get.Code != http.StatusOK {
		t.Fatalf("unexpected get status: got %d want %d", get.Code, http.StatusOK)
	}
	body := decodeResponseBody(t, get)
	payload, ok := body.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", body.Payload)
	}
	if payload["llm_completion_retry_count"] != float64(0) {
		t.Fatalf(
			"unexpected llm_completion_retry_count: got %v want %d",
			payload["llm_completion_retry_count"],
			0,
		)
	}
	if payload["llm_completion_retry_interval_ms"] != float64(0) {
		t.Fatalf(
			"unexpected llm_completion_retry_interval_ms: got %v want %d",
			payload["llm_completion_retry_interval_ms"],
			0,
		)
	}
}
