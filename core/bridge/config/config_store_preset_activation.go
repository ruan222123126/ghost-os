package config

import (
	"fmt"
	"strings"
)

func (s *store) ApplyPreset(presetID string) (Preset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fileCfg, configPath, err := s.loadStoredFileConfigLocked()
	if err != nil {
		return Preset{}, err
	}
	promptsDir, err := resolvePromptsDir(fileCfg, currentEnv())
	if err != nil {
		return Preset{}, err
	}

	preset, err := loadPresetByID(promptsDir, presetID)
	if err != nil {
		return Preset{}, err
	}
	promptFiles, err := LoadSystemPromptFiles(promptsDir)
	if err != nil {
		return Preset{}, err
	}

	nextPromptLibrary, err := applyPresetToPromptLibrary(promptFiles.PromptLibrary, preset)
	if err != nil {
		return Preset{}, err
	}
	if _, err := UpdateSystemPromptFiles(promptsDir, SystemPromptUpdateRequest{
		PromptLibrary: &nextPromptLibrary,
	}); err != nil {
		return Preset{}, err
	}

	fileCfg = applyPresetToFileConfig(fileCfg, preset)
	if err := s.persistLocked(configPath, fileCfg); err != nil {
		return Preset{}, err
	}
	return preset, nil
}

func loadPresetByID(promptsDir string, presetID string) (Preset, error) {
	state, err := loadPresetState(promptsDir)
	if err != nil {
		return Preset{}, err
	}

	index := presetIndexByID(state.presets, presetID)
	if index >= 0 {
		return state.presets[index], nil
	}
	return Preset{}, fmt.Errorf("%w: %s", errPresetNotFound, strings.TrimSpace(presetID))
}
