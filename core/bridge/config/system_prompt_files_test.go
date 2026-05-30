package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadSystemPromptFilesCreatesDefaultFiles(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")

	files, err := LoadSystemPromptFiles(promptsDir)
	if err != nil {
		t.Fatalf("LoadSystemPromptFiles: %v", err)
	}
	if files.CorePrompt != "" {
		t.Fatalf("expected core prompt file to start empty, got %+v", files)
	}
	if len(files.PromptLibrary) != 0 {
		t.Fatalf("expected prompt library to start empty, got %+v", files.PromptLibrary)
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
		CorePrompt: stringPointer("core block"),
	})
	if err != nil {
		t.Fatalf("UpdateSystemPromptFiles: %v", err)
	}
	if updated.CorePrompt != "core block" {
		t.Fatalf("unexpected updated prompts: %+v", updated)
	}
	assertSingleCoreJobCard(t, updated.PromptLibrary, "core block")

	reloaded, err := LoadSystemPromptFiles(promptsDir)
	if err != nil {
		t.Fatalf("LoadSystemPromptFiles after update: %v", err)
	}
	if !reflect.DeepEqual(reloaded, updated) {
		t.Fatalf("unexpected reloaded prompts: got %+v want %+v", reloaded, updated)
	}

	_, err = UpdateSystemPromptFiles(promptsDir, SystemPromptUpdateRequest{})
	if !errors.Is(err, errSystemPromptUpdateEmpty) {
		t.Fatalf("expected empty update error, got %v", err)
	}
}

func TestMigrateLegacyPromptsDirMovesGhostRootIntoCanonicalDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configPath := filepath.Join(home, ghostOSDirName, "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	ghostPromptsDir := filepath.Join(home, ghostDirName, promptsDirName)
	if err := writeBridgeFileConfig(configPath, bridgeFileConfig{
		PromptsDir: stringPointer(ghostPromptsDir),
	}); err != nil {
		t.Fatalf("writeBridgeFileConfig: %v", err)
	}
	ghostRoot := filepath.Join(ghostPromptsDir, systemPromptDirName)
	if err := os.MkdirAll(ghostRoot, systemPromptDirPerm); err != nil {
		t.Fatalf("MkdirAll(%s): %v", ghostRoot, err)
	}
	if err := os.WriteFile(
		filepath.Join(ghostRoot, systemPromptCorePromptKey+systemPromptFileExt),
		[]byte("from ghost"),
		systemPromptFilePerm,
	); err != nil {
		t.Fatalf("WriteFile(core_prompt): %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(ghostRoot, systemPromptPromptLibraryKey+systemPromptJSONExt),
		[]byte(`[{"id":"core-job","name":"Core Job","insert_point":"core_job","content":"from ghost","active":true}]`),
		systemPromptFilePerm,
	); err != nil {
		t.Fatalf("WriteFile(prompt_library): %v", err)
	}

	report, err := MigrateLegacyPromptsDir()
	if err != nil {
		t.Fatalf("MigrateLegacyPromptsDir: %v", err)
	}
	if report.BackupDir == "" {
		t.Fatal("expected legacy prompts backup dir")
	}
	if !report.ConfigUpdated {
		t.Fatal("expected config prompts_dir rewrite")
	}

	primaryPromptsDir := filepath.Join(home, ghostOSDirName, promptsDirName)
	primaryRoot := filepath.Join(primaryPromptsDir, systemPromptDirName)
	assertSystemPromptFile(t, primaryRoot, systemPromptCorePromptKey, "from ghost")
	if _, err := os.Stat(report.BackupDir); err != nil {
		t.Fatalf("expected backup dir %s: %v", report.BackupDir, err)
	}
}

func TestMigrateLegacySystemPromptsArchivesLegacyPromptFiles(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")
	root := filepath.Join(promptsDir, systemPromptDirName)
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROMPTS_DIR", promptsDir)
	if err := writeBridgeFileConfig(configPath, bridgeFileConfig{
		PromptsDir: stringPointer(promptsDir),
	}); err != nil {
		t.Fatalf("writeBridgeFileConfig: %v", err)
	}
	if err := os.MkdirAll(root, systemPromptDirPerm); err != nil {
		t.Fatalf("MkdirAll(%s): %v", root, err)
	}
	for _, key := range systemPromptFileKeys() {
		fileName, ok := systemPromptFileName(key)
		if !ok {
			t.Fatalf("systemPromptFileName(%s): missing", key)
		}
		if err := os.WriteFile(filepath.Join(root, fileName), nil, systemPromptFilePerm); err != nil {
			t.Fatalf("WriteFile(%s): %v", key, err)
		}
	}
	for _, key := range legacySystemPromptFileKeys() {
		if err := os.WriteFile(filepath.Join(root, key+systemPromptFileExt), nil, systemPromptFilePerm); err != nil {
			t.Fatalf("WriteFile(%s): %v", key, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, systemPromptInitFile), []byte("initialized"), systemPromptFilePerm); err != nil {
		t.Fatalf("WriteFile(init marker): %v", err)
	}
	if _, err := ensurePresetRoot(promptsDir); err != nil {
		t.Fatalf("ensurePresetRoot: %v", err)
	}

	report, err := MigrateLegacySystemPrompts()
	if err != nil {
		t.Fatalf("MigrateLegacySystemPrompts: %v", err)
	}
	if len(report.ArchivedFiles) != len(legacySystemPromptFileKeys()) {
		t.Fatalf("unexpected archived files: %+v", report.ArchivedFiles)
	}
	for _, key := range legacySystemPromptFileKeys() {
		path := filepath.Join(root, key+systemPromptFileExt)
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected legacy prompt file archived: %s", path)
		}
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
