package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
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
	ghostPromptLibrary := []SystemPromptLibraryItem{
		{
			ID:          "core-job",
			Name:        "Core Job",
			InsertPoint: SystemPromptInsertPointCoreJob,
			Content:     "from ghost",
			Active:      true,
		},
	}
	ghostLibraryRaw, err := marshalPromptLibrary(ghostPromptLibrary)
	if err != nil {
		t.Fatalf("marshalPromptLibrary: %v", err)
	}
	ghostLibraryPath := filepath.Join(ghostRoot, systemPromptPromptLibraryKey+systemPromptJSONExt)
	if err := os.WriteFile(ghostLibraryPath, []byte(ghostLibraryRaw), systemPromptFilePerm); err != nil {
		t.Fatalf("WriteFile(%s): %v", ghostLibraryPath, err)
	}
	if err := os.Chtimes(ghostLibraryPath, newer, newer); err != nil {
		t.Fatalf("Chtimes(%s): %v", ghostLibraryPath, err)
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

func TestLoadSystemPromptFilesRemovesLegacyPromptFiles(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")
	root := filepath.Join(promptsDir, systemPromptDirName)
	if err := os.MkdirAll(root, systemPromptDirPerm); err != nil {
		t.Fatalf("MkdirAll(%s): %v", root, err)
	}
	for _, key := range append(systemPromptFileKeys(), legacySystemPromptFileKeys()...) {
		if err := os.WriteFile(filepath.Join(root, key+systemPromptFileExt), nil, systemPromptFilePerm); err != nil {
			t.Fatalf("WriteFile(%s): %v", key, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, systemPromptInitFile), []byte("initialized"), systemPromptFilePerm); err != nil {
		t.Fatalf("WriteFile(init marker): %v", err)
	}

	files, err := LoadSystemPromptFiles(promptsDir)
	if err != nil {
		t.Fatalf("LoadSystemPromptFiles: %v", err)
	}
	if files.CorePrompt != "" {
		t.Fatalf("expected empty core prompt after migration, got %+v", files)
	}
	if len(files.PromptLibrary) != 0 {
		t.Fatalf("expected empty prompt library after migration, got %+v", files.PromptLibrary)
	}
	for _, key := range legacySystemPromptFileKeys() {
		path := filepath.Join(root, key+systemPromptFileExt)
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected legacy prompt file removed: %s", path)
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
