package config

import (
	"crypto/rand"
	"fmt"
	"reflect"
	"strings"
)

type presetState struct {
	roots   []string
	presets []Preset
}

func LoadPresets(promptsDir string) ([]Preset, error) {
	state, err := loadPresetState(promptsDir)
	if err != nil {
		return nil, err
	}
	return state.presets, nil
}

func CreatePreset(promptsDir string, req PresetCreateRequest) (Preset, error) {
	state, err := loadPresetState(promptsDir)
	if err != nil {
		return Preset{}, err
	}

	preset, err := buildCreatedPreset(state.presets, req)
	if err != nil {
		return Preset{}, err
	}
	presets, err := appendNormalizedPreset(state.presets, preset, promptsDir)
	if err != nil {
		return Preset{}, err
	}
	if err := writePresetsToRoots(state.roots, presets); err != nil {
		return Preset{}, err
	}
	return presets[len(presets)-1], nil
}

func UpdatePreset(promptsDir string, presetID string, req PresetUpdateRequest) (Preset, error) {
	if !req.hasUpdates() {
		return Preset{}, errPresetUpdateEmpty
	}
	state, err := loadPresetState(promptsDir)
	if err != nil {
		return Preset{}, err
	}

	index := presetIndexByID(state.presets, presetID)
	if index < 0 {
		return Preset{}, fmt.Errorf("%w: %s", errPresetNotFound, strings.TrimSpace(presetID))
	}
	next := applyPresetUpdate(state.presets[index], req)
	presets, err := replaceNormalizedPreset(state.presets, index, next, promptsDir)
	if err != nil {
		return Preset{}, err
	}
	if err := writePresetsToRoots(state.roots, presets); err != nil {
		return Preset{}, err
	}
	return presets[index], nil
}

func DeletePreset(promptsDir string, presetID string) (Preset, error) {
	state, err := loadPresetState(promptsDir)
	if err != nil {
		return Preset{}, err
	}

	index := presetIndexByID(state.presets, presetID)
	if index < 0 {
		return Preset{}, fmt.Errorf("%w: %s", errPresetNotFound, strings.TrimSpace(presetID))
	}
	deleted := state.presets[index]
	presets := append(state.presets[:index:index], state.presets[index+1:]...)
	if err := writePresetsToRoots(state.roots, presets); err != nil {
		return Preset{}, err
	}
	return deleted, nil
}

func loadPresetState(promptsDir string) (presetState, error) {
	roots, err := ensurePresetRoots(promptsDir)
	if err != nil {
		return presetState{}, err
	}
	if err := syncPresetRoots(roots); err != nil {
		return presetState{}, err
	}

	presets, err := readPresetsFromRoot(roots[0])
	if err != nil {
		return presetState{}, err
	}
	normalized, changed, err := normalizeLoadedPresets(promptsDir, presets)
	if err != nil {
		return presetState{}, err
	}
	if changed {
		if err := writePresetsToRoots(roots, normalized); err != nil {
			return presetState{}, err
		}
	}
	return presetState{roots: roots, presets: normalized}, nil
}

func normalizeLoadedPresets(promptsDir string, presets []Preset) ([]Preset, bool, error) {
	library, err := loadPresetPromptLibrary(promptsDir)
	if err != nil {
		return nil, false, err
	}
	normalized, err := normalizePresetList(presets, library)
	if err != nil {
		return nil, false, err
	}
	return normalized, !reflect.DeepEqual(presets, normalized), nil
}

func loadPresetPromptLibrary(promptsDir string) (promptLibraryByID, error) {
	files, err := LoadSystemPromptFiles(promptsDir)
	if err != nil {
		return nil, err
	}
	return buildPromptLibraryByID(files.PromptLibrary), nil
}

func buildCreatedPreset(existing []Preset, req PresetCreateRequest) (Preset, error) {
	id, err := newPresetID(existing)
	if err != nil {
		return Preset{}, err
	}
	return Preset{
		ID:            id,
		Name:          req.Name,
		ToolAllowlist: req.ToolAllowlist,
		PromptRefs:    req.PromptRefs,
	}, nil
}

func appendNormalizedPreset(existing []Preset, preset Preset, promptsDir string) ([]Preset, error) {
	presets := append(append([]Preset{}, existing...), preset)
	return normalizeLoadedPresetsOnly(promptsDir, presets)
}

func replaceNormalizedPreset(
	existing []Preset,
	index int,
	preset Preset,
	promptsDir string,
) ([]Preset, error) {
	presets := append([]Preset{}, existing...)
	presets[index] = preset
	return normalizeLoadedPresetsOnly(promptsDir, presets)
}

func normalizeLoadedPresetsOnly(promptsDir string, presets []Preset) ([]Preset, error) {
	library, err := loadPresetPromptLibrary(promptsDir)
	if err != nil {
		return nil, err
	}
	return normalizePresetList(presets, library)
}

func presetIndexByID(presets []Preset, presetID string) int {
	trimmedID := strings.TrimSpace(presetID)
	for index, preset := range presets {
		if preset.ID == trimmedID {
			return index
		}
	}
	return -1
}

func applyPresetUpdate(preset Preset, req PresetUpdateRequest) Preset {
	next := preset
	if req.Name != nil {
		next.Name = *req.Name
	}
	if req.ToolAllowlist != nil {
		next.ToolAllowlist = *req.ToolAllowlist
	}
	if req.PromptRefs != nil {
		next.PromptRefs = *req.PromptRefs
	}
	return next
}

func newPresetID(existing []Preset) (string, error) {
	existingIDs := make(map[string]struct{}, len(existing))
	for _, preset := range existing {
		existingIDs[preset.ID] = struct{}{}
	}
	for {
		id, err := randomPresetID()
		if err != nil {
			return "", err
		}
		if _, exists := existingIDs[id]; !exists {
			return id, nil
		}
	}
}

func randomPresetID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate preset id: %w", err)
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return fmt.Sprintf(
		"preset-%x-%x-%x-%x-%x",
		raw[0:4],
		raw[4:6],
		raw[6:8],
		raw[8:10],
		raw[10:16],
	), nil
}
