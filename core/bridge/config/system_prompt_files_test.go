package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadSystemPromptFilesCreatesDefaultFiles(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")

	files, err := LoadSystemPromptFiles(promptsDir)
	if err != nil {
		t.Fatalf("LoadSystemPromptFiles: %v", err)
	}
	if files.GlobalTemplate != defaultSystemPromptGlobalTemplate {
		t.Fatalf("unexpected global template: got %q want %q", files.GlobalTemplate, defaultSystemPromptGlobalTemplate)
	}
	if files.CorePrompt != "" || files.ToolPrompt != "" || files.ToolKeySpec != "" {
		t.Fatalf("expected local prompt files to start empty, got %+v", files)
	}

	root := filepath.Join(promptsDir, systemPromptDirName)
	for _, key := range systemPromptFileKeys() {
		assertSystemPromptFile(t, root, key, defaultSystemPromptFileValues()[key])
	}
	assertSystemPromptMarker(t, root)
}

func TestUpdateSystemPromptFilesRoundTripAndAllowsEmptyStrings(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")
	if _, err := LoadSystemPromptFiles(promptsDir); err != nil {
		t.Fatalf("LoadSystemPromptFiles: %v", err)
	}

	updated, err := UpdateSystemPromptFiles(promptsDir, SystemPromptUpdateRequest{
		GlobalTemplate: stringPointer("{{base_prompt}}\n{{core_prompt}}\n{{tool_prompt}}\n{{tool_key_spec}}"),
		CorePrompt:     stringPointer("core block"),
		ToolPrompt:     stringPointer("tool block"),
		ToolKeySpec:    stringPointer(""),
	})
	if err != nil {
		t.Fatalf("UpdateSystemPromptFiles: %v", err)
	}
	if updated.GlobalTemplate != "{{base_prompt}}\n{{core_prompt}}\n{{tool_prompt}}\n{{tool_key_spec}}" {
		t.Fatalf("unexpected global template: %+v", updated)
	}
	if updated.CorePrompt != "core block" || updated.ToolPrompt != "tool block" || updated.ToolKeySpec != "" {
		t.Fatalf("unexpected updated prompts: %+v", updated)
	}

	reloaded, err := LoadSystemPromptFiles(promptsDir)
	if err != nil {
		t.Fatalf("LoadSystemPromptFiles after update: %v", err)
	}
	if reloaded != updated {
		t.Fatalf("unexpected reloaded prompts: got %+v want %+v", reloaded, updated)
	}

	_, err = UpdateSystemPromptFiles(promptsDir, SystemPromptUpdateRequest{})
	if !errors.Is(err, errSystemPromptUpdateEmpty) {
		t.Fatalf("expected empty update error, got %v", err)
	}
}

func TestSystemPromptFilesSyncBetweenGhostAndGhostOS(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	primaryPromptsDir := filepath.Join(home, ghostOSDirName, promptsDirName)
	if _, err := UpdateSystemPromptFiles(primaryPromptsDir, SystemPromptUpdateRequest{
		CorePrompt: stringPointer("from primary"),
	}); err != nil {
		t.Fatalf("UpdateSystemPromptFiles: %v", err)
	}

	primaryRoot := filepath.Join(primaryPromptsDir, systemPromptDirName)
	ghostPromptsDir := filepath.Join(home, ghostDirName, promptsDirName)
	ghostRoot := filepath.Join(ghostPromptsDir, systemPromptDirName)
	assertSystemPromptFile(t, primaryRoot, systemPromptCorePromptKey, "from primary")
	assertSystemPromptFile(t, ghostRoot, systemPromptCorePromptKey, "from primary")

	ghostPath := filepath.Join(ghostRoot, systemPromptCorePromptKey+systemPromptFileExt)
	if err := os.WriteFile(ghostPath, []byte("from ghost"), systemPromptFilePerm); err != nil {
		t.Fatalf("WriteFile(%s): %v", ghostPath, err)
	}
	newer := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(ghostPath, newer, newer); err != nil {
		t.Fatalf("Chtimes(%s): %v", ghostPath, err)
	}

	reloaded, err := LoadSystemPromptFiles(primaryPromptsDir)
	if err != nil {
		t.Fatalf("LoadSystemPromptFiles: %v", err)
	}
	if reloaded.CorePrompt != "from ghost" {
		t.Fatalf("expected mirrored prompt to win, got %+v", reloaded)
	}

	assertSystemPromptFile(t, primaryRoot, systemPromptCorePromptKey, "from ghost")
	assertSystemPromptFile(t, ghostRoot, systemPromptCorePromptKey, "from ghost")
}

func TestRenderSystemPromptPreservesUnknownPlaceholders(t *testing.T) {
	rendered := RenderSystemPrompt(SystemPromptFiles{
		GlobalTemplate: "{{base_prompt}}\n{{unknown}}\n{{core_prompt}}",
		CorePrompt:     "core",
	}, "base")

	if !strings.Contains(rendered, "base") {
		t.Fatalf("expected base prompt in rendered output, got %q", rendered)
	}
	if !strings.Contains(rendered, "{{unknown}}") {
		t.Fatalf("expected unknown placeholder to remain, got %q", rendered)
	}
	if !strings.Contains(rendered, "core") {
		t.Fatalf("expected core prompt in rendered output, got %q", rendered)
	}
}

func assertSystemPromptFile(t *testing.T, root string, key string, want string) {
	t.Helper()

	path, err := systemPromptFilePath(root, key)
	if err != nil {
		t.Fatalf("systemPromptFilePath(%s): %v", key, err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	if got := strings.TrimSpace(string(raw)); got != want {
		t.Fatalf("unexpected prompt file %s: got %q want %q", path, got, want)
	}
}

func assertSystemPromptMarker(t *testing.T, root string) {
	t.Helper()

	path := filepath.Join(root, systemPromptInitFile)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected init marker %s: %v", path, err)
	}
}
