package app

import (
	"path/filepath"
	"testing"

	"ghost-os/bridge/llm"
)

func TestConfigStoreProviderCRUDPersistsToml(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	store, err := NewConfigStoreFromEnv()
	if err != nil {
		t.Fatalf("NewConfigStoreFromEnv: %v", err)
	}

	if err := store.AddProvider(providerConfig{
		Name:    "crs",
		Type:    llm.ProviderCustom,
		BaseURL: "https://lldai.online/openai",
		APIKey:  optionalStringPointer("sk-xxx"),
		Models:  []string{"gpt-5.4", "gpt-4"},
	}); err != nil {
		t.Fatalf("AddProvider: %v", err)
	}
	if err := store.AddProvider(providerConfig{
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
	if err := store.UpdateProvider("openai", providerConfig{
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

	store, err := NewConfigStoreFromEnv()
	if err != nil {
		t.Fatalf("NewConfigStoreFromEnv: %v", err)
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

	store, err := NewConfigStoreFromEnv()
	if err != nil {
		t.Fatalf("NewConfigStoreFromEnv: %v", err)
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
