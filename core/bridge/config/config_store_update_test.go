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
	t.Setenv("GHOST_SESSION_HUMAN_LOG_FULL_ENABLED", "true")

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
	if snapshot.WebRooterBaseURL != defaultWebRooterBaseURL {
		t.Fatalf("unexpected web_rooter_base_url: got %q want %q", snapshot.WebRooterBaseURL, defaultWebRooterBaseURL)
	}
	if snapshot.WebRooterTimeoutMS != defaultWebRooterTimeoutMS {
		t.Fatalf(
			"unexpected web_rooter_timeout_ms: got %d want %d",
			snapshot.WebRooterTimeoutMS,
			defaultWebRooterTimeoutMS,
		)
	}
	if !snapshot.WebRooterAPITokenSet {
		t.Fatal("expected web_rooter_api_token_set to be true")
	}
	if !snapshot.SessionHumanLogFullEnabled {
		t.Fatal("expected session_human_log_full_enabled to be true")
	}
	if !snapshot.AssistantMarkdownEnabled {
		t.Fatal("expected assistant_markdown_enabled to be true")
	}
	if snapshot.MemoryModeEnabled {
		t.Fatal("expected memory_mode_enabled to be false")
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
	if fileCfg.SessionHumanLogFullEnabled != nil {
		t.Fatalf("snapshot should not persist session_human_log_full_enabled, got %#v", fileCfg.SessionHumanLogFullEnabled)
	}
	if fileCfg.AssistantMarkdownEnabled != nil {
		t.Fatalf("snapshot should not persist assistant_markdown_enabled, got %#v", fileCfg.AssistantMarkdownEnabled)
	}
	if fileCfg.MemoryModeEnabled != nil {
		t.Fatalf("snapshot should not persist memory_mode_enabled, got %#v", fileCfg.MemoryModeEnabled)
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

func TestConfigStoreUpdatePersistsSessionHumanLogFullEnabled(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")

	store, err := newStoreFromEnv()
	if err != nil {
		t.Fatalf("newStoreFromEnv: %v", err)
	}

	enabled := true
	if err := store.Update(configUpdateRequest{
		SessionHumanLogFullEnabled: &enabled,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !store.RuntimeConfig().SessionHumanLogFullEnabled {
		t.Fatal("expected runtime session_human_log_full_enabled to be true")
	}
	if !store.Snapshot().SessionHumanLogFullEnabled {
		t.Fatal("expected snapshot session_human_log_full_enabled to be true")
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if fileCfg.SessionHumanLogFullEnabled == nil || !*fileCfg.SessionHumanLogFullEnabled {
		t.Fatalf("unexpected persisted session_human_log_full_enabled: %#v", fileCfg.SessionHumanLogFullEnabled)
	}
}

func TestConfigStoreUpdatePersistsAssistantMarkdownEnabled(t *testing.T) {
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
		AssistantMarkdownEnabled: &disabled,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if store.RuntimeConfig().AssistantMarkdownEnabled {
		t.Fatal("expected runtime assistant_markdown_enabled to be false")
	}
	if store.Snapshot().AssistantMarkdownEnabled {
		t.Fatal("expected snapshot assistant_markdown_enabled to be false")
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if fileCfg.AssistantMarkdownEnabled == nil || *fileCfg.AssistantMarkdownEnabled {
		t.Fatalf("unexpected persisted assistant_markdown_enabled: %#v", fileCfg.AssistantMarkdownEnabled)
	}
}

func TestConfigStoreUpdatePersistsMemoryModeEnabled(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")

	store, err := newStoreFromEnv()
	if err != nil {
		t.Fatalf("newStoreFromEnv: %v", err)
	}

	enabled := true
	if err := store.Update(configUpdateRequest{
		MemoryModeEnabled: &enabled,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !store.RuntimeConfig().MemoryModeEnabled {
		t.Fatal("expected runtime memory_mode_enabled to be true")
	}
	if !store.Snapshot().MemoryModeEnabled {
		t.Fatal("expected snapshot memory_mode_enabled to be true")
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if fileCfg.MemoryModeEnabled == nil || !*fileCfg.MemoryModeEnabled {
		t.Fatalf("unexpected persisted memory_mode_enabled: %#v", fileCfg.MemoryModeEnabled)
	}
}
