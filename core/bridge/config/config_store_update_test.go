package config

import (
	"path/filepath"
	"testing"
)

func TestConfigStoreSnapshotDoesNotMaterializeRuntimeIntoFile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_API_KEY", "snapshot-key")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")
	t.Setenv("GHOST_MODEL", "snapshot-model")
	t.Setenv("GHOST_WEB_SEARCH_TAVILY_URL", "https://proxy.example/tavily")
	t.Setenv("GHOST_WEB_SEARCH_TAVILY_API_KEY", "snapshot-tavily")
	t.Setenv("GHOST_WEB_ROOTER_ENABLED", "true")
	t.Setenv("GHOST_WEB_ROOTER_API_TOKEN", "snapshot-rooter-token")

	store, err := newStoreFromEnv()
	if err != nil {
		t.Fatalf("newStoreFromEnv: %v", err)
	}

	snapshot := store.Snapshot()
	if snapshot.Provider != "custom" {
		t.Fatalf("unexpected provider: got %q want %q", snapshot.Provider, "custom")
	}
	if !snapshot.APIKeySet {
		t.Fatal("expected api_key_set to be true")
	}
	if !snapshot.WebSearchTavilyAPIKeySet {
		t.Fatal("expected web_search_tavily_api_key_set to be true")
	}
	if snapshot.WebSearchTavilyURL != "https://proxy.example/tavily" {
		t.Fatalf("unexpected web_search_tavily_url: got %q want %q", snapshot.WebSearchTavilyURL, "https://proxy.example/tavily")
	}
	if !snapshot.WebRooterEnabled {
		t.Fatal("expected web_rooter_enabled to be true")
	}
	if !snapshot.WebRooterAPITokenSet {
		t.Fatal("expected web_rooter_api_token_set to be true")
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if len(fileCfg.Providers) != 0 {
		t.Fatalf("snapshot should not materialize providers, got %+v", fileCfg.Providers)
	}
	if fileCfg.WebSearchTavilyAPIKey != nil {
		t.Fatalf("snapshot should not persist web search api key, got %#v", fileCfg.WebSearchTavilyAPIKey)
	}
	if fileCfg.WebSearchTavilyURL != nil {
		t.Fatalf("snapshot should not persist web search url, got %#v", fileCfg.WebSearchTavilyURL)
	}
	if fileCfg.WebRooterEnabled != nil {
		t.Fatalf("snapshot should not persist web_rooter enabled, got %#v", fileCfg.WebRooterEnabled)
	}
	if fileCfg.WebRooterAPIToken != nil {
		t.Fatalf("snapshot should not persist web_rooter api token, got %#v", fileCfg.WebRooterAPIToken)
	}
}

func TestConfigStoreUpdateLoadsRuntimeGraphQLIntoPatchBase(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")

	store, err := newStoreFromEnv()
	if err != nil {
		t.Fatalf("newStoreFromEnv: %v", err)
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

func TestConfigStoreUpdatePersistsGraphQLTextSanitizeSetting(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")

	store, err := newStoreFromEnv()
	if err != nil {
		t.Fatalf("newStoreFromEnv: %v", err)
	}

	disabled := false
	if err := store.Update(configUpdateRequest{
		GraphQLTextSanitizeEnabled: &disabled,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if store.RuntimeConfig().GraphQL.TextSanitizeEnabled {
		t.Fatal("expected runtime graphql_text_sanitize_enabled to be false")
	}
	if store.Snapshot().GraphQLTextSanitizeEnabled {
		t.Fatal("expected snapshot graphql_text_sanitize_enabled to be false")
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if fileCfg.GraphQLTextSanitizeEnabled == nil || *fileCfg.GraphQLTextSanitizeEnabled {
		t.Fatalf("unexpected persisted graphql_text_sanitize_enabled: %#v", fileCfg.GraphQLTextSanitizeEnabled)
	}
}
