package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreUpdateToolPersistsAllowlistAndPromptOverride(t *testing.T) {
	configPath, store := newToolStoreForTest(t)
	mustSeedToolAllowlist(t, configPath, "script_exec")
	basePrompt, ok := ToolBasePrompt("script_exec")
	if !ok {
		t.Fatal("missing base prompt for script_exec")
	}
	assertToolState(t, store, "script_exec", true, basePrompt)

	disableToolWithPrompt(t, store, "script_exec", "  use script_exec for deterministic local automation  ")
	assertToolState(t, store, "script_exec", false, "use script_exec for deterministic local automation")
	assertPersistedToolConfig(t, "script_exec", false, true, "use script_exec for deterministic local automation")

	enableToolAndClearPrompt(t, store, "script_exec")
	assertToolState(t, store, "script_exec", true, "")
	assertPersistedToolConfig(t, "script_exec", true, false, "")

	if configPath == "" {
		t.Fatal("config path must not be empty")
	}
}

func TestStoreUpdateToolValidatesInput(t *testing.T) {
	_, store := newToolStoreForTest(t)

	disabled := false
	if err := store.UpdateTool(ToolUpdateRequest{Name: " ", Enabled: &disabled}); !errors.Is(err, errToolNameRequired) {
		t.Fatalf("expected errToolNameRequired, got %v", err)
	}
	if err := store.UpdateTool(ToolUpdateRequest{Name: "not_exists", Enabled: &disabled}); !errors.Is(err, errToolNotFound) {
		t.Fatalf("expected errToolNotFound, got %v", err)
	}
	if err := store.UpdateTool(ToolUpdateRequest{Name: "script_exec"}); !errors.Is(err, errToolUpdateEmpty) {
		t.Fatalf("expected errToolUpdateEmpty, got %v", err)
	}
}

func TestStoreListToolsUsesAllowlistAsEnabledSource(t *testing.T) {
	configPath, store := newToolStoreForTest(t)
	basePrompt, ok := ToolBasePrompt("script_exec")
	if !ok {
		t.Fatal("missing base prompt for script_exec")
	}
	fileCfg, err := normalizeBridgeFileConfigForWrite(bridgeFileConfig{
		ToolAllowlist: []string{"script_exec"},
		ToolBlocklist: []string{"script_exec"},
	})
	if err != nil {
		t.Fatalf("normalizeBridgeFileConfigForWrite: %v", err)
	}
	if err := writeBridgeFileConfig(configPath, fileCfg); err != nil {
		t.Fatalf("writeBridgeFileConfig(%s): %v", configPath, err)
	}

	assertToolState(t, store, "script_exec", true, basePrompt)
}

func findToolRecord(items []ToolRecord, name string) (ToolRecord, bool) {
	for _, item := range items {
		if item.Name == name {
			return item, true
		}
	}
	return ToolRecord{}, false
}

func hasToolName(items []string, name string) bool {
	for _, item := range items {
		if item == name {
			return true
		}
	}
	return false
}

func newToolStoreForTest(t *testing.T) (string, *store) {
	t.Helper()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROMPTS_DIR", filepath.Join(tempDir, "prompts"))
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")

	store, err := newStoreFromEnv()
	if err != nil {
		t.Fatalf("newStoreFromEnv: %v", err)
	}
	return configPath, store
}

func mustSeedToolAllowlist(t *testing.T, configPath string, names ...string) {
	t.Helper()

	fileCfg, err := normalizeBridgeFileConfigForWrite(bridgeFileConfig{
		ToolAllowlist: names,
	})
	if err != nil {
		t.Fatalf("normalizeBridgeFileConfigForWrite: %v", err)
	}
	if err := writeBridgeFileConfig(configPath, fileCfg); err != nil {
		t.Fatalf("writeBridgeFileConfig(%s): %v", configPath, err)
	}
}

func mustLoadToolFileConfig(t *testing.T) bridgeFileConfig {
	t.Helper()

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	return fileCfg
}

func assertToolState(t *testing.T, store *store, name string, enabled bool, promptOverride string) {
	t.Helper()

	items, err := store.ListTools()
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	record, ok := findToolRecord(items, name)
	if !ok {
		t.Fatalf("expected %s in tool list", name)
	}
	if record.Enabled != enabled {
		t.Fatalf("unexpected enabled state for %s: got %v want %v", name, record.Enabled, enabled)
	}
	if record.PromptOverride != promptOverride {
		t.Fatalf("unexpected prompt override for %s: got %q want %q", name, record.PromptOverride, promptOverride)
	}
}

func disableToolWithPrompt(t *testing.T, store *store, name string, prompt string) {
	t.Helper()

	enabled := false
	if err := store.UpdateTool(ToolUpdateRequest{
		Name:           name,
		Enabled:        &enabled,
		PromptOverride: &prompt,
	}); err != nil {
		t.Fatalf("UpdateTool disable+prompt: %v", err)
	}
}

func enableToolAndClearPrompt(t *testing.T, store *store, name string) {
	t.Helper()

	enabled := true
	clear := ""
	if err := store.UpdateTool(ToolUpdateRequest{
		Name:           name,
		Enabled:        &enabled,
		PromptOverride: &clear,
	}); err != nil {
		t.Fatalf("UpdateTool enable+clear: %v", err)
	}
}

func assertPersistedToolConfig(t *testing.T, name string, allowed bool, blocked bool, promptOverride string) {
	t.Helper()

	fileCfg := mustLoadToolFileConfig(t)
	if hasToolName(fileCfg.ToolAllowlist, name) != allowed {
		t.Fatalf("unexpected persisted allowlist state for %s", name)
	}
	if hasToolName(fileCfg.ToolBlocklist, name) != blocked {
		t.Fatalf("unexpected persisted blocked state for %s", name)
	}
	if len(fileCfg.ToolPromptOverrides) != 0 {
		t.Fatalf("expected legacy tool_prompt_overrides to stay empty, got %+v", fileCfg.ToolPromptOverrides)
	}

	path := filepath.Join(strings.TrimSpace(os.Getenv("GHOST_PROMPTS_DIR")), toolPromptDirName, name+toolPromptFileExt)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	if got := strings.TrimSpace(string(raw)); got != promptOverride {
		t.Fatalf("unexpected prompt file content for %s: got %q want %q", name, got, promptOverride)
	}
}
