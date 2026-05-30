package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateLegacyToolPromptsMovesLegacyOverridesIntoPromptFiles(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")
	promptsDir := filepath.Join(tempDir, "prompts")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROMPTS_DIR", promptsDir)
	if err := writeBridgeFileConfig(configPath, bridgeFileConfig{
		ToolPromptOverrides: map[string]string{
			"script_exec": "  legacy script prompt  ",
		},
	}); err != nil {
		t.Fatalf("writeBridgeFileConfig: %v", err)
	}

	report, err := MigrateLegacyToolPrompts()
	if err != nil {
		t.Fatalf("MigrateLegacyToolPrompts: %v", err)
	}
	if !report.ConfigUpdated {
		t.Fatal("expected config to be updated")
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if len(fileCfg.ToolPromptOverrides) != 0 {
		t.Fatalf("expected legacy tool_prompt_overrides to be cleared, got %+v", fileCfg.ToolPromptOverrides)
	}

	path := filepath.Join(promptsDir, toolPromptDirName, "script_exec"+toolPromptFileExt)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	if got := strings.TrimSpace(string(raw)); got != "legacy script prompt" {
		t.Fatalf("unexpected migrated prompt: got %q want %q", got, "legacy script prompt")
	}
}

func TestMigrateLegacyToolPromptsKeepsExistingCustomPromptFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")
	promptsDir := filepath.Join(tempDir, "prompts")
	root := filepath.Join(promptsDir, toolPromptDirName)
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROMPTS_DIR", promptsDir)
	if err := os.MkdirAll(root, toolPromptDirPerm); err != nil {
		t.Fatalf("MkdirAll(%s): %v", root, err)
	}
	path := filepath.Join(root, "script_exec"+toolPromptFileExt)
	if err := os.WriteFile(path, []byte("existing file prompt"), toolPromptFilePerm); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
	if err := writeBridgeFileConfig(configPath, bridgeFileConfig{
		ToolPromptOverrides: map[string]string{
			"script_exec": "legacy prompt should not overwrite",
		},
	}); err != nil {
		t.Fatalf("writeBridgeFileConfig: %v", err)
	}

	report, err := MigrateLegacyToolPrompts()
	if err != nil {
		t.Fatalf("MigrateLegacyToolPrompts: %v", err)
	}
	if len(report.SkippedTools) != 1 || report.SkippedTools[0] != "script_exec" {
		t.Fatalf("unexpected skipped tools: %+v", report.SkippedTools)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	if got := strings.TrimSpace(string(raw)); got != "existing file prompt" {
		t.Fatalf("expected existing prompt file to win, got %q", got)
	}
}
