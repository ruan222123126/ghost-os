package config

import (
	"path/filepath"
	"testing"
)

func TestConfigStoreUpdateLoadsRuntimeGraphQLIntoPatchBase(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")

	store, err := NewConfigStoreFromEnv()
	if err != nil {
		t.Fatalf("NewConfigStoreFromEnv: %v", err)
	}

	defaultSource := "crm"
	initialAPIKey := "crm-secret"
	if err := store.Update(configUpdateRequest{
		GraphQLDefaultSource: &defaultSource,
		GraphQLSources: []GraphQLSourceInput{{
			Name:       "crm",
			Endpoint:   "https://crm.example/graphql",
			APIKey:     &initialAPIKey,
			SchemaPath: "/schemas/crm.json",
		}},
	}); err != nil {
		t.Fatalf("seed Update: %v", err)
	}

	if err := writeBridgeFileConfig(configPath, bridgeFileConfig{}); err != nil {
		t.Fatalf("writeBridgeFileConfig: %v", err)
	}

	updatedEndpoint := "https://crm-v2.example/graphql"
	updatedSchemaPath := "/schemas/crm-v2.json"
	if err := store.Update(configUpdateRequest{
		GraphQLSourceUpsert: &GraphQLSourceInput{
			Name:       "crm",
			Endpoint:   updatedEndpoint,
			SchemaPath: updatedSchemaPath,
		},
	}); err != nil {
		t.Fatalf("Update after clearing file: %v", err)
	}

	runtime := store.RuntimeConfig()
	source := findGraphQLSource(t, runtime.GraphQL.Sources, "crm")
	if source.APIKey != initialAPIKey {
		t.Fatalf("unexpected runtime graphql api key: got %q want %q", source.APIKey, initialAPIKey)
	}
	if source.Endpoint != updatedEndpoint {
		t.Fatalf("unexpected runtime graphql endpoint: got %q want %q", source.Endpoint, updatedEndpoint)
	}
	if source.SchemaPath != updatedSchemaPath {
		t.Fatalf("unexpected runtime graphql schema path: got %q want %q", source.SchemaPath, updatedSchemaPath)
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if len(fileCfg.GraphQLSources) != 1 {
		t.Fatalf("unexpected persisted graphql sources: %+v", fileCfg.GraphQLSources)
	}
	if fileCfg.GraphQLSources[0].APIKey == nil || *fileCfg.GraphQLSources[0].APIKey != initialAPIKey {
		t.Fatalf("unexpected persisted graphql api key: %#v", fileCfg.GraphQLSources[0].APIKey)
	}
	if fileCfg.GraphQLSources[0].Endpoint != updatedEndpoint {
		t.Fatalf("unexpected persisted graphql endpoint: got %q want %q", fileCfg.GraphQLSources[0].Endpoint, updatedEndpoint)
	}
	if fileCfg.GraphQLSources[0].SchemaPath != updatedSchemaPath {
		t.Fatalf(
			"unexpected persisted graphql schema path: got %q want %q",
			fileCfg.GraphQLSources[0].SchemaPath,
			updatedSchemaPath,
		)
	}
}
