package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadToolPromptOverridesFromFilesCreatesToolFiles(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")

	overrides, err := loadToolPromptOverridesFromFiles(promptsDir)
	if err != nil {
		t.Fatalf("loadToolPromptOverridesFromFiles: %v", err)
	}
	basePrompt, ok := ToolBasePrompt("script_exec")
	if !ok || strings.TrimSpace(basePrompt) == "" {
		t.Fatal("missing base prompt for script_exec")
	}
	if got := overrides["script_exec"]; got != basePrompt {
		t.Fatalf("unexpected default script_exec prompt: got %q want %q", got, basePrompt)
	}

	root := filepath.Join(promptsDir, toolPromptDirName)
	for _, name := range configuredToolNames() {
		path := filepath.Join(root, name+toolPromptFileExt)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected tool prompt file %s: %v", path, err)
		}
	}
	markerPath := filepath.Join(root, toolPromptInitFile)
	if _, err := os.Stat(markerPath); err != nil {
		t.Fatalf("expected tool prompt init marker %s: %v", markerPath, err)
	}
}

func TestWriteToolPromptOverrideToFileRoundTrip(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")

	if err := writeToolPromptOverrideToFile(promptsDir, "script_exec", "  use script carefully  "); err != nil {
		t.Fatalf("writeToolPromptOverrideToFile: %v", err)
	}
	overrides, err := loadToolPromptOverridesFromFiles(promptsDir)
	if err != nil {
		t.Fatalf("loadToolPromptOverridesFromFiles: %v", err)
	}
	if got := overrides["script_exec"]; got != "use script carefully" {
		t.Fatalf("unexpected prompt override: got %q want %q", got, "use script carefully")
	}

	if err := writeToolPromptOverrideToFile(promptsDir, "script_exec", ""); err != nil {
		t.Fatalf("clear writeToolPromptOverrideToFile: %v", err)
	}
	overrides, err = loadToolPromptOverridesFromFiles(promptsDir)
	if err != nil {
		t.Fatalf("loadToolPromptOverridesFromFiles after clear: %v", err)
	}
	if got := overrides["script_exec"]; got != "" {
		t.Fatalf("expected script_exec to stay empty after clear, got %q", got)
	}
}

func TestLoadToolPromptOverridesPublicFromFiles(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")
	if err := writeToolPromptOverrideToFile(promptsDir, "script_exec", "from file prompt"); err != nil {
		t.Fatalf("writeToolPromptOverrideToFile: %v", err)
	}

	overrides, err := LoadToolPromptOverrides(promptsDir)
	if err != nil {
		t.Fatalf("LoadToolPromptOverrides: %v", err)
	}
	if got := overrides["script_exec"]; got != "from file prompt" {
		t.Fatalf("unexpected script_exec prompt override: got %q want %q", got, "from file prompt")
	}
}

func TestLoadToolPromptOverridesFromFilesBackfillsLegacyEmptyFilesBeforeInit(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")
	root := filepath.Join(promptsDir, toolPromptDirName)
	if err := os.MkdirAll(root, toolPromptDirPerm); err != nil {
		t.Fatalf("MkdirAll(%s): %v", root, err)
	}
	scriptPath := filepath.Join(root, "script_exec"+toolPromptFileExt)
	if err := os.WriteFile(scriptPath, []byte(""), toolPromptFilePerm); err != nil {
		t.Fatalf("WriteFile(%s): %v", scriptPath, err)
	}

	overrides, err := loadToolPromptOverridesFromFiles(promptsDir)
	if err != nil {
		t.Fatalf("loadToolPromptOverridesFromFiles: %v", err)
	}
	basePrompt, ok := ToolBasePrompt("script_exec")
	if !ok {
		t.Fatal("missing base prompt for script_exec")
	}
	if got := overrides["script_exec"]; got != basePrompt {
		t.Fatalf("expected script_exec to be backfilled with base prompt, got %q", got)
	}
}

func TestToolPromptFilesSyncBetweenGhostAndGhostOS(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	primaryPromptsDir := filepath.Join(home, ghostOSDirName, promptsDirName)
	if err := writeToolPromptOverrideToFile(primaryPromptsDir, "script_exec", "from web update"); err != nil {
		t.Fatalf("writeToolPromptOverrideToFile: %v", err)
	}

	assertToolPromptFile(t, primaryPromptsDir, "script_exec", "from web update")
	ghostPromptsDir := filepath.Join(home, ghostDirName, promptsDirName)
	assertToolPromptFile(t, ghostPromptsDir, "script_exec", "from web update")

	ghostPath := filepath.Join(ghostPromptsDir, toolPromptDirName, "script_exec"+toolPromptFileExt)
	if err := os.WriteFile(ghostPath, []byte("from ghost file"), toolPromptFilePerm); err != nil {
		t.Fatalf("WriteFile(%s): %v", ghostPath, err)
	}
	newer := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(ghostPath, newer, newer); err != nil {
		t.Fatalf("Chtimes(%s): %v", ghostPath, err)
	}

	overrides, err := loadToolPromptOverridesFromFiles(primaryPromptsDir)
	if err != nil {
		t.Fatalf("loadToolPromptOverridesFromFiles: %v", err)
	}
	if got := overrides["script_exec"]; got != "from ghost file" {
		t.Fatalf("expected mirrored prompt from .ghost file, got %q", got)
	}

	assertToolPromptFile(t, primaryPromptsDir, "script_exec", "from ghost file")
	assertToolPromptFile(t, ghostPromptsDir, "script_exec", "from ghost file")
}

func TestToolPromptFilesPreferNonDefaultPromptOverNewDefaultMirror(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	primaryPromptsDir := filepath.Join(home, ghostOSDirName, promptsDirName)
	primaryRoot := filepath.Join(primaryPromptsDir, toolPromptDirName)
	if err := os.MkdirAll(primaryRoot, toolPromptDirPerm); err != nil {
		t.Fatalf("MkdirAll(%s): %v", primaryRoot, err)
	}
	primaryScriptPath := filepath.Join(primaryRoot, "script_exec"+toolPromptFileExt)
	if err := os.WriteFile(primaryScriptPath, []byte("persisted custom prompt"), toolPromptFilePerm); err != nil {
		t.Fatalf("WriteFile(%s): %v", primaryScriptPath, err)
	}
	oldTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(primaryScriptPath, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes(%s): %v", primaryScriptPath, err)
	}

	overrides, err := loadToolPromptOverridesFromFiles(primaryPromptsDir)
	if err != nil {
		t.Fatalf("loadToolPromptOverridesFromFiles: %v", err)
	}
	if got := overrides["script_exec"]; got != "persisted custom prompt" {
		t.Fatalf("expected custom prompt to survive mirror initialization, got %q", got)
	}

	assertToolPromptFile(t, primaryPromptsDir, "script_exec", "persisted custom prompt")
	ghostPromptsDir := filepath.Join(home, ghostDirName, promptsDirName)
	assertToolPromptFile(t, ghostPromptsDir, "script_exec", "persisted custom prompt")
}

func assertToolPromptFile(t *testing.T, promptsDir string, name string, want string) {
	t.Helper()

	path := filepath.Join(promptsDir, toolPromptDirName, name+toolPromptFileExt)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	if got := strings.TrimSpace(string(raw)); got != want {
		t.Fatalf("unexpected prompt file %s: got %q want %q", path, got, want)
	}
}
