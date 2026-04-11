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

	update := serveRequest(handler, http.MethodPost, "/api/config", `{"provider":"custom","api_key":"new-key","base_url":"http://localhost:1234","model":"local-model","chat_path":"/v1/messages"}`, nil)
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
	if payload["chat_path"] != "/v1/messages" {
		t.Fatalf("unexpected chat_path: got %v want %q", payload["chat_path"], "/v1/messages")
	}
	if payload["api_key_set"] != true {
		t.Fatalf("unexpected api_key_set: got %v want true", payload["api_key_set"])
	}
	if payload["model_selection_enabled"] != true {
		t.Fatalf("unexpected model_selection_enabled: got %v want true", payload["model_selection_enabled"])
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
	if payload["model_selection_enabled"] != true {
		t.Fatalf("unexpected model_selection_enabled: got %v want true", payload["model_selection_enabled"])
	}
}

func TestConfigGetReturnsEmptyGraphQLArraysWhenUnset(t *testing.T) {
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

	sources, ok := payload["graphql_sources"].([]any)
	if !ok || len(sources) != 0 {
		t.Fatalf("expected graphql_sources to be an empty array, got %#v", payload["graphql_sources"])
	}
	policies, ok := payload["graphql_mutation_policies"].([]any)
	if !ok || len(policies) != 0 {
		t.Fatalf(
			"expected graphql_mutation_policies to be an empty array, got %#v",
			payload["graphql_mutation_policies"],
		)
	}
}
