package transport

import (
	"net/http"
	"testing"
)

func TestProviderCRUDRoutes(t *testing.T) {
	handler := newTestHandler(t, nil)

	create := serveRequest(handler, http.MethodPost, "/api/config/providers", `{"name":"crs","type":"custom","base_url":"https://lldai.online/openai","api_key":"sk-xxx","models":["gpt-5.4"]}`, nil)
	if create.Code != http.StatusOK {
		t.Fatalf("unexpected create status: got %d want %d", create.Code, http.StatusOK)
	}

	list := serveRequest(handler, http.MethodGet, "/api/config/providers", "", nil)
	if list.Code != http.StatusOK {
		t.Fatalf("unexpected list status: got %d want %d", list.Code, http.StatusOK)
	}
	listBody := decodeResponseBody(t, list)
	listPayload, ok := listBody.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected list payload type: %T", listBody.Payload)
	}
	if listPayload["active_provider"] != "crs" {
		t.Fatalf("unexpected active provider: got %v want %q", listPayload["active_provider"], "crs")
	}
	providers, ok := listPayload["providers"].([]any)
	if !ok || len(providers) != 1 {
		t.Fatalf("unexpected providers payload: %#v", listPayload["providers"])
	}

	second := serveRequest(handler, http.MethodPost, "/api/config/providers", `{"name":"openai","type":"openai","api_key":"sk-yyy","models":["gpt-5.4"]}`, nil)
	if second.Code != http.StatusOK {
		t.Fatalf("unexpected second create status: got %d want %d", second.Code, http.StatusOK)
	}

	activate := serveRequest(handler, http.MethodPut, "/api/config/active-provider", `{"name":"openai"}`, nil)
	if activate.Code != http.StatusOK {
		t.Fatalf("unexpected activate status: got %d want %d", activate.Code, http.StatusOK)
	}

	update := serveRequest(handler, http.MethodPut, "/api/config/providers/openai", `{"name":"openai","type":"openai","models":["gpt-5.4"]}`, nil)
	if update.Code != http.StatusOK {
		t.Fatalf("unexpected update status: got %d want %d", update.Code, http.StatusOK)
	}

	config := serveRequest(handler, http.MethodGet, "/api/config", "", nil)
	if config.Code != http.StatusOK {
		t.Fatalf("unexpected config status: got %d want %d", config.Code, http.StatusOK)
	}
	configBody := decodeResponseBody(t, config)
	configPayload, ok := configBody.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected config payload type: %T", configBody.Payload)
	}
	if configPayload["provider"] != "openai" {
		t.Fatalf("unexpected config provider: got %v want %q", configPayload["provider"], "openai")
	}
	if configPayload["provider_type"] != "openai" {
		t.Fatalf("unexpected config provider_type: got %v want %q", configPayload["provider_type"], "openai")
	}
	if configPayload["base_url"] != "https://api.openai.com/v1" {
		t.Fatalf("unexpected config base_url: got %v", configPayload["base_url"])
	}
	if configPayload["model"] != "gpt-5.4" {
		t.Fatalf("unexpected config model: got %v want %q", configPayload["model"], "gpt-5.4")
	}

	deleteResp := serveRequest(handler, http.MethodDelete, "/api/config/providers/crs", "", nil)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("unexpected delete status: got %d want %d", deleteResp.Code, http.StatusOK)
	}

	finalList := serveRequest(handler, http.MethodGet, "/api/config/providers", "", nil)
	if finalList.Code != http.StatusOK {
		t.Fatalf("unexpected final list status: got %d want %d", finalList.Code, http.StatusOK)
	}
	finalBody := decodeResponseBody(t, finalList)
	finalPayload, ok := finalBody.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected final payload type: %T", finalBody.Payload)
	}
	remainingProviders, ok := finalPayload["providers"].([]any)
	if !ok || len(remainingProviders) != 1 {
		t.Fatalf("unexpected remaining providers: %#v", finalPayload["providers"])
	}
	if finalPayload["active_provider"] != "openai" {
		t.Fatalf("unexpected final active provider: got %v want %q", finalPayload["active_provider"], "openai")
	}
}

func TestProviderExportRoute(t *testing.T) {
	handler := newTestHandler(t, nil)

	create := serveRequest(
		handler,
		http.MethodPost,
		"/api/config/providers",
		`{"name":"crs","type":"custom","base_url":"https://lldai.online/openai","api_key":"sk-xxx","models":["gpt-5.4"]}`,
		nil,
	)
	if create.Code != http.StatusOK {
		t.Fatalf("unexpected create status: got %d want %d", create.Code, http.StatusOK)
	}

	exportResp := serveRequest(handler, http.MethodPost, "/api/config/providers/export", `{"name":"crs"}`, nil)
	if exportResp.Code != http.StatusOK {
		t.Fatalf("unexpected export status: got %d want %d body=%s", exportResp.Code, http.StatusOK, exportResp.Body.String())
	}

	exportBody := decodeResponseBody(t, exportResp)
	exportPayload, ok := exportBody.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected export payload type: %T", exportBody.Payload)
	}
	if exportPayload["name"] != "crs" || exportPayload["api_key"] != "sk-xxx" {
		t.Fatalf("unexpected export payload: %#v", exportPayload)
	}
}
