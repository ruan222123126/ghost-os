package config

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeConfigFromEnvMaterializesLegacyGraphQLSource(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_GRAPHQL_ENABLED", "true")
	t.Setenv("GHOST_GRAPHQL_ENDPOINT", "https://env.example/graphql")
	t.Setenv("GHOST_GRAPHQL_API_KEY", "env-graphql-key")
	t.Setenv("GHOST_GRAPHQL_SCHEMA_PATH", "/env/schema.json")
	t.Setenv("GHOST_GRAPHQL_TIMEOUT_MS", "9000")
	t.Setenv("GHOST_GRAPHQL_MAX_RESPONSE_BYTES", "8192")

	if err := writeBridgeFileConfig(configPath, bridgeFileConfig{
		GraphQLEnabled:    boolPointer(true),
		GraphQLEndpoint:   stringPointer("https://file.example/graphql"),
		GraphQLSchemaPath: stringPointer("/file/schema.json"),
		GraphQLTimeoutMS:  optionalIntPointer(5000),
		GraphQLHeaders: map[string]string{
			"X-File": "file",
		},
	}); err != nil {
		t.Fatalf("writeBridgeFileConfig: %v", err)
	}

	runtime, err := runtimeConfigFromEnv()
	if err != nil {
		t.Fatalf("runtimeConfigFromEnv: %v", err)
	}

	if runtime.GraphQL.DefaultSource != DefaultGraphQLLegacySourceName {
		t.Fatalf("unexpected graphql default source: %q", runtime.GraphQL.DefaultSource)
	}
	source := findGraphQLSource(t, runtime.GraphQL.Sources, DefaultGraphQLLegacySourceName)
	if source.Endpoint != "https://file.example/graphql" {
		t.Fatalf("unexpected graphql endpoint: %q", source.Endpoint)
	}
	if source.APIKey != "env-graphql-key" {
		t.Fatalf("unexpected graphql api key: %q", source.APIKey)
	}
	if source.SchemaPath != "/file/schema.json" {
		t.Fatalf("unexpected graphql schema path: %q", source.SchemaPath)
	}
	if source.TimeoutMS != 5000 {
		t.Fatalf("unexpected graphql timeout: %d", source.TimeoutMS)
	}
	if source.MaxResponseBytes != 8192 {
		t.Fatalf("unexpected graphql max bytes: %d", source.MaxResponseBytes)
	}
	if source.Headers["X-File"] != "file" {
		t.Fatalf("unexpected graphql headers: %+v", source.Headers)
	}
}

func TestRuntimeConfigFromEnvUsesGraphQLSourcesLayout(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	if err := writeBridgeFileConfig(configPath, bridgeFileConfig{
		GraphQLDefaultSource: stringPointer("crm"),
		GraphQLSources: []graphQLSourceFileConfig{
			{
				Name:             "billing",
				Endpoint:         "https://billing.example/graphql",
				SchemaPath:       "/schemas/billing.json",
				TimeoutMS:        4000,
				MaxResponseBytes: 4096,
				MaxDepth:         6,
				MaxFields:        24,
				MaxRootFields:    2,
				MaxFragments:     3,
			},
			{
				Name:             "crm",
				Description:      "CRM data",
				Endpoint:         "https://crm.example/graphql",
				APIKey:           stringPointer("crm-key"),
				SchemaPath:       "/schemas/crm.json",
				TimeoutMS:        7000,
				MaxResponseBytes: 16384,
				MaxDepth:         7,
				MaxFields:        48,
				MaxRootFields:    3,
				MaxFragments:     5,
				Domains: []graphQLDomainFileConfig{{
					Name:        "orders",
					RootQueries: []string{"order", "orders"},
					Types:       []string{"Order", "OrderEdge"},
					MaxDepth:    4,
				}},
			},
		},
	}); err != nil {
		t.Fatalf("writeBridgeFileConfig: %v", err)
	}

	runtime, err := runtimeConfigFromEnv()
	if err != nil {
		t.Fatalf("runtimeConfigFromEnv: %v", err)
	}

	if runtime.GraphQL.DefaultSource != "crm" {
		t.Fatalf("unexpected graphql default source: %q", runtime.GraphQL.DefaultSource)
	}
	if len(runtime.GraphQL.Sources) != 2 {
		t.Fatalf("unexpected graphql source count: %d", len(runtime.GraphQL.Sources))
	}
	source := findGraphQLSource(t, runtime.GraphQL.Sources, "crm")
	if source.Description != "CRM data" {
		t.Fatalf("unexpected source description: %q", source.Description)
	}
	if source.MaxFragments != 5 {
		t.Fatalf("unexpected source max fragments: %d", source.MaxFragments)
	}
	if len(source.Domains) != 1 || source.Domains[0].Name != "orders" {
		t.Fatalf("unexpected source domains: %+v", source.Domains)
	}
	if len(runtime.GraphQL.MutationPolicies) != 0 {
		t.Fatalf("expected no mutation policies by default, got %+v", runtime.GraphQL.MutationPolicies)
	}
}

func TestRuntimeConfigFromEnvLoadsGraphQLMutationPolicies(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	if err := writeBridgeFileConfig(configPath, bridgeFileConfig{
		GraphQLDefaultSource: stringPointer("crm"),
		GraphQLSources: []graphQLSourceFileConfig{{
			Name:             "crm",
			Endpoint:         "https://crm.example/graphql",
			SchemaPath:       "/schemas/crm.json",
			TimeoutMS:        7000,
			MaxResponseBytes: 16384,
			MaxDepth:         7,
			MaxFields:        48,
			MaxRootFields:    3,
			MaxFragments:     5,
			Domains: []graphQLDomainFileConfig{{
				Name:        "orders",
				RootQueries: []string{"order"},
				Types:       []string{"Order"},
			}},
		}},
		GraphQLMutationPolicies: []graphQLMutationPolicyFileConfig{{
			Name:              "capture_order",
			Source:            "crm",
			Domain:            "orders",
			RootMutation:      "captureOrder",
			IdempotencyMode:   "header",
			IdempotencyHeader: "Idempotency-Key",
			MaxDepth:          2,
			MaxFields:         8,
		}},
	}); err != nil {
		t.Fatalf("writeBridgeFileConfig: %v", err)
	}

	runtime, err := runtimeConfigFromEnv()
	if err != nil {
		t.Fatalf("runtimeConfigFromEnv: %v", err)
	}
	if len(runtime.GraphQL.MutationPolicies) != 1 {
		t.Fatalf("unexpected mutation policy count: %d", len(runtime.GraphQL.MutationPolicies))
	}
	policy := runtime.GraphQL.MutationPolicies[0]
	if policy.Name != "capture_order" || policy.RootMutation != "captureOrder" {
		t.Fatalf("unexpected mutation policy: %+v", policy)
	}
}

func TestConfigStoreUpdatePersistsGraphQLSourcesAndHidesAPIKeys(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")
	t.Setenv("GHOST_GRAPHQL_ENABLED", "true")
	t.Setenv("GHOST_GRAPHQL_ENDPOINT", "https://env.example/graphql")
	t.Setenv("GHOST_GRAPHQL_API_KEY", "env-graphql-key")
	t.Setenv("GHOST_GRAPHQL_SCHEMA_PATH", "/env/schema.json")

	store, err := NewConfigStoreFromEnv()
	if err != nil {
		t.Fatalf("NewConfigStoreFromEnv: %v", err)
	}

	defaultSource := "crm"
	crmAPIKey := "crm-secret"
	if err := store.Update(configUpdateRequest{
		GraphQLDefaultSource: &defaultSource,
		GraphQLSources: []GraphQLSourceInput{{
			Name:             "crm",
			Description:      "CRM",
			Endpoint:         "https://crm.example/graphql",
			APIKey:           &crmAPIKey,
			SchemaPath:       "/schemas/crm.json",
			TimeoutMS:        4500,
			MaxResponseBytes: 8192,
			MaxDepth:         5,
			MaxFields:        32,
			MaxRootFields:    2,
			MaxFragments:     4,
			Headers: map[string]string{
				"X-Tenant": "tenant-1",
			},
			Domains: []GraphQLDomainInput{{
				Name:        "orders",
				RootQueries: []string{"order"},
				Types:       []string{"Order"},
			}},
		}},
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	runtime := store.RuntimeConfig()
	if runtime.GraphQL.DefaultSource != "crm" {
		t.Fatalf("unexpected runtime graphql default source: %q", runtime.GraphQL.DefaultSource)
	}
	source := findGraphQLSource(t, runtime.GraphQL.Sources, "crm")
	if source.APIKey != crmAPIKey {
		t.Fatalf("unexpected runtime graphql api key: %q", source.APIKey)
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if fileCfg.GraphQLEndpoint != nil || fileCfg.GraphQLSchemaPath != nil || fileCfg.GraphQLAPIKey != nil {
		t.Fatalf("expected legacy graphql fields to be cleared, got %+v", fileCfg)
	}
	if fileCfg.GraphQLDefaultSource == nil || *fileCfg.GraphQLDefaultSource != "crm" {
		t.Fatalf("unexpected persisted graphql default source: %#v", fileCfg.GraphQLDefaultSource)
	}
	if len(fileCfg.GraphQLSources) != 1 || fileCfg.GraphQLSources[0].Name != "crm" {
		t.Fatalf("unexpected persisted graphql sources: %+v", fileCfg.GraphQLSources)
	}
	if fileCfg.GraphQLSources[0].APIKey == nil || *fileCfg.GraphQLSources[0].APIKey != crmAPIKey {
		t.Fatalf("unexpected persisted graphql api key: %+v", fileCfg.GraphQLSources[0].APIKey)
	}

	snapshot := store.Snapshot()
	if len(snapshot.GraphQLSources) != 1 || !snapshot.GraphQLSources[0].APIKeySet {
		t.Fatalf("unexpected graphql source snapshot: %+v", snapshot.GraphQLSources)
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(encoded), crmAPIKey) {
		t.Fatalf("graphql source api key should not leak in snapshot: %s", string(encoded))
	}
}

func TestRuntimeConfigFromEnvFailsOnInvalidGraphQLSource(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	if err := writeBridgeFileConfig(configPath, bridgeFileConfig{
		GraphQLDefaultSource: stringPointer("crm"),
		GraphQLSources: []graphQLSourceFileConfig{{
			Name:       "crm",
			Endpoint:   "https://crm.example/graphql",
			SchemaPath: "",
		}},
	}); err != nil {
		t.Fatalf("writeBridgeFileConfig: %v", err)
	}

	if _, err := runtimeConfigFromEnv(); err == nil {
		t.Fatal("expected invalid graphql source config error")
	}
}

func TestRuntimeConfigFromEnvFailsOnInvalidGraphQLMutationPolicySource(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	if err := writeBridgeFileConfig(configPath, bridgeFileConfig{
		GraphQLDefaultSource: stringPointer("crm"),
		GraphQLSources: []graphQLSourceFileConfig{{
			Name:             "crm",
			Endpoint:         "https://crm.example/graphql",
			SchemaPath:       "/schemas/crm.json",
			TimeoutMS:        3000,
			MaxResponseBytes: 4096,
			MaxDepth:         6,
			MaxFields:        16,
			MaxRootFields:    2,
			MaxFragments:     4,
			Domains: []graphQLDomainFileConfig{{
				Name:        "orders",
				RootQueries: []string{"order"},
			}},
		}},
		GraphQLMutationPolicies: []graphQLMutationPolicyFileConfig{{
			Name:              "bad_policy",
			Source:            "billing",
			Domain:            "orders",
			RootMutation:      "captureOrder",
			IdempotencyMode:   "header",
			IdempotencyHeader: "Idempotency-Key",
		}},
	}); err != nil {
		t.Fatalf("writeBridgeFileConfig: %v", err)
	}

	if _, err := runtimeConfigFromEnv(); err == nil || !strings.Contains(err.Error(), `source "billing"`) {
		t.Fatalf("expected invalid mutation policy source error, got %v", err)
	}
}

func TestRuntimeConfigFromEnvFailsOnMissingGraphQLMutationIdempotency(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	if err := writeBridgeFileConfig(configPath, bridgeFileConfig{
		GraphQLDefaultSource: stringPointer("crm"),
		GraphQLSources: []graphQLSourceFileConfig{{
			Name:             "crm",
			Endpoint:         "https://crm.example/graphql",
			SchemaPath:       "/schemas/crm.json",
			TimeoutMS:        3000,
			MaxResponseBytes: 4096,
			MaxDepth:         6,
			MaxFields:        16,
			MaxRootFields:    2,
			MaxFragments:     4,
			Domains: []graphQLDomainFileConfig{{
				Name:        "orders",
				RootQueries: []string{"order"},
			}},
		}},
		GraphQLMutationPolicies: []graphQLMutationPolicyFileConfig{{
			Name:         "missing_idempotency",
			Source:       "crm",
			Domain:       "orders",
			RootMutation: "captureOrder",
		}},
	}); err != nil {
		t.Fatalf("writeBridgeFileConfig: %v", err)
	}

	if _, err := runtimeConfigFromEnv(); err == nil || !strings.Contains(err.Error(), "idempotency_mode") {
		t.Fatalf("expected missing idempotency error, got %v", err)
	}
}

func findGraphQLSource(t *testing.T, sources []GraphQLSourceConfig, name string) GraphQLSourceConfig {
	t.Helper()
	for _, source := range sources {
		if source.Name == name {
			return source
		}
	}
	t.Fatalf("graphql source %q was not found in %+v", name, sources)
	return GraphQLSourceConfig{}
}
