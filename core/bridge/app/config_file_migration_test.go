package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
)

func TestLoadBridgeFileConfigMigratesLegacyProviderLayoutToNamedTables(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	legacyConfig := []byte(`model_provider = "custom"
model = "qwen-coder"
memory_decision_enabled = true
memory_decision_capture_on_turn = false
memory_decision_path = "~/.ghost-os/memory/decision"
memory_decision_max_hits = 5
memory_decision_min_confidence = 0.76
memory_decision_min_reuse_score = 0.71
memory_decision_recipe_enabled = false
memory_decision_recipe_interval = "8h"
memory_decision_recipe_min_support = 4
memory_decision_debug_enabled = true
memory_decision_selector_hint_enabled = false

[[model_providers]]
name = "custom"
base_url = "http://localhost:11434/v1"
api_key = "legacy-key"
`)
	if err := os.WriteFile(configPath, legacyConfig, 0o600); err != nil {
		t.Fatalf("write legacy config file: %v", err)
	}

	loadedPath, migrated, err := migrateBridgeFileConfigIfLegacy()
	if err != nil {
		t.Fatalf("migrateBridgeFileConfigIfLegacy: %v", err)
	}
	if loadedPath != configPath {
		t.Fatalf("unexpected config path: got %q want %q", loadedPath, configPath)
	}
	if !migrated {
		t.Fatalf("expected migration to run")
	}

	cfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if cfg.ActiveProvider == nil || *cfg.ActiveProvider != "custom" {
		t.Fatalf("unexpected active provider: %#v", cfg.ActiveProvider)
	}
	providers := normalizeProviderConfigs(cfg.Providers, stringValue(cfg.Model))
	if len(providers) != 1 {
		t.Fatalf("unexpected provider count: got %d want 1", len(providers))
	}
	if providers[0].Type != llm.ProviderCustom {
		t.Fatalf("unexpected provider type: got %q want %q", providers[0].Type, llm.ProviderCustom)
	}
	if providers[0].BaseURL != "http://localhost:11434/v1" {
		t.Fatalf("unexpected migrated base url: got %q", providers[0].BaseURL)
	}
	if cfg.MemoryDecisionRecipeInterval == nil || *cfg.MemoryDecisionRecipeInterval != "8h" {
		t.Fatalf("unexpected migrated decision recipe interval: %#v", cfg.MemoryDecisionRecipeInterval)
	}
	if cfg.MemoryDecisionCaptureOnTurn == nil || *cfg.MemoryDecisionCaptureOnTurn {
		t.Fatalf("unexpected migrated decision capture toggle: %#v", cfg.MemoryDecisionCaptureOnTurn)
	}
	if cfg.MemoryDecisionSelectorHintEnabled == nil || *cfg.MemoryDecisionSelectorHintEnabled {
		t.Fatalf("unexpected migrated selector hint toggle: %#v", cfg.MemoryDecisionSelectorHintEnabled)
	}
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read migrated config file: %v", err)
	}
	contents := string(raw)
	if strings.Contains(contents, "[[model_providers]]") || strings.Contains(contents, "model_provider") {
		t.Fatalf("expected legacy provider layout to be removed, got %s", contents)
	}
	if !strings.Contains(contents, "[providers.custom]") {
		t.Fatalf("expected named provider table, got %s", contents)
	}
}

func TestLoadBridgeFileConfigDoesNotWriteBackLegacyProviderLayout(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	legacyConfig := []byte(`model_provider = "custom"
model = "qwen-coder"

[[model_providers]]
name = "custom"
base_url = "http://localhost:11434/v1"
api_key = "legacy-key"
`)
	if err := os.WriteFile(configPath, legacyConfig, 0o600); err != nil {
		t.Fatalf("write legacy config file: %v", err)
	}

	if _, _, err := loadBridgeFileConfig(); err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}
	contents := string(raw)
	if !strings.Contains(contents, "[[model_providers]]") || !strings.Contains(contents, "model_provider") {
		t.Fatalf("expected legacy provider layout to remain until explicitly migrated, got %s", contents)
	}
	if strings.Contains(contents, "[providers.custom]") {
		t.Fatalf("expected named provider table to not be written implicitly, got %s", contents)
	}
}
