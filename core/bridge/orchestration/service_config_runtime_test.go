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

	if err := service.configStore.Update(configUpdateRequest{
		GraphqlSources: []graphqlSourceInput{{
			Name:       "crm",
			Endpoint:   "https://crm.example/graphql",
			SchemaPath: "/schemas/crm.json",
			Headers: map[string]string{
				"X-Tenant": "tenant-1",
			},
		}},
	}); err != nil {
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

	_, status, err := service.dispatchAction(
		context.Background(),
		busActionConfigUpdate,
		raw,
		"trace-graphql-header-type",
	)
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
	if len(snapshot.GraphqlSources) != 1 {
		t.Fatalf("unexpected graphql sources: %+v", snapshot.GraphqlSources)
	}
	if snapshot.GraphqlSources[0].Headers["X-Tenant"] != "tenant-1" {
		t.Fatalf("graphql headers changed after invalid update: %+v", snapshot.GraphqlSources[0].Headers)
	}
}
