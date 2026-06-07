package presets

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"ghost-os/bridge/config/internal/prompts"
	"ghost-os/bridge/config/internal/tools"
)

type promptLibraryByID map[string]prompts.SystemPromptLibraryItem

func parsePresetList(raw string) ([]Preset, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []Preset{}, nil
	}

	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.DisallowUnknownFields()
	var presets []Preset
	if err := decoder.Decode(&presets); err != nil {
		return nil, fmt.Errorf("%w: parse presets: %v", errPresetInvalid, err)
	}
	if err := rejectTrailingPresetJSON(decoder); err != nil {
		return nil, err
	}
	return presets, nil
}

func rejectTrailingPresetJSON(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("%w: parse presets: %v", errPresetInvalid, err)
	}
	return fmt.Errorf("%w: presets must be a single JSON array", errPresetInvalid)
}

func marshalPresetList(presets []Preset) (string, error) {
	payload, err := json.Marshal(presets)
	if err != nil {
		return "", fmt.Errorf("%w: marshal presets: %v", errPresetInvalid, err)
	}
	return string(payload), nil
}

func buildPromptLibraryByID(library []prompts.SystemPromptLibraryItem) promptLibraryByID {
	index := make(promptLibraryByID, len(library))
	for _, item := range library {
		index[item.ID] = item
	}
	return index
}

func normalizePresetList(presets []Preset, library promptLibraryByID) ([]Preset, error) {
	normalized := make([]Preset, 0, len(presets))
	ids := make(map[string]struct{}, len(presets))

	for index, preset := range presets {
		next, err := normalizePreset(preset, index, library)
		if err != nil {
			return nil, err
		}
		if _, exists := ids[next.ID]; exists {
			return nil, newPresetValidationError("preset %d id %q is duplicated", index, next.ID)
		}
		ids[next.ID] = struct{}{}
		normalized = append(normalized, next)
	}
	return normalized, nil
}

func normalizePreset(preset Preset, index int, library promptLibraryByID) (Preset, error) {
	id := strings.TrimSpace(preset.ID)
	if id == "" {
		return Preset{}, newPresetValidationError("preset %d id is required", index)
	}

	name := strings.TrimSpace(preset.Name)
	if name == "" {
		return Preset{}, newPresetValidationError("preset %d name is required", index)
	}

	toolAllowlist, err := normalizePresetToolAllowlist(preset.ToolAllowlist, index)
	if err != nil {
		return Preset{}, err
	}
	promptRefs, err := normalizePresetPromptRefs(preset.PromptRefs, index, library)
	if err != nil {
		return Preset{}, err
	}
	return Preset{
		ID:            id,
		Name:          name,
		ToolAllowlist: toolAllowlist,
		PromptRefs:    promptRefs,
	}, nil
}

func normalizePresetToolAllowlist(raw []string, index int) ([]string, error) {
	normalized := tools.NormalizeConfiguredToolNames(raw)
	valid := tools.ValidConfiguredToolNames()
	for _, name := range normalized {
		if !valid[name] {
			return nil, newPresetValidationError(
				"preset %d tool_allowlist contains unknown tool %q",
				index,
				name,
			)
		}
	}
	if len(normalized) == 0 {
		return []string{}, nil
	}
	return normalized, nil
}

func normalizePresetPromptRefs(
	refs PresetPromptRefs,
	index int,
	library promptLibraryByID,
) (PresetPromptRefs, error) {
	rule := strings.TrimSpace(refs.Rule)
	coreJob := strings.TrimSpace(refs.CoreJob)
	memory := strings.TrimSpace(refs.Memory)

	if err := validatePresetPromptRef(index, "rule", rule, prompts.SystemPromptInsertPointRule, library); err != nil {
		return PresetPromptRefs{}, err
	}
	if err := validatePresetPromptRef(index, "core_job", coreJob, prompts.SystemPromptInsertPointCoreJob, library); err != nil {
		return PresetPromptRefs{}, err
	}
	if err := validatePresetPromptRef(index, "memory", memory, prompts.SystemPromptInsertPointMemory, library); err != nil {
		return PresetPromptRefs{}, err
	}
	contextRefs, err := normalizePresetContextPromptRefs(index, refs.Context, library)
	if err != nil {
		return PresetPromptRefs{}, err
	}
	return PresetPromptRefs{
		Rule:    rule,
		CoreJob: coreJob,
		Memory:  memory,
		Context: contextRefs,
	}, nil
}

func normalizePresetContextPromptRefs(
	index int,
	refs []string,
	library promptLibraryByID,
) ([]string, error) {
	if len(refs) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{}, len(refs))
	normalized := make([]string, 0, len(refs))
	for refIndex, rawRef := range refs {
		refID := strings.TrimSpace(rawRef)
		if refID == "" {
			continue
		}
		if _, exists := seen[refID]; exists {
			continue
		}
		slot := fmt.Sprintf("context[%d]", refIndex)
		if err := validatePresetPromptRef(index, slot, refID, prompts.SystemPromptInsertPointContext, library); err != nil {
			return nil, err
		}
		seen[refID] = struct{}{}
		normalized = append(normalized, refID)
	}
	if len(normalized) == 0 {
		return nil, nil
	}
	return normalized, nil
}

func validatePresetPromptRef(
	index int,
	slot string,
	refID string,
	insertPoint prompts.SystemPromptInsertPoint,
	library promptLibraryByID,
) error {
	if refID == "" {
		return nil
	}
	item, ok := library[refID]
	if !ok {
		return newPresetValidationError(
			"preset %d prompt_refs.%s references unknown prompt %q",
			index,
			slot,
			refID,
		)
	}
	if item.InsertPoint != insertPoint {
		return newPresetValidationError(
			"preset %d prompt_refs.%s references insert_point %q, want %q",
			index,
			slot,
			item.InsertPoint,
			insertPoint,
		)
	}
	return nil
}

func newPresetValidationError(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errPresetInvalid, fmt.Sprintf(format, args...))
}
