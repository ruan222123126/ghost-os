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
	if payload["session_system_prompt_visible_enabled"] != true {
		t.Fatalf("unexpected session_system_prompt_visible_enabled: got %v want true", payload["session_system_prompt_visible_enabled"])
	}
	if payload["max_turns"] != float64(20) {
		t.Fatalf("unexpected max_turns: got %v want %d", payload["max_turns"], 20)
	}
	if payload["task_execution_timeout_ms"] != float64(300000) {
		t.Fatalf(
			"unexpected task_execution_timeout_ms: got %v want %d",
			payload["task_execution_timeout_ms"],
			300000,
		)
	}
	if payload["assistant_markdown_enabled"] != true {
		t.Fatalf("unexpected assistant_markdown_enabled: got %v want true", payload["assistant_markdown_enabled"])
	}
	if payload["tool_call_compact_output_enabled"] != false {
		t.Fatalf(
			"unexpected tool_call_compact_output_enabled: got %v want false",
			payload["tool_call_compact_output_enabled"],
		)
	}
	if payload["memory_mode_enabled"] != false {
		t.Fatalf("unexpected memory_mode_enabled: got %v want false", payload["memory_mode_enabled"])
	}
	if payload["microcompact_enabled"] != false {
		t.Fatalf("unexpected microcompact_enabled: got %v want false", payload["microcompact_enabled"])
	}

	providersResp := serveRequest(handler, http.MethodPost, "/api/bus", `{"action":"CONFIG_PROVIDERS_GET","params":{},"trace_id":"trace-config-providers"}`, nil)
	if providersResp.Code != http.StatusOK {
		t.Fatalf("unexpected status for config providers get: got %d want %d", providersResp.Code, http.StatusOK)
	}
	providersBody := decodeResponseBody(t, providersResp)
	providersPayload, ok := providersBody.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected providers payload type: %T", providersBody.Payload)
	}
	if providersPayload["active_provider"] != "custom" {
		t.Fatalf("unexpected active_provider: got %v want %q", providersPayload["active_provider"], "custom")
	}
	if _, ok := providersPayload["providers"].([]any); !ok {
		t.Fatalf("unexpected providers list: %#v", providersPayload["providers"])
	}

	updateResp := serveRequest(handler, http.MethodPost, "/api/bus", `{"action":"CONFIG_UPDATE","params":{"provider":"custom","model":"local","max_turns":9,"task_execution_timeout_ms":600000,"session_system_prompt_visible_enabled":false,"assistant_markdown_enabled":false,"tool_call_compact_output_enabled":true,"memory_mode_enabled":true,"microcompact_enabled":true},"trace_id":"trace-config-update"}`, nil)
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
	if updatePayload["session_system_prompt_visible_enabled"] != false {
		t.Fatalf("unexpected session_system_prompt_visible_enabled: got %v want false", updatePayload["session_system_prompt_visible_enabled"])
	}
	if updatePayload["max_turns"] != float64(9) {
		t.Fatalf("unexpected max_turns: got %v want %d", updatePayload["max_turns"], 9)
	}
	if updatePayload["task_execution_timeout_ms"] != float64(600000) {
		t.Fatalf(
			"unexpected task_execution_timeout_ms: got %v want %d",
			updatePayload["task_execution_timeout_ms"],
			600000,
		)
	}
	if updatePayload["assistant_markdown_enabled"] != false {
		t.Fatalf("unexpected assistant_markdown_enabled: got %v want false", updatePayload["assistant_markdown_enabled"])
	}
	if updatePayload["tool_call_compact_output_enabled"] != true {
		t.Fatalf(
			"unexpected tool_call_compact_output_enabled: got %v want true",
			updatePayload["tool_call_compact_output_enabled"],
		)
	}
	if updatePayload["memory_mode_enabled"] != true {
		t.Fatalf("unexpected memory_mode_enabled: got %v want true", updatePayload["memory_mode_enabled"])
	}
	if updatePayload["microcompact_enabled"] != true {
		t.Fatalf("unexpected microcompact_enabled: got %v want true", updatePayload["microcompact_enabled"])
	}
}
