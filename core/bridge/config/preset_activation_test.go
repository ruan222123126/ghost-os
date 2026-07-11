package config

import (
	"os"
	"reflect"
	"testing"
)

func TestApplyPresetToPromptLibraryReordersActiveContextCards(t *testing.T) {
	preset := Preset{
		Name:          "Research",
		ToolAllowlist: []string{"script_exec"},
		PromptRefs: PresetPromptRefs{
			Rule:    "rule-card",
			CoreJob: "core-card",
			Context: []string{"context-b", "context-a"},
		},
	}

	library, err := applyPresetToPromptLibrary(presetActivationPromptLibraryFixture(), preset)
	if err != nil {
		t.Fatalf("applyPresetToPromptLibrary: %v", err)
	}

	if got := activePromptIDsFor(t, library, SystemPromptInsertPointContext); !reflect.DeepEqual(got, []string{"context-b", "context-a"}) {
		t.Fatalf("unexpected active context ids: got %v want %v", got, []string{"context-b", "context-a"})
	}
	if got := promptIDsFor(t, library, SystemPromptInsertPointContext); !reflect.DeepEqual(got, []string{"context-b", "context-a", "context-c"}) {
		t.Fatalf("unexpected saved context order: got %v want %v", got, []string{"context-b", "context-a", "context-c"})
	}
}

func TestStoreApplyPresetPersistsStrictToolAllowlistAndPromptRefs(t *testing.T) {
	configPath, store := newToolStoreForTest(t)
	promptsDir := os.Getenv("GHOST_PROMPTS_DIR")
	seedPromptLibraryForPresetTest(t, promptsDir)
	mustSeedToolAllowlist(t, configPath, "sfind")

	preset, err := store.CreatePreset(PresetCreateRequest{
		Name:          "Research",
		ToolAllowlist: []string{"web_search", "script_exec"},
		PromptRefs: PresetPromptRefs{
			Rule:    "rule-card",
			CoreJob: "core-card",
			Context: []string{"context-b", "context-a"},
		},
	})
	if err != nil {
		t.Fatalf("CreatePreset: %v", err)
	}

	applied, err := store.ApplyPreset(preset.ID)
	if err != nil {
		t.Fatalf("ApplyPreset: %v", err)
	}
	if applied.ID != preset.ID {
		t.Fatalf("unexpected applied preset: got %+v want %+v", applied, preset)
	}

	fileCfg := mustLoadToolFileConfig(t)
	if fileCfg.ToolAllowlistOnly == nil || !*fileCfg.ToolAllowlistOnly {
		t.Fatalf("expected tool_allowlist_only to be enabled, got %+v", fileCfg.ToolAllowlistOnly)
	}
	if !reflect.DeepEqual(fileCfg.ToolAllowlist, []string{"script_exec", "web_search"}) {
		t.Fatalf("unexpected persisted allowlist: got %v", fileCfg.ToolAllowlist)
	}
	expectedBlocklist := normalizeConfiguredToolNames([]string{
		"list_files",
		"read_file",
		"search_files",
		"write_file",
		"apply_diff",
		"bash_exec",
		"codex_cli",
		"screen_control",
		"sfind",
		"ask_human",
	})
	if !reflect.DeepEqual(fileCfg.ToolBlocklist, expectedBlocklist) {
		t.Fatalf("unexpected persisted blocklist: got %v want %v", fileCfg.ToolBlocklist, expectedBlocklist)
	}

	files, err := store.SystemPrompts()
	if err != nil {
		t.Fatalf("SystemPrompts: %v", err)
	}
	if got := activePromptIDsFor(t, files.PromptLibrary, SystemPromptInsertPointContext); !reflect.DeepEqual(got, []string{"context-b", "context-a"}) {
		t.Fatalf("unexpected active context ids after apply: got %v want %v", got, []string{"context-b", "context-a"})
	}
}

func presetActivationPromptLibraryFixture() []SystemPromptLibraryItem {
	return []SystemPromptLibraryItem{
		{ID: "rule-card", Name: "Rule", InsertPoint: SystemPromptInsertPointRule, Content: "rule", Active: false},
		{ID: "core-card", Name: "Core", InsertPoint: SystemPromptInsertPointCoreJob, Content: "core", Active: true},
		{ID: "memory-card", Name: "Memory", InsertPoint: SystemPromptInsertPointMemory, Content: "memory", Active: true},
		{ID: "context-a", Name: "Context A", InsertPoint: SystemPromptInsertPointContext, Content: "context a", Active: true},
		{ID: "context-b", Name: "Context B", InsertPoint: SystemPromptInsertPointContext, Content: "context b", Active: false},
		{ID: "context-c", Name: "Context C", InsertPoint: SystemPromptInsertPointContext, Content: "context c", Active: true},
	}
}

func seedPromptLibraryForPresetTest(t *testing.T, promptsDir string) {
	t.Helper()

	library := presetActivationPromptLibraryFixture()
	if _, err := UpdateSystemPromptFiles(promptsDir, SystemPromptUpdateRequest{
		PromptLibrary: &library,
	}); err != nil {
		t.Fatalf("UpdateSystemPromptFiles: %v", err)
	}
}

func promptIDsFor(
	t *testing.T,
	library []SystemPromptLibraryItem,
	insertPoint SystemPromptInsertPoint,
) []string {
	t.Helper()

	ids := make([]string, 0, len(library))
	for _, item := range library {
		if item.InsertPoint == insertPoint {
			ids = append(ids, item.ID)
		}
	}
	return ids
}

func activePromptIDsFor(
	t *testing.T,
	library []SystemPromptLibraryItem,
	insertPoint SystemPromptInsertPoint,
) []string {
	t.Helper()

	ids := make([]string, 0, len(library))
	for _, item := range library {
		if item.InsertPoint == insertPoint && item.Active {
			ids = append(ids, item.ID)
		}
	}
	return ids
}
