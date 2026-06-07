package prompts

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

func stringPointer(value string) *string {
	return &value
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
