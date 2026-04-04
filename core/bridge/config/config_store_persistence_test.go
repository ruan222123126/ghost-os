package config

import (
	"path/filepath"
	"testing"

	"ghost-os/bridge/llm"
)

func TestConfigStoreProviderCRUDPersistsToml(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	store, err := newStoreFromEnv()
	if err != nil {
		t.Fatalf("newStoreFromEnv: %v", err)
	}

	if err := store.AddProvider(ProviderRecord{
		Name:    "crs",
		Type:    llm.ProviderCustom,
		BaseURL: "https://lldai.online/openai",
		APIKey:  optionalStringPointer("sk-xxx"),
		Models:  []string{"gpt-5.4", "gpt-4"},
	}); err != nil {
		t.Fatalf("AddProvider: %v", err)
	}
	if err := store.AddProvider(ProviderRecord{
		Name:    "openai",
		Type:    llm.ProviderOpenAI,
		BaseURL: defaultBaseURL,
		APIKey:  optionalStringPointer("sk-yyy"),
	}); err != nil {
		t.Fatalf("AddProvider second provider: %v", err)
	}
	if err := store.SetActiveProvider("openai"); err != nil {
		t.Fatalf("SetActiveProvider: %v", err)
	}
	if err := store.UpdateProvider("openai", ProviderRecord{
		Name:    "openai",
		Type:    llm.ProviderOpenAI,
		BaseURL: "https://api.openai.com/v1",
		Models:  []string{"gpt-5.4"},
	}); err != nil {
		t.Fatalf("UpdateProvider: %v", err)
	}
	if err := store.DeleteProvider("crs"); err != nil {
		t.Fatalf("DeleteProvider: %v", err)
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if fileCfg.ActiveProvider == nil || *fileCfg.ActiveProvider != "openai" {
		t.Fatalf("unexpected active provider: %#v", fileCfg.ActiveProvider)
	}
	providers := normalizeProviderConfigs(fileCfg.Providers, stringValue(fileCfg.Model))
	if len(providers) != 1 {
		t.Fatalf("unexpected provider count: got %d want 1", len(providers))
	}
	if providers[0].APIKey == nil || *providers[0].APIKey != "sk-yyy" {
		t.Fatalf("expected update to preserve api key, got %#v", providers[0].APIKey)
	}
	if providers[0].Type != llm.ProviderOpenAI {
		t.Fatalf("expected provider type to persist, got %q", providers[0].Type)
	}
	if runtime := store.RuntimeConfig(); runtime.ProviderName != "openai" {
		t.Fatalf("unexpected runtime provider: got %q want %q", runtime.ProviderName, "openai")
	}
}

func TestConfigStoreRuntimeConfigRemainsSnapshotAfterEnvChanges(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_API_KEY", "initial-key")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")
	t.Setenv("GHOST_MODEL", "initial-model")
	t.Setenv("GHOST_CHAT_PATH", "/initial/chat")

	store, err := newStoreFromEnv()
	if err != nil {
		t.Fatalf("newStoreFromEnv: %v", err)
	}

	t.Setenv("GHOST_PROVIDER", "openai")
	t.Setenv("GHOST_API_KEY", "drifted-key")
	t.Setenv("GHOST_BASE_URL", "https://drifted.example/v1")
	t.Setenv("GHOST_MODEL", "drifted-model")
	t.Setenv("GHOST_CHAT_PATH", "/drifted/chat")

	runtime := store.RuntimeConfig()
	if runtime.ProviderName != "custom" {
		t.Fatalf("unexpected provider name: got %q want %q", runtime.ProviderName, "custom")
	}
	if runtime.APIKey != "initial-key" {
		t.Fatalf("unexpected api key: got %q want %q", runtime.APIKey, "initial-key")
	}
	if runtime.BaseURL != "https://initial.example/v1" {
		t.Fatalf("unexpected base url: got %q want %q", runtime.BaseURL, "https://initial.example/v1")
	}
	if runtime.Model != "initial-model" {
		t.Fatalf("unexpected model: got %q want %q", runtime.Model, "initial-model")
	}
	if runtime.ChatPath != "/initial/chat" {
		t.Fatalf("unexpected chat path: got %q want %q", runtime.ChatPath, "/initial/chat")
	}
}

func TestConfigStoreUpdateKeepsSnapshotFallbackInsteadOfReloadingEnv(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_API_KEY", "initial-key")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")
	t.Setenv("GHOST_MODEL", "initial-model")

	store, err := newStoreFromEnv()
	if err != nil {
		t.Fatalf("newStoreFromEnv: %v", err)
	}

	t.Setenv("GHOST_API_KEY", "drifted-key")
	t.Setenv("GHOST_BASE_URL", "https://drifted.example/v1")

	newBaseURL := "https://persisted.example/v1"
	if err := store.Update(configUpdateRequest{BaseURL: &newBaseURL}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	runtime := store.RuntimeConfig()
	if runtime.APIKey != "initial-key" {
		t.Fatalf("unexpected runtime api key: got %q want %q", runtime.APIKey, "initial-key")
	}
	if runtime.BaseURL != newBaseURL {
		t.Fatalf("unexpected runtime base url: got %q want %q", runtime.BaseURL, newBaseURL)
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	providers := normalizeProviderConfigs(fileCfg.Providers, stringValue(fileCfg.Model))
	if len(providers) != 1 {
		t.Fatalf("unexpected provider count: got %d want 1", len(providers))
	}
	if providers[0].APIKey == nil || *providers[0].APIKey != "initial-key" {
		t.Fatalf("unexpected persisted api key: %#v", providers[0].APIKey)
	}
	if providers[0].BaseURL != newBaseURL {
		t.Fatalf("unexpected persisted base url: got %q want %q", providers[0].BaseURL, newBaseURL)
	}
}

func TestConfigStoreUpdatePersistsWebSearchSettings(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")
	t.Setenv("GHOST_WEB_SEARCH_TAVILY_URL", "https://proxy.example/tavily")
	t.Setenv("GHOST_WEB_SEARCH_TAVILY_API_KEY", "initial-tavily-key")

	store, err := newStoreFromEnv()
	if err != nil {
		t.Fatalf("newStoreFromEnv: %v", err)
	}

	webSearchExaURL := "https://proxy.example/exa"
	webSearchExaAPIKey := "updated-exa-key"
	if err := store.Update(configUpdateRequest{
		WebSearchExaURL:    &webSearchExaURL,
		WebSearchExaAPIKey: &webSearchExaAPIKey,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	runtime := store.RuntimeConfig()
	if runtime.WebSearchTavilyURL != "https://proxy.example/tavily" {
		t.Fatalf("unexpected tavily url: got %q want %q", runtime.WebSearchTavilyURL, "https://proxy.example/tavily")
	}
	if runtime.WebSearchExaURL != webSearchExaURL {
		t.Fatalf("unexpected exa url: got %q want %q", runtime.WebSearchExaURL, webSearchExaURL)
	}
	if runtime.WebSearchTavilyAPIKey != "initial-tavily-key" {
		t.Fatalf("unexpected tavily api key: got %q want %q", runtime.WebSearchTavilyAPIKey, "initial-tavily-key")
	}
	if runtime.WebSearchExaAPIKey != webSearchExaAPIKey {
		t.Fatalf("unexpected exa api key: got %q want %q", runtime.WebSearchExaAPIKey, webSearchExaAPIKey)
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if fileCfg.WebSearchTavilyAPIKey == nil || *fileCfg.WebSearchTavilyAPIKey != "initial-tavily-key" {
		t.Fatalf("unexpected persisted tavily api key: %#v", fileCfg.WebSearchTavilyAPIKey)
	}
	if fileCfg.WebSearchTavilyURL == nil || *fileCfg.WebSearchTavilyURL != "https://proxy.example/tavily" {
		t.Fatalf("unexpected persisted tavily url: %#v", fileCfg.WebSearchTavilyURL)
	}
	if fileCfg.WebSearchExaURL == nil || *fileCfg.WebSearchExaURL != webSearchExaURL {
		t.Fatalf("unexpected persisted exa url: %#v", fileCfg.WebSearchExaURL)
	}
	if fileCfg.WebSearchExaAPIKey == nil || *fileCfg.WebSearchExaAPIKey != webSearchExaAPIKey {
		t.Fatalf("unexpected persisted exa api key: %#v", fileCfg.WebSearchExaAPIKey)
	}

	snapshot := store.Snapshot()
	if snapshot.WebSearchTavilyURL != "https://proxy.example/tavily" {
		t.Fatalf("unexpected snapshot tavily url: got %q want %q", snapshot.WebSearchTavilyURL, "https://proxy.example/tavily")
	}
	if snapshot.WebSearchExaURL != webSearchExaURL {
		t.Fatalf("unexpected snapshot exa url: got %q want %q", snapshot.WebSearchExaURL, webSearchExaURL)
	}
	if !snapshot.WebSearchTavilyAPIKeySet {
		t.Fatal("expected tavily api key flag to stay true")
	}
	if !snapshot.WebSearchExaAPIKeySet {
		t.Fatal("expected exa api key flag to be true")
	}
}

func TestConfigStoreUpdatePersistsWebRooterSettings(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")
	t.Setenv("GHOST_WEB_ROOTER_ENABLED", "false")
	t.Setenv("GHOST_WEB_ROOTER_BASE_URL", "http://127.0.0.1:8765")

	store, err := newStoreFromEnv()
	if err != nil {
		t.Fatalf("newStoreFromEnv: %v", err)
	}

	webRooterEnabled := true
	webRooterAPIToken := "updated-rooter-token"
	webRooterTimeoutMS := 12_345
	if err := store.Update(configUpdateRequest{
		WebRooterEnabled:   &webRooterEnabled,
		WebRooterAPIToken:  &webRooterAPIToken,
		WebRooterTimeoutMS: &webRooterTimeoutMS,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	runtime := store.RuntimeConfig()
	if !runtime.WebRooterEnabled {
		t.Fatal("expected runtime web_rooter to be enabled")
	}
	if runtime.WebRooterBaseURL != "http://127.0.0.1:8765" {
		t.Fatalf("unexpected web_rooter base url: got %q want %q", runtime.WebRooterBaseURL, "http://127.0.0.1:8765")
	}
	if runtime.WebRooterAPIToken != webRooterAPIToken {
		t.Fatalf("unexpected web_rooter api token: got %q want %q", runtime.WebRooterAPIToken, webRooterAPIToken)
	}
	if runtime.WebRooterTimeoutMS != webRooterTimeoutMS {
		t.Fatalf("unexpected web_rooter timeout: got %d want %d", runtime.WebRooterTimeoutMS, webRooterTimeoutMS)
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if fileCfg.WebRooterEnabled == nil || !*fileCfg.WebRooterEnabled {
		t.Fatalf("unexpected persisted web_rooter enabled: %#v", fileCfg.WebRooterEnabled)
	}
	if fileCfg.WebRooterBaseURL == nil || *fileCfg.WebRooterBaseURL != "http://127.0.0.1:8765" {
		t.Fatalf("unexpected persisted web_rooter base url: %#v", fileCfg.WebRooterBaseURL)
	}
	if fileCfg.WebRooterAPIToken == nil || *fileCfg.WebRooterAPIToken != webRooterAPIToken {
		t.Fatalf("unexpected persisted web_rooter api token: %#v", fileCfg.WebRooterAPIToken)
	}
	if fileCfg.WebRooterTimeoutMS == nil || *fileCfg.WebRooterTimeoutMS != webRooterTimeoutMS {
		t.Fatalf("unexpected persisted web_rooter timeout: %#v", fileCfg.WebRooterTimeoutMS)
	}

	snapshot := store.Snapshot()
	if !snapshot.WebRooterEnabled {
		t.Fatal("expected web_rooter enabled flag to be true")
	}
	if !snapshot.WebRooterAPITokenSet {
		t.Fatal("expected web_rooter api token flag to be true")
	}
}
