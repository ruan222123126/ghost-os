package orchestration

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestConfigUpdateRejectsNonStringGraphQLHeaders(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	if err := service.configStore.Update(configUpdateRequestToStoreRequest(configUpdateRequest{
		GraphqlSources: []graphqlSourceInput{{
			Name:       "crm",
			Endpoint:   "https://crm.example/graphql",
			SchemaPath: "/schemas/crm.json",
			Headers: map[string]string{
				"X-Tenant": "tenant-1",
			},
		}},
	})); err != nil {
		t.Fatalf("seed graphql source: %v", err)
	}

	raw := json.RawMessage(`{
		"graphql_source_upsert": {
			"name": "crm",
			"endpoint": "https://crm.example/graphql",
			"schema_path": "/schemas/crm.json",
			"headers": {
				"X-Tenant": 123
			}
		}
	}`)

	_, err := service.dispatchAction(
		context.Background(),
		busActionConfigUpdate,
		raw,
		"trace-graphql-header-type",
	)
	status := legacyStatusFromServiceError(err)
	if status != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", status)
	}
	if err == nil {
		t.Fatal("expected config update to fail")
	}
	if !strings.Contains(err.Error(), "invalid params") || !strings.Contains(err.Error(), "headers") {
		t.Fatalf("unexpected error: %v", err)
	}

	snapshot := service.configStore.Snapshot()
	if len(snapshot.GraphQLSources) != 1 {
		t.Fatalf("unexpected graphql sources: %+v", snapshot.GraphQLSources)
	}
	if snapshot.GraphQLSources[0].Headers["X-Tenant"] != "tenant-1" {
		t.Fatalf("graphql headers changed after invalid update: %+v", snapshot.GraphQLSources[0].Headers)
	}
}

func TestConfigUpdatePropagatesWebRooterFieldsThroughOrchestration(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	raw := json.RawMessage(`{
		"session_human_log_full_enabled": true,
		"web_rooter_enabled": true,
		"web_rooter_base_url": "http://127.0.0.1:9988",
		"web_rooter_api_token": "rooter-token",
		"web_rooter_timeout_ms": 12345
	}`)

	result, err := service.dispatchAction(
		context.Background(),
		busActionConfigUpdate,
		raw,
		"trace-web-rooter-update",
	)
	if err != nil {
		t.Fatalf("config update failed: %v", err)
	}
	status := legacyStatusFromServiceOutcome(result.Outcome)
	if status != http.StatusOK {
		t.Fatalf("unexpected status: %d", status)
	}

	snapshot, ok := result.Payload.(configResponse)
	if !ok {
		t.Fatalf("unexpected payload type: %T", result.Payload)
	}
	if !snapshot.WebRooterEnabled {
		t.Fatal("expected web_rooter_enabled in snapshot")
	}
	if snapshot.WebRooterBaseURL != "http://127.0.0.1:9988" {
		t.Fatalf("unexpected web_rooter_base_url in snapshot: %q", snapshot.WebRooterBaseURL)
	}
	if snapshot.WebRooterTimeoutMs != 12345 {
		t.Fatalf("unexpected web_rooter_timeout_ms in snapshot: %d", snapshot.WebRooterTimeoutMs)
	}
	if !snapshot.SessionHumanLogFullEnabled {
		t.Fatal("expected session_human_log_full_enabled in snapshot")
	}
	if !snapshot.WebRooterAPITokenSet {
		t.Fatal("expected web_rooter_api_token_set in snapshot")
	}

	cfg, err := service.configStore.Config()
	if err != nil {
		t.Fatalf("load runtime config: %v", err)
	}
	if !cfg.WebRooterEnabled {
		t.Fatal("expected web_rooter_enabled in runtime config")
	}
	if !cfg.SessionHumanLogFullEnabled {
		t.Fatal("expected session_human_log_full_enabled in runtime config")
	}
	if cfg.WebRooterBaseURL != "http://127.0.0.1:9988" {
		t.Fatalf("unexpected web_rooter_base_url: %q", cfg.WebRooterBaseURL)
	}
	if cfg.WebRooterAPIToken != "rooter-token" {
		t.Fatalf("unexpected web_rooter_api_token: %q", cfg.WebRooterAPIToken)
	}
	if cfg.WebRooterTimeoutMS != 12345 {
		t.Fatalf("unexpected web_rooter_timeout_ms: %d", cfg.WebRooterTimeoutMS)
	}
}
