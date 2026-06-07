package presets

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	promptstore "ghost-os/bridge/config/internal/prompts"
)

func TestLoadPresetsCreatesDefaultFile(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")

	presets, err := LoadPresets(promptsDir)
	if err != nil {
		t.Fatalf("LoadPresets: %v", err)
	}
	if len(presets) != 0 {
		t.Fatalf("expected empty preset list, got %+v", presets)
	}

	root := filepath.Join(promptsDir, promptstore.SystemPromptDirName)
	assertPresetFileContent(t, root, []Preset{})
}

func TestLoadPresetsNormalizesStoredValues(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")
	seedPresetPromptLibrary(t, promptsDir)

	root := filepath.Join(promptsDir, promptstore.SystemPromptDirName)
	if err := os.MkdirAll(root, promptstore.SystemPromptDirPerm); err != nil {
		t.Fatalf("MkdirAll(%s): %v", root, err)
	}

	raw := []Preset{
		{
			ID:            " preset-a ",
			Name:          " Default Preset ",
			ToolAllowlist: []string{" web_search ", "script_exec", "web_search", ""},
			PromptRefs: PresetPromptRefs{
				Rule:    " rule-card ",
				CoreJob: " core-card ",
			},
		},
	}
	payload, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, presetFileName), payload, promptstore.SystemPromptFilePerm); err != nil {
		t.Fatalf("WriteFile(%s): %v", presetFileName, err)
	}

	presets, err := LoadPresets(promptsDir)
	if err != nil {
		t.Fatalf("LoadPresets: %v", err)
	}
	want := []Preset{
		{
			ID:            "preset-a",
			Name:          "Default Preset",
			ToolAllowlist: []string{"script_exec", "web_search"},
			PromptRefs: PresetPromptRefs{
				Rule:    "rule-card",
				CoreJob: "core-card",
			},
		},
	}
	if !reflect.DeepEqual(presets, want) {
		t.Fatalf("unexpected presets: got %+v want %+v", presets, want)
	}
	assertPresetFileContent(t, root, want)
}

func TestCreateUpdateDeletePresetRoundTrip(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")
	seedPresetPromptLibrary(t, promptsDir)

	created, err := CreatePreset(promptsDir, PresetCreateRequest{
		Name:          "  Daily  ",
		ToolAllowlist: []string{"web_search", "script_exec", "web_search"},
		PromptRefs: PresetPromptRefs{
			Rule:   "rule-card",
			Memory: "memory-card",
		},
	})
	if err != nil {
		t.Fatalf("CreatePreset: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected non-empty preset id, got %+v", created)
	}
	if !reflect.DeepEqual(created.ToolAllowlist, []string{"script_exec", "web_search"}) {
		t.Fatalf("unexpected created preset: %+v", created)
	}

	updated, err := UpdatePreset(promptsDir, created.ID, PresetUpdateRequest{
		Name:          stringPointer(" Updated "),
		ToolAllowlist: &[]string{"sfind", "script_exec", "sfind"},
		PromptRefs: &PresetPromptRefs{
			CoreJob: "core-card",
		},
	})
	if err != nil {
		t.Fatalf("UpdatePreset: %v", err)
	}
	if updated.Name != "Updated" {
		t.Fatalf("unexpected updated name: %+v", updated)
	}
	if !reflect.DeepEqual(updated.ToolAllowlist, []string{"script_exec", "sfind"}) {
		t.Fatalf("unexpected updated tool allowlist: %+v", updated.ToolAllowlist)
	}
	if updated.PromptRefs.Rule != "" || updated.PromptRefs.Memory != "" || updated.PromptRefs.CoreJob != "core-card" {
		t.Fatalf("unexpected updated prompt refs: %+v", updated.PromptRefs)
	}

	reloaded, err := LoadPresets(promptsDir)
	if err != nil {
		t.Fatalf("LoadPresets: %v", err)
	}
	if !reflect.DeepEqual(reloaded, []Preset{updated}) {
		t.Fatalf("unexpected reloaded presets: got %+v want %+v", reloaded, []Preset{updated})
	}

	deleted, err := DeletePreset(promptsDir, created.ID)
	if err != nil {
		t.Fatalf("DeletePreset: %v", err)
	}
	if !reflect.DeepEqual(deleted, updated) {
		t.Fatalf("unexpected deleted preset: got %+v want %+v", deleted, updated)
	}

	finalPresets, err := LoadPresets(promptsDir)
	if err != nil {
		t.Fatalf("LoadPresets after delete: %v", err)
	}
	if len(finalPresets) != 0 {
		t.Fatalf("expected empty presets after delete, got %+v", finalPresets)
	}
}

func TestCreatePresetRejectsUnknownTool(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")
	seedPresetPromptLibrary(t, promptsDir)

	_, err := CreatePreset(promptsDir, PresetCreateRequest{
		Name:          "Invalid Tool",
		ToolAllowlist: []string{"not_exists"},
	})
	if !errors.Is(err, errPresetInvalid) {
		t.Fatalf("expected preset validation error, got %v", err)
	}
}

func TestCreatePresetRejectsMissingPromptRef(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")
	seedPresetPromptLibrary(t, promptsDir)

	_, err := CreatePreset(promptsDir, PresetCreateRequest{
		Name: "Missing Prompt",
		PromptRefs: PresetPromptRefs{
			Rule: "missing-card",
		},
	})
	if !errors.Is(err, errPresetInvalid) {
		t.Fatalf("expected preset validation error, got %v", err)
	}
}

func TestCreatePresetRejectsWrongPromptInsertPoint(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")
	seedPresetPromptLibrary(t, promptsDir)

	_, err := CreatePreset(promptsDir, PresetCreateRequest{
		Name: "Wrong Insert Point",
		PromptRefs: PresetPromptRefs{
			Memory: "rule-card",
		},
	})
	if !errors.Is(err, errPresetInvalid) {
		t.Fatalf("expected preset validation error, got %v", err)
	}
}

func seedPresetPromptLibrary(t *testing.T, promptsDir string) {
	t.Helper()

	library := []promptstore.SystemPromptLibraryItem{
		{
			ID:          "rule-card",
			Name:        "Rule",
			InsertPoint: promptstore.SystemPromptInsertPointRule,
			Content:     "rule content",
			Active:      true,
		},
		{
			ID:          "core-card",
			Name:        "Core",
			InsertPoint: promptstore.SystemPromptInsertPointCoreJob,
			Content:     "core content",
			Active:      true,
		},
		{
			ID:          "memory-card",
			Name:        "Memory",
			InsertPoint: promptstore.SystemPromptInsertPointMemory,
			Content:     "memory content",
			Active:      true,
		},
		{
			ID:          "context-a",
			Name:        "Context A",
			InsertPoint: promptstore.SystemPromptInsertPointContext,
			Content:     "context alpha",
			Active:      true,
		},
		{
			ID:          "context-b",
			Name:        "Context B",
			InsertPoint: promptstore.SystemPromptInsertPointContext,
			Content:     "context beta",
			Active:      true,
		},
	}
	if _, err := promptstore.UpdateSystemPromptFiles(promptsDir, promptstore.SystemPromptUpdateRequest{
		PromptLibrary: &library,
	}); err != nil {
		t.Fatalf("UpdateSystemPromptFiles: %v", err)
	}
}

func stringPointer(value string) *string {
	return &value
}

func assertPresetFileContent(t *testing.T, root string, want []Preset) {
	t.Helper()

	path := filepath.Join(root, presetFileName)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	var got []Preset
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("json.Unmarshal(%s): %v", path, err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected preset file content: got %+v want %+v", got, want)
	}
}
