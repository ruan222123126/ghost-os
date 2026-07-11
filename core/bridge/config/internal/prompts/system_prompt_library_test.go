package prompts

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestUpdateSystemPromptFilesFromPromptLibraryCompilesCorePrompt(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")

	library := []SystemPromptLibraryItem{
		{
			ID:          "draft",
			Name:        "Draft Card",
			InsertPoint: SystemPromptInsertPointCoreJob,
			Content:     "draft core",
			Active:      false,
		},
		{
			ID:          "active",
			Name:        "Active Card",
			InsertPoint: SystemPromptInsertPointCoreJob,
			Content:     "active core",
			Active:      true,
		},
	}
	updated, err := UpdateSystemPromptFiles(promptsDir, SystemPromptUpdateRequest{
		PromptLibrary: &library,
	})
	if err != nil {
		t.Fatalf("UpdateSystemPromptFiles: %v", err)
	}
	if updated.CorePrompt != "active core" {
		t.Fatalf("expected compiled core prompt from active card, got %+v", updated)
	}
	if !reflect.DeepEqual(updated.PromptLibrary, library) {
		t.Fatalf("expected prompt library unchanged, got %+v", updated.PromptLibrary)
	}
}

func TestUpdateSystemPromptFilesRejectsMultipleActiveCardsInOneInsertPoint(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")

	library := []SystemPromptLibraryItem{
		{
			ID:          "card-a",
			Name:        "A",
			InsertPoint: SystemPromptInsertPointCoreJob,
			Content:     "core a",
			Active:      true,
		},
		{
			ID:          "card-b",
			Name:        "B",
			InsertPoint: SystemPromptInsertPointCoreJob,
			Content:     "core b",
			Active:      true,
		},
	}
	_, err := UpdateSystemPromptFiles(promptsDir, SystemPromptUpdateRequest{
		PromptLibrary: &library,
	})
	if !errors.Is(err, errSystemPromptLibraryInvalid) {
		t.Fatalf("expected prompt library validation error, got %v", err)
	}
}

func TestUpdateSystemPromptFilesAllowsMemoryInsertPoint(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")
	library := []SystemPromptLibraryItem{
		{
			ID:          "rule-card",
			Name:        "Rule",
			InsertPoint: SystemPromptInsertPointRule,
			Content:     "custom rule",
			Active:      true,
		},
		{
			ID:          "core-job",
			Name:        "Core Job",
			InsertPoint: SystemPromptInsertPointCoreJob,
			Content:     "core guidance",
			Active:      true,
		},
		{
			ID:          "memory-card",
			Name:        "Memory",
			InsertPoint: SystemPromptInsertPointMemory,
			Content:     "memory guidance",
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

func TestUpdateSystemPromptFilesAllowsRuleInsertPointOnly(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")
	library := []SystemPromptLibraryItem{
		{
			ID:          "rule-card",
			Name:        "Rule",
			InsertPoint: SystemPromptInsertPointRule,
			Content:     "custom rule",
			Active:      true,
		},
	}
	updated, err := UpdateSystemPromptFiles(promptsDir, SystemPromptUpdateRequest{
		PromptLibrary: &library,
	})
	if err != nil {
		t.Fatalf("UpdateSystemPromptFiles: %v", err)
	}
	if updated.CorePrompt != "" {
		t.Fatalf("expected rule-only library to keep core prompt empty, got %+v", updated)
	}
	if !reflect.DeepEqual(updated.PromptLibrary, library) {
		t.Fatalf("expected prompt library unchanged, got %+v", updated.PromptLibrary)
	}
}

func TestUpdateSystemPromptFilesWithCorePromptRebuildsPromptLibrary(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")

	initialLibrary := []SystemPromptLibraryItem{
		{
			ID:          "card-a",
			Name:        "Card A",
			InsertPoint: SystemPromptInsertPointCoreJob,
			Content:     "core a",
			Active:      true,
		},
	}
	if _, err := UpdateSystemPromptFiles(promptsDir, SystemPromptUpdateRequest{
		PromptLibrary: &initialLibrary,
	}); err != nil {
		t.Fatalf("UpdateSystemPromptFiles initial prompt library: %v", err)
	}

	updated, err := UpdateSystemPromptFiles(promptsDir, SystemPromptUpdateRequest{
		CorePrompt: stringPointer("patched core"),
	})
	if err != nil {
		t.Fatalf("UpdateSystemPromptFiles core prompt: %v", err)
	}
	if updated.CorePrompt != "patched core" {
		t.Fatalf("expected updated core prompt, got %+v", updated)
	}
	assertSingleCoreJobCard(t, updated.PromptLibrary, "patched core")
}

func assertSingleCoreJobCard(t *testing.T, library []SystemPromptLibraryItem, content string) {
	t.Helper()

	if len(library) != 1 {
		t.Fatalf("expected one prompt library card, got %+v", library)
	}
	card := library[0]
	if card.ID == "" {
		t.Fatalf("expected non-empty prompt library id, got %+v", card)
	}
	if card.Name == "" {
		t.Fatalf("expected non-empty prompt library name, got %+v", card)
	}
	if card.InsertPoint != SystemPromptInsertPointCoreJob {
		t.Fatalf("expected core_job insert point, got %+v", card)
	}
	if !card.Active {
		t.Fatalf("expected active prompt library card, got %+v", card)
	}
	if card.Content != content {
		t.Fatalf("expected prompt library content %q, got %+v", content, card)
	}
}
