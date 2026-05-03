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
	if snapshot.MaxTurns != defaultMaxTurns {
		t.Fatalf("unexpected max_turns: got %d want %d", snapshot.MaxTurns, defaultMaxTurns)
	}
	if !snapshot.SessionHumanLogFullEnabled {
		t.Fatal("expected session_human_log_full_enabled to be true")
	}
	if !snapshot.SessionSystemPromptVisible {
		t.Fatal("expected session_system_prompt_visible_enabled to be true")
	}
	if !snapshot.AssistantMarkdownEnabled {
		t.Fatal("expected assistant_markdown_enabled to be true")
	}
	if snapshot.ToolCallCompactOutputEnabled {
		t.Fatal("expected tool_call_compact_output_enabled to be false")
	}
	if snapshot.MemoryModeEnabled {
		t.Fatal("expected memory_mode_enabled to be false")
	}
	if snapshot.MicrocompactEnabled {
		t.Fatal("expected microcompact_enabled to be false")
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
	if fileCfg.SessionHumanLogFullEnabled != nil {
		t.Fatalf("snapshot should not persist session_human_log_full_enabled, got %#v", fileCfg.SessionHumanLogFullEnabled)
	}
	if fileCfg.MaxTurns != nil {
		t.Fatalf("snapshot should not persist max_turns, got %#v", fileCfg.MaxTurns)
	}
	if fileCfg.SessionSystemPromptVisible != nil {
		t.Fatalf("snapshot should not persist session_system_prompt_visible_enabled, got %#v", fileCfg.SessionSystemPromptVisible)
	}
	if fileCfg.AssistantMarkdownEnabled != nil {
		t.Fatalf("snapshot should not persist assistant_markdown_enabled, got %#v", fileCfg.AssistantMarkdownEnabled)
	}
	if fileCfg.ToolCallCompactOutputEnabled != nil {
		t.Fatalf(
			"snapshot should not persist tool_call_compact_output_enabled, got %#v",
			fileCfg.ToolCallCompactOutputEnabled,
		)
	}
	if fileCfg.MemoryModeEnabled != nil {
		t.Fatalf("snapshot should not persist memory_mode_enabled, got %#v", fileCfg.MemoryModeEnabled)
	}
	if fileCfg.MicrocompactEnabled != nil {
		t.Fatalf("snapshot should not persist microcompact_enabled, got %#v", fileCfg.MicrocompactEnabled)
	}
}

func TestConfigStoreUpdatePersistsMaxTurns(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")

	store, err := newStoreFromEnv()
	if err != nil {
		t.Fatalf("newStoreFromEnv: %v", err)
	}

	value := 9
	if err := store.Update(configUpdateRequest{
		MaxTurns: &value,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if store.RuntimeConfig().MaxTurns != value {
		t.Fatalf("expected runtime max_turns to be %d", value)
	}
	if store.Snapshot().MaxTurns != value {
		t.Fatalf("expected snapshot max_turns to be %d", value)
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if fileCfg.MaxTurns == nil || *fileCfg.MaxTurns != value {
		t.Fatalf("unexpected persisted max_turns: %#v", fileCfg.MaxTurns)
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

func TestConfigStoreUpdatePersistsSessionSystemPromptVisibleEnabled(t *testing.T) {
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
		SessionSystemPromptVisible: &disabled,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if store.RuntimeConfig().SessionSystemPromptVisible {
		t.Fatal("expected runtime session_system_prompt_visible_enabled to be false")
	}
	if store.Snapshot().SessionSystemPromptVisible {
		t.Fatal("expected snapshot session_system_prompt_visible_enabled to be false")
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if fileCfg.SessionSystemPromptVisible == nil || *fileCfg.SessionSystemPromptVisible {
		t.Fatalf("unexpected persisted session_system_prompt_visible_enabled: %#v", fileCfg.SessionSystemPromptVisible)
	}
}

func TestConfigStoreUpdatePersistsToolCallCompactOutputEnabled(t *testing.T) {
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
		ToolCallCompactOutputEnabled: &enabled,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !store.RuntimeConfig().ToolCallCompactOutputEnabled {
		t.Fatal("expected runtime tool_call_compact_output_enabled to be true")
	}
	if !store.Snapshot().ToolCallCompactOutputEnabled {
		t.Fatal("expected snapshot tool_call_compact_output_enabled to be true")
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if fileCfg.ToolCallCompactOutputEnabled == nil || !*fileCfg.ToolCallCompactOutputEnabled {
		t.Fatalf(
			"unexpected persisted tool_call_compact_output_enabled: %#v",
			fileCfg.ToolCallCompactOutputEnabled,
		)
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

func TestConfigStoreUpdatePersistsMicrocompactEnabled(t *testing.T) {
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
		MicrocompactEnabled: &enabled,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !store.RuntimeConfig().MicrocompactEnabled {
		t.Fatal("expected runtime microcompact_enabled to be true")
	}
	if !store.Snapshot().MicrocompactEnabled {
		t.Fatal("expected snapshot microcompact_enabled to be true")
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if fileCfg.MicrocompactEnabled == nil || !*fileCfg.MicrocompactEnabled {
		t.Fatalf("unexpected persisted microcompact_enabled: %#v", fileCfg.MicrocompactEnabled)
	}
}
