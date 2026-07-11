package presets

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCreatePresetNormalizesContextPromptRefs(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")
	seedPresetPromptLibrary(t, promptsDir)

	created, err := CreatePreset(promptsDir, PresetCreateRequest{
		Name: "Context Preset",
		PromptRefs: PresetPromptRefs{
			Context: []string{" context-b ", "", "context-a", "context-b", "context-a"},
		},
	})
	if err != nil {
		t.Fatalf("CreatePreset: %v", err)
	}

	if !reflect.DeepEqual(created.PromptRefs.Context, []string{"context-b", "context-a"}) {
		t.Fatalf("unexpected context refs: %+v", created.PromptRefs)
	}
}

func TestCreatePresetRejectsContextPromptWithWrongInsertPoint(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")
	seedPresetPromptLibrary(t, promptsDir)

	_, err := CreatePreset(promptsDir, PresetCreateRequest{
		Name: "Invalid Context",
		PromptRefs: PresetPromptRefs{
			Context: []string{"rule-card"},
		},
	})
	if !errors.Is(err, errPresetInvalid) {
		t.Fatalf("expected preset validation error, got %v", err)
	}
}
