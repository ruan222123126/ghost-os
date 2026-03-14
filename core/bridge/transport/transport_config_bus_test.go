package transport

import (
	"net/http"
	"testing"
)

func TestBusConfigGetAndUpdateRegression(t *testing.T) {
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_BASE_URL", "http://127.0.0.1:9999")
	t.Setenv("GHOST_MODEL", "test-model")
	t.Setenv("GHOST_CHAT_PATH", "/v1/chat")
	t.Setenv("GHOST_API_KEY", "secret")

	handler := newTestHandler(t, nil)

	getResp := serveRequest(handler, http.MethodPost, "/api/bus", `{"action":"CONFIG_GET","params":{},"trace_id":"trace-config-get"}`, nil)
	if getResp.Code != http.StatusOK {
		t.Fatalf("unexpected status for config get: got %d want %d", getResp.Code, http.StatusOK)
	}
	getBody := decodeResponseBody(t, getResp)
	payload, ok := getBody.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", getBody.Payload)
	}
	if payload["provider"] != "custom" {
		t.Fatalf("unexpected provider: got %v want %q", payload["provider"], "custom")
	}
	if payload["provider_type"] != "custom" {
		t.Fatalf("unexpected provider_type: got %v want %q", payload["provider_type"], "custom")
	}
	if payload["api_key_set"] != true {
		t.Fatalf("unexpected api_key_set: got %v want true", payload["api_key_set"])
	}
	if payload["model_selection_enabled"] != true {
		t.Fatalf("unexpected model_selection_enabled: got %v want true", payload["model_selection_enabled"])
	}

	updateResp := serveRequest(handler, http.MethodPost, "/api/bus", `{"action":"CONFIG_UPDATE","params":{"provider":"custom","model":"local"},"trace_id":"trace-config-update"}`, nil)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("unexpected status for config update: got %d want %d", updateResp.Code, http.StatusOK)
	}
	updateBody := decodeResponseBody(t, updateResp)
	updatePayload, ok := updateBody.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected update payload type: %T", updateBody.Payload)
	}
	if updatePayload["model"] != "local" {
		t.Fatalf("unexpected model: got %v want %q", updatePayload["model"], "local")
	}
	if updatePayload["model_selection_enabled"] != true {
		t.Fatalf("unexpected model_selection_enabled: got %v want true", updatePayload["model_selection_enabled"])
	}
}
