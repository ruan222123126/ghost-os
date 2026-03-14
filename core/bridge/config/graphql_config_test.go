package config

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeConfigFromEnvResolvesGraphQLSettings(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_GRAPHQL_ENABLED", "true")
	t.Setenv("GHOST_GRAPHQL_ENDPOINT", "https://env.example/graphql")
	t.Setenv("GHOST_GRAPHQL_API_KEY", "env-graphql-key")
	t.Setenv("GHOST_GRAPHQL_SCHEMA_PATH", "/env/schema.json")
	t.Setenv("GHOST_GRAPHQL_TIMEOUT_MS", "9000")
	t.Setenv("GHOST_GRAPHQL_MAX_RESPONSE_BYTES", "8192")
	t.Setenv("GHOST_GRAPHQL_HEADERS", `{"X-Env":"env"}`)

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

	if !runtime.GraphQL.Enabled {
		t.Fatal("expected graphql to stay enabled")
	}
	if runtime.GraphQL.Endpoint != "https://file.example/graphql" {
		t.Fatalf("unexpected graphql endpoint: %q", runtime.GraphQL.Endpoint)
	}
	if runtime.GraphQL.APIKey != "env-graphql-key" {
		t.Fatalf("unexpected graphql api key: %q", runtime.GraphQL.APIKey)
	}
	if runtime.GraphQL.SchemaPath != "/file/schema.json" {
		t.Fatalf("unexpected graphql schema path: %q", runtime.GraphQL.SchemaPath)
	}
	if runtime.GraphQL.TimeoutMS != 5000 {
		t.Fatalf("unexpected graphql timeout: %d", runtime.GraphQL.TimeoutMS)
	}
	if runtime.GraphQL.MaxResponseBytes != 8192 {
		t.Fatalf("unexpected graphql max bytes: %d", runtime.GraphQL.MaxResponseBytes)
	}
	if runtime.GraphQL.Headers["X-File"] != "file" {
		t.Fatalf("unexpected graphql headers: %+v", runtime.GraphQL.Headers)
	}
}

func TestConfigStoreUpdatePersistsGraphQLSettingsAndHidesAPIKey(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")
	t.Setenv("GHOST_GRAPHQL_ENABLED", "true")
	t.Setenv("GHOST_GRAPHQL_ENDPOINT", "https://env.example/graphql")
	t.Setenv("GHOST_GRAPHQL_API_KEY", "env-graphql-key")
	t.Setenv("GHOST_GRAPHQL_SCHEMA_PATH", "/env/schema.json")
	t.Setenv("GHOST_GRAPHQL_TIMEOUT_MS", "9000")
	t.Setenv("GHOST_GRAPHQL_MAX_RESPONSE_BYTES", "4096")

	store, err := NewConfigStoreFromEnv()
	if err != nil {
		t.Fatalf("NewConfigStoreFromEnv: %v", err)
	}

	endpoint := "https://persisted.example/graphql"
	schemaPath := "/persisted/schema.json"
	timeoutMS := 2500
	if err := store.Update(configUpdateRequest{
		GraphQLEndpoint:   &endpoint,
		GraphQLSchemaPath: &schemaPath,
		GraphQLTimeoutMS:  &timeoutMS,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	runtime := store.RuntimeConfig()
	if runtime.GraphQL.Endpoint != endpoint {
		t.Fatalf("unexpected runtime graphql endpoint: %q", runtime.GraphQL.Endpoint)
	}
	if runtime.GraphQL.SchemaPath != schemaPath {
		t.Fatalf("unexpected runtime graphql schema path: %q", runtime.GraphQL.SchemaPath)
	}
	if runtime.GraphQL.TimeoutMS != timeoutMS {
		t.Fatalf("unexpected runtime graphql timeout: %d", runtime.GraphQL.TimeoutMS)
	}
	if runtime.GraphQL.APIKey != "env-graphql-key" {
		t.Fatalf("expected api key snapshot to survive materialization, got %q", runtime.GraphQL.APIKey)
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if fileCfg.GraphQLEndpoint == nil || *fileCfg.GraphQLEndpoint != endpoint {
		t.Fatalf("unexpected persisted graphql endpoint: %#v", fileCfg.GraphQLEndpoint)
	}
	if fileCfg.GraphQLSchemaPath == nil || *fileCfg.GraphQLSchemaPath != schemaPath {
		t.Fatalf("unexpected persisted graphql schema path: %#v", fileCfg.GraphQLSchemaPath)
	}
	if fileCfg.GraphQLTimeoutMS == nil || *fileCfg.GraphQLTimeoutMS != timeoutMS {
		t.Fatalf("unexpected persisted graphql timeout: %#v", fileCfg.GraphQLTimeoutMS)
	}
	if fileCfg.GraphQLAPIKey == nil || *fileCfg.GraphQLAPIKey != "env-graphql-key" {
		t.Fatalf("unexpected persisted graphql api key: %#v", fileCfg.GraphQLAPIKey)
	}

	snapshot := store.Snapshot()
	if !snapshot.GraphQLAPIKeySet {
		t.Fatal("expected graphql api key flag to stay true")
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(encoded) == "" {
		t.Fatal("expected snapshot json to be non-empty")
	}
	if containsJSONString(string(encoded), "graphql_api_key") {
		t.Fatalf("graphql api key should not leak in snapshot: %s", string(encoded))
	}
}

func containsJSONString(source string, key string) bool {
	return strings.Contains(source, `"`+key+`"`)
}
