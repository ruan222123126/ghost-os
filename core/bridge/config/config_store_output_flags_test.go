package config

import (
	"path/filepath"
	"testing"
)

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
	if err := store.Update(configUpdateRequest{ToolCallCompactOutputEnabled: &enabled}); err != nil {
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
		t.Fatalf("unexpected persisted tool_call_compact_output_enabled: %#v", fileCfg.ToolCallCompactOutputEnabled)
	}
}

func TestConfigStoreUpdatePersistsMemoryModeEnabled(t *testing.T) {
	store := newOutputFlagStore(t)
	enabled := true
	if err := store.Update(configUpdateRequest{MemoryModeEnabled: &enabled}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !store.RuntimeConfig().MemoryModeEnabled || !store.Snapshot().MemoryModeEnabled {
		t.Fatal("expected memory_mode_enabled to be true")
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
	store := newOutputFlagStore(t)
	enabled := true
	if err := store.Update(configUpdateRequest{MicrocompactEnabled: &enabled}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !store.RuntimeConfig().MicrocompactEnabled || !store.Snapshot().MicrocompactEnabled {
		t.Fatal("expected microcompact_enabled to be true")
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if fileCfg.MicrocompactEnabled == nil || !*fileCfg.MicrocompactEnabled {
		t.Fatalf("unexpected persisted microcompact_enabled: %#v", fileCfg.MicrocompactEnabled)
	}
}

func newOutputFlagStore(t *testing.T) *store {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")

	store, err := newStoreFromEnv()
	if err != nil {
		t.Fatalf("newStoreFromEnv: %v", err)
	}
	return store
}
