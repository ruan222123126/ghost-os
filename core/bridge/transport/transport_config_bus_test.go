package transport

import (
	"net/http"
	"net/http/httptest"
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

func TestBusProviderWriteActionsRegression(t *testing.T) {
	handler := newTestHandler(t, nil)

	createResp := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"CONFIG_PROVIDER_CREATE","params":{"name":"crs","type":"custom","base_url":"https://example.com/v1","api_key":"sk-xxx","models":["gpt-5.4"],"context_window_tokens":100000},"trace_id":"trace-provider-create"}`,
		nil,
	)
	if createResp.Code != http.StatusOK {
		t.Fatalf("unexpected create status: got %d want %d body=%s", createResp.Code, http.StatusOK, createResp.Body.String())
	}
	createPayload := providerListPayloadFromBusResponse(t, createResp)
	if createPayload["active_provider"] != "crs" {
		t.Fatalf("unexpected active provider after create: got %v want %q", createPayload["active_provider"], "crs")
	}
	providers := providerItemsFromPayload(t, createPayload)
	if len(providers) != 1 {
		t.Fatalf("unexpected provider count after create: got %d want 1", len(providers))
	}

	updateResp := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"CONFIG_PROVIDER_UPDATE","params":{"name":"crs","provider":{"name":"crs-renamed","type":"custom","base_url":"https://example.com/openai","models":["gpt-5.5"],"context_window_tokens":200000}},"trace_id":"trace-provider-update"}`,
		nil,
	)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("unexpected update status: got %d want %d body=%s", updateResp.Code, http.StatusOK, updateResp.Body.String())
	}
	updatePayload := providerListPayloadFromBusResponse(t, updateResp)
	if updatePayload["active_provider"] != "crs-renamed" {
		t.Fatalf("unexpected active provider after rename: got %v want %q", updatePayload["active_provider"], "crs-renamed")
	}
	updatedProviders := providerItemsFromPayload(t, updatePayload)
	if len(updatedProviders) != 1 {
		t.Fatalf("unexpected provider count after update: got %d want 1", len(updatedProviders))
	}
	updated, ok := updatedProviders[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected provider item type: %T", updatedProviders[0])
	}
	if updated["name"] != "crs-renamed" || updated["base_url"] != "https://example.com/openai" {
		t.Fatalf("unexpected updated provider: %#v", updated)
	}
	if updated["api_key_set"] != true {
		t.Fatalf("expected update without api_key to preserve existing key, got %#v", updated)
	}

	deleteResp := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"CONFIG_PROVIDER_DELETE","params":{"name":"crs-renamed"},"trace_id":"trace-provider-delete"}`,
		nil,
	)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("unexpected delete status: got %d want %d body=%s", deleteResp.Code, http.StatusOK, deleteResp.Body.String())
	}
	deletePayload := providerListPayloadFromBusResponse(t, deleteResp)
	if providers := providerItemsFromPayload(t, deletePayload); len(providers) != 0 {
		t.Fatalf("unexpected provider count after delete: got %d want 0", len(providers))
	}
}

func TestBusProviderExportRegression(t *testing.T) {
	handler := newTestHandler(t, nil)

	createResp := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"CONFIG_PROVIDER_CREATE","params":{"name":"crs","type":"custom","base_url":"https://example.com/v1","api_key":"sk-xxx","models":["gpt-5.4"]},"trace_id":"trace-provider-create"}`,
		nil,
	)
	if createResp.Code != http.StatusOK {
		t.Fatalf("unexpected create status: got %d want %d body=%s", createResp.Code, http.StatusOK, createResp.Body.String())
	}

	exportResp := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"CONFIG_PROVIDER_EXPORT","params":{"name":"crs"},"trace_id":"trace-provider-export"}`,
		nil,
	)
	if exportResp.Code != http.StatusOK {
		t.Fatalf("unexpected export status: got %d want %d body=%s", exportResp.Code, http.StatusOK, exportResp.Body.String())
	}

	body := decodeResponseBody(t, exportResp)
	payload, ok := body.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected provider export payload type: %T", body.Payload)
	}
	if payload["name"] != "crs" {
		t.Fatalf("unexpected export name: %#v", payload)
	}
	if payload["provider_id"] == "" {
		t.Fatalf("expected provider_id in export payload: %#v", payload)
	}
	if payload["api_key"] != "sk-xxx" {
		t.Fatalf("expected api_key in export payload: %#v", payload)
	}
}

func TestBusProviderWriteActionErrorsRegression(t *testing.T) {
	handler := newTestHandler(t, nil)

	alphaResp := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"CONFIG_PROVIDER_CREATE","params":{"name":"alpha","type":"custom","base_url":"https://alpha.example/v1"},"trace_id":"trace-provider-alpha"}`,
		nil,
	)
	if alphaResp.Code != http.StatusOK {
		t.Fatalf("unexpected alpha create status: got %d want %d body=%s", alphaResp.Code, http.StatusOK, alphaResp.Body.String())
	}

	betaResp := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"CONFIG_PROVIDER_CREATE","params":{"name":"beta","type":"custom","base_url":"https://beta.example/v1"},"trace_id":"trace-provider-beta"}`,
		nil,
	)
	if betaResp.Code != http.StatusOK {
		t.Fatalf("unexpected beta create status: got %d want %d body=%s", betaResp.Code, http.StatusOK, betaResp.Body.String())
	}

	missingResp := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"CONFIG_PROVIDER_UPDATE","params":{"name":"missing","provider":{"name":"missing","type":"custom","base_url":"https://missing.example/v1"}},"trace_id":"trace-provider-missing"}`,
		nil,
	)
	if missingResp.Code != http.StatusNotFound {
		t.Fatalf("unexpected missing update status: got %d want %d body=%s", missingResp.Code, http.StatusNotFound, missingResp.Body.String())
	}

	duplicateResp := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"CONFIG_PROVIDER_UPDATE","params":{"name":"beta","provider":{"name":"alpha","type":"custom","base_url":"https://duplicate.example/v1"}},"trace_id":"trace-provider-duplicate"}`,
		nil,
	)
	if duplicateResp.Code != http.StatusConflict {
		t.Fatalf("unexpected duplicate update status: got %d want %d body=%s", duplicateResp.Code, http.StatusConflict, duplicateResp.Body.String())
	}
}

func providerListPayloadFromBusResponse(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	body := decodeResponseBody(t, recorder)
	payload, ok := body.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected provider payload type: %T", body.Payload)
	}
	return payload
}

func providerItemsFromPayload(t *testing.T, payload map[string]any) []any {
	t.Helper()
	providers, ok := payload["providers"].([]any)
	if !ok {
		t.Fatalf("unexpected providers list: %#v", payload["providers"])
	}
	return providers
}
