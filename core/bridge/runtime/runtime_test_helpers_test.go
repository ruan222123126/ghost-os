package runtime

import (
	"path/filepath"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
)

func newRuntimeTestStore(t *testing.T) *ConfigStore {
	t.Helper()

	store, err := bridgeconfig.NewStoreFromEnv()
	if err != nil {
		t.Fatalf("new config store: %v", err)
	}
	return WrapConfigStore(store)
}

func setupRuntimeFactoryTestEnv(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	t.Setenv("GHOST_CONFIG_PATH", filepath.Join(tempDir, "config.toml"))
	t.Setenv("GHOST_API_KEY", "test-key")
	t.Setenv("GHOST_ARTIFACTS_PATH", filepath.Join(tempDir, "artifacts"))
	t.Setenv("GHOST_PROMPTS_DIR", filepath.Join(tempDir, "prompts"))
	return tempDir
}

func containsToolName(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
}
