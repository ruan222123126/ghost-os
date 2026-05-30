package transport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunServeLegacyPreflightRejectsLegacyToolPromptOverrides(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROMPTS_DIR", filepath.Join(tempDir, "prompts"))
	t.Setenv("GHOST_SESSIONS_PATH", filepath.Join(tempDir, "sessions"))
	t.Setenv("GHOST_TASKS_PATH", filepath.Join(tempDir, "tasks"))

	if err := os.WriteFile(configPath, []byte("tool_prompt_overrides = { script_exec = \"legacy\" }\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(config): %v", err)
	}

	err := runServeLegacyPreflight()
	if err == nil || !strings.Contains(err.Error(), "bin/ghost-bridge migrate tool-prompts") {
		t.Fatalf("expected tool-prompts migration error, got %v", err)
	}
}

func TestRunServeLegacyPreflightRejectsLegacySessionJSON(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")
	sessionsPath := filepath.Join(tempDir, "sessions")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROMPTS_DIR", filepath.Join(tempDir, "prompts"))
	t.Setenv("GHOST_TASKS_PATH", filepath.Join(tempDir, "tasks"))

	if err := os.WriteFile(configPath, []byte("sessions_path = \""+sessionsPath+"\"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(config): %v", err)
	}
	if err := os.MkdirAll(sessionsPath, 0o700); err != nil {
		t.Fatalf("MkdirAll(sessions): %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionsPath, "legacy-session.json"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("WriteFile(legacy session): %v", err)
	}

	err := runServeLegacyPreflight()
	if err == nil || !strings.Contains(err.Error(), "bin/ghost-bridge migrate sessions") {
		t.Fatalf("expected sessions migration error, got %v", err)
	}
}
