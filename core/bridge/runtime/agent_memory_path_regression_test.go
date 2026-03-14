package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"ghost-os/bridge/memorystore"
)

func TestAgentRuntimeFactoryUsesMemoryPathFromEnv(t *testing.T) {
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	memoryPath := filepath.Join(tempDir, "override-memory", "memory.db")

	t.Setenv("GHOST_CONFIG_PATH", filepath.Join(tempDir, "config.toml"))
	t.Setenv("GHOST_API_KEY", "test-key")
	t.Setenv("GHOST_RSS_FEEDS_PATH", filepath.Join(tempDir, "rss", "feeds.json"))
	t.Setenv("HOME", homeDir)
	t.Setenv("GHOST_MEMORY_PATH", memoryPath)
	makeReadOnlyRuntimeMemoryDB(t, filepath.Join(homeDir, ".ghost-os", "memory", "memory.db"))

	store := newRuntimeTestStore(t)
	deps, err := newAgentRuntimeFactory().Build(store)
	if err != nil {
		t.Fatalf("build runtime deps: %v", err)
	}
	t.Cleanup(deps.Close)

	if _, err := os.Stat(memoryPath); err != nil {
		t.Fatalf("expected runtime to initialize memory database at override path: %v", err)
	}
}

func makeReadOnlyRuntimeMemoryDB(t *testing.T, path string) {
	t.Helper()

	store, err := memorystore.NewStore(path)
	if err != nil {
		t.Fatalf("create default memory database: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close default memory database: %v", err)
	}

	dir := filepath.Dir(path)
	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatalf("chmod memory database: %v", err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod memory directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(dir, 0o700)
		_ = os.Chmod(path, 0o600)
	})
}
