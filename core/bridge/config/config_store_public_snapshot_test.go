package config

import (
	"path/filepath"
	"testing"
)

func TestPublicSnapshotHidesProviderDefaultChatPath(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")

	store, err := newStoreFromEnv()
	if err != nil {
		t.Fatalf("newStoreFromEnv: %v", err)
	}

	snapshot, err := store.PublicSnapshot()
	if err != nil {
		t.Fatalf("PublicSnapshot: %v", err)
	}
	if snapshot.ChatPath != "" {
		t.Fatalf("expected default chat_path to stay blank, got %q", snapshot.ChatPath)
	}
}

func TestPublicSnapshotKeepsExplicitRuntimeChatPath(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")
	t.Setenv("GHOST_CHAT_PATH", "/env/chat")

	store, err := newStoreFromEnv()
	if err != nil {
		t.Fatalf("newStoreFromEnv: %v", err)
	}

	snapshot, err := store.PublicSnapshot()
	if err != nil {
		t.Fatalf("PublicSnapshot: %v", err)
	}
	if snapshot.ChatPath != "/env/chat" {
		t.Fatalf("unexpected chat_path: got %q want %q", snapshot.ChatPath, "/env/chat")
	}
}

func TestPublicSnapshotKeepsExplicitFileChatPathEvenWhenMatchingDefault(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")

	store, err := newStoreFromEnv()
	if err != nil {
		t.Fatalf("newStoreFromEnv: %v", err)
	}

	defaultPath := "/chat/completions"
	if err := store.Update(configUpdateRequest{ChatPath: &defaultPath}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	snapshot, err := store.PublicSnapshot()
	if err != nil {
		t.Fatalf("PublicSnapshot: %v", err)
	}
	if snapshot.ChatPath != defaultPath {
		t.Fatalf("unexpected chat_path: got %q want %q", snapshot.ChatPath, defaultPath)
	}
}
