package config

import (
	"path/filepath"
	"testing"
)

func TestConfigSessionTitleModeDefaultsToSessionID(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")

	store, err := newStoreFromEnv()
	if err != nil {
		t.Fatalf("newStoreFromEnv: %v", err)
	}
	if store.Snapshot().SessionTitleMode != defaultSessionTitleMode {
		t.Fatalf("unexpected session_title_mode: got %q", store.Snapshot().SessionTitleMode)
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if fileCfg.SessionTitleMode != nil {
		t.Fatalf("snapshot should not persist session_title_mode, got %#v", fileCfg.SessionTitleMode)
	}
}

func TestConfigStoreUpdatePersistsSessionTitleMode(t *testing.T) {
	store := newSessionTitleModeStore(t)
	mode := SessionTitleModeFirstMessage
	if err := store.Update(configUpdateRequest{SessionTitleMode: &mode}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if store.RuntimeConfig().SessionTitleMode != mode {
		t.Fatalf("unexpected runtime session_title_mode: got %q", store.RuntimeConfig().SessionTitleMode)
	}
	if store.Snapshot().SessionTitleMode != mode {
		t.Fatalf("unexpected snapshot session_title_mode: got %q", store.Snapshot().SessionTitleMode)
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if fileCfg.SessionTitleMode == nil || *fileCfg.SessionTitleMode != mode {
		t.Fatalf("unexpected persisted session_title_mode: %#v", fileCfg.SessionTitleMode)
	}
}

func TestConfigStoreUpdateRejectsInvalidSessionTitleMode(t *testing.T) {
	store := newSessionTitleModeStore(t)
	mode := "bad"
	if err := store.Update(configUpdateRequest{SessionTitleMode: &mode}); err == nil {
		t.Fatal("expected invalid session_title_mode error")
	}
}

func newSessionTitleModeStore(t *testing.T) *store {
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
