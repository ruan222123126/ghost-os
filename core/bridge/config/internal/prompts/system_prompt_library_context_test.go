package prompts

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestUpdateSystemPromptFilesAllowsMultipleActiveContextCards(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")
	library := []SystemPromptLibraryItem{
		{
			ID:          "core-card",
			Name:        "Core",
			InsertPoint: SystemPromptInsertPointCoreJob,
			Content:     "core guidance",
			Active:      true,
		},
		{
			ID:          "context-a",
			Name:        "Context A",
			InsertPoint: SystemPromptInsertPointContext,
			Content:     "context alpha",
			Active:      true,
		},
		{
			ID:          "context-b",
			Name:        "Context B",
			InsertPoint: SystemPromptInsertPointContext,
			Content:     "context beta",
			Active:      true,
		},
	}

	updated, err := UpdateSystemPromptFiles(promptsDir, SystemPromptUpdateRequest{
		PromptLibrary: &library,
	})
	if err != nil {
		t.Fatalf("UpdateSystemPromptFiles: %v", err)
	}
	if updated.CorePrompt != "core guidance" {
		t.Fatalf("expected core prompt to compile only from core_job, got %+v", updated)
	}
	if !reflect.DeepEqual(updated.PromptLibrary, library) {
		t.Fatalf("expected prompt library unchanged, got %+v", updated.PromptLibrary)
	}
}

func TestNormalizePromptLibraryRejectsMultipleActiveMemoryCards(t *testing.T) {
	_, err := normalizePromptLibrary([]SystemPromptLibraryItem{
		{
			ID:          "memory-a",
			Name:        "Memory A",
			InsertPoint: SystemPromptInsertPointMemory,
			Content:     "memory a",
			Active:      true,
		},
		{
			ID:          "memory-b",
			Name:        "Memory B",
			InsertPoint: SystemPromptInsertPointMemory,
			Content:     "memory b",
			Active:      true,
		},
	})
	if !errors.Is(err, errSystemPromptLibraryInvalid) {
		t.Fatalf("expected prompt library validation error, got %v", err)
	}
}
