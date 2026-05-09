package config

import "strings"

func ApplyPresetToSystemPromptFiles(files SystemPromptFiles, preset Preset) (SystemPromptFiles, error) {
	library, err := applyPresetToPromptLibrary(files.PromptLibrary, preset)
	if err != nil {
		return SystemPromptFiles{}, err
	}
	next := cloneSystemPromptFiles(files)
	next.PromptLibrary = library
	next.CorePrompt = compileCorePromptFromLibrary(library)
	return next, nil
}

func FindPresetByID(presets []Preset, presetID string) (Preset, bool) {
	target := strings.TrimSpace(presetID)
	for _, preset := range presets {
		if strings.TrimSpace(preset.ID) == target {
			return preset, true
		}
	}
	return Preset{}, false
}
