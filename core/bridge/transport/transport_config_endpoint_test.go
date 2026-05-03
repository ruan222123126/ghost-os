package transport

import (
	"net/http"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
)

func TestHandleConfigMethodNotAllowed(t *testing.T) {
	handler := newTestHandler(t, nil)
	recorder := serveRequest(handler, http.MethodPut, "/api/config", "", nil)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
}

func TestConfigUpdateThenGetUsesStore(t *testing.T) {
	handler := newTestHandler(t, nil)

	update := serveRequest(handler, http.MethodPost, "/api/config", `{"provider":"custom","api_key":"new-key","base_url":"http://localhost:1234","model":"local-model","chat_path":"/v1/messages","max_turns":9,"llm_completion_retry_count":2,"llm_completion_retry_interval_ms":300,"session_system_prompt_visible_enabled":false,"assistant_markdown_enabled":false,"tool_call_compact_output_enabled":true,"memory_mode_enabled":true,"microcompact_enabled":true}`, nil)
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
	if payload["provider"] != "custom" {
		t.Fatalf("unexpected provider: got %v want %q", payload["provider"], "custom")
	}
	if payload["provider_type"] != "custom" {
		t.Fatalf("unexpected provider_type: got %v want %q", payload["provider_type"], "custom")
	}
	if payload["base_url"] != "http://localhost:1234" {
		t.Fatalf("unexpected base_url: got %v want %q", payload["base_url"], "http://localhost:1234")
	}
	if payload["model"] != "local-model" {
		t.Fatalf("unexpected model: got %v want %q", payload["model"], "local-model")
	}
	if payload["max_turns"] != float64(9) {
		t.Fatalf("unexpected max_turns: got %v want %d", payload["max_turns"], 9)
	}
	if payload["llm_completion_retry_count"] != float64(2) {
		t.Fatalf(
			"unexpected llm_completion_retry_count: got %v want %d",
			payload["llm_completion_retry_count"],
			2,
		)
	}
	if payload["llm_completion_retry_interval_ms"] != float64(300) {
		t.Fatalf(
			"unexpected llm_completion_retry_interval_ms: got %v want %d",
			payload["llm_completion_retry_interval_ms"],
			300,
		)
	}
	if payload["chat_path"] != "/v1/messages" {
		t.Fatalf("unexpected chat_path: got %v want %q", payload["chat_path"], "/v1/messages")
	}
	if payload["api_key_set"] != true {
		t.Fatalf("unexpected api_key_set: got %v want true", payload["api_key_set"])
	}
	if payload["model_selection_enabled"] != true {
		t.Fatalf("unexpected model_selection_enabled: got %v want true", payload["model_selection_enabled"])
	}
	if payload["session_system_prompt_visible_enabled"] != false {
		t.Fatalf("unexpected session_system_prompt_visible_enabled: got %v want false", payload["session_system_prompt_visible_enabled"])
	}
	if payload["assistant_markdown_enabled"] != false {
		t.Fatalf("unexpected assistant_markdown_enabled: got %v want false", payload["assistant_markdown_enabled"])
	}
	if payload["tool_call_compact_output_enabled"] != true {
		t.Fatalf(
			"unexpected tool_call_compact_output_enabled: got %v want true",
			payload["tool_call_compact_output_enabled"],
		)
	}
	if payload["memory_mode_enabled"] != true {
		t.Fatalf("unexpected memory_mode_enabled: got %v want true", payload["memory_mode_enabled"])
	}
	if payload["microcompact_enabled"] != true {
		t.Fatalf("unexpected microcompact_enabled: got %v want true", payload["microcompact_enabled"])
	}
}

func TestConfigUpdateEmptyBaseURLAndModelResetDefaults(t *testing.T) {
	handler := newTestHandler(t, nil)

	update := serveRequest(handler, http.MethodPost, "/api/config", `{"base_url":"","model":""}`, nil)
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
	if payload["base_url"] != bridgeconfig.DefaultBaseURL {
		t.Fatalf("unexpected base_url: got %v want %q", payload["base_url"], bridgeconfig.DefaultBaseURL)
	}
	if payload["model"] != bridgeconfig.DefaultModel {
		t.Fatalf("unexpected model: got %v want %q", payload["model"], bridgeconfig.DefaultModel)
	}
	if payload["chat_path"] != "" {
		t.Fatalf("unexpected chat_path: got %v want empty string", payload["chat_path"])
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
	if payload["llm_completion_retry_count"] != float64(1) {
		t.Fatalf(
			"unexpected llm_completion_retry_count: got %v want %d",
			payload["llm_completion_retry_count"],
			1,
		)
	}
	if payload["llm_completion_retry_interval_ms"] != float64(200) {
		t.Fatalf(
			"unexpected llm_completion_retry_interval_ms: got %v want %d",
			payload["llm_completion_retry_interval_ms"],
			200,
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
}

func TestConfigUpdateEmptyChatPathClearsOverride(t *testing.T) {
	handler := newTestHandler(t, nil)

	update := serveRequest(handler, http.MethodPost, "/api/config", `{"chat_path":"/v1/messages"}`, nil)
	if update.Code != http.StatusOK {
		t.Fatalf("unexpected first update status: got %d want %d", update.Code, http.StatusOK)
	}

	reset := serveRequest(handler, http.MethodPost, "/api/config", `{"chat_path":""}`, nil)
	if reset.Code != http.StatusOK {
		t.Fatalf("unexpected reset status: got %d want %d", reset.Code, http.StatusOK)
	}
	body := decodeResponseBody(t, reset)
	payload, ok := body.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", body.Payload)
	}
	if payload["chat_path"] != "" {
		t.Fatalf("unexpected reset chat_path: got %v want empty string", payload["chat_path"])
	}

	get := serveRequest(handler, http.MethodGet, "/api/config", "", nil)
	if get.Code != http.StatusOK {
		t.Fatalf("unexpected get status: got %d want %d", get.Code, http.StatusOK)
	}
	body = decodeResponseBody(t, get)
	payload, ok = body.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", body.Payload)
	}
	if payload["chat_path"] != "" {
		t.Fatalf("unexpected persisted chat_path: got %v want empty string", payload["chat_path"])
	}
}

func TestConfigGetReturnsRuntimeDefaultsWhenUnset(t *testing.T) {
	handler := newTestHandler(t, nil)

	get := serveRequest(handler, http.MethodGet, "/api/config", "", nil)
	if get.Code != http.StatusOK {
		t.Fatalf("unexpected get status: got %d want %d", get.Code, http.StatusOK)
	}
	body := decodeResponseBody(t, get)
	payload, ok := body.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", body.Payload)
	}
	if payload["session_system_prompt_visible_enabled"] != true {
		t.Fatalf("unexpected session_system_prompt_visible_enabled: got %v want true", payload["session_system_prompt_visible_enabled"])
	}
	if payload["max_turns"] != float64(20) {
		t.Fatalf("unexpected max_turns: got %v want %d", payload["max_turns"], 20)
	}
	if payload["llm_completion_retry_count"] != float64(1) {
		t.Fatalf(
			"unexpected llm_completion_retry_count: got %v want %d",
			payload["llm_completion_retry_count"],
			1,
		)
	}
	if payload["llm_completion_retry_interval_ms"] != float64(200) {
		t.Fatalf(
			"unexpected llm_completion_retry_interval_ms: got %v want %d",
			payload["llm_completion_retry_interval_ms"],
			200,
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
	if payload["chat_path"] != "" {
		t.Fatalf("unexpected chat_path: got %v want empty string", payload["chat_path"])
	}
}
