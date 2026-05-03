package config

import "strings"

func (s *store) Presets() ([]Preset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fileCfg, _, err := s.loadStoredFileConfigLocked()
	if err != nil {
		return nil, err
	}
	promptsDir, err := resolvePromptsDir(fileCfg, currentEnv())
	if err != nil {
		return nil, err
	}
	return LoadPresets(promptsDir)
}

func (s *store) CreatePreset(req PresetCreateRequest) (Preset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fileCfg, _, err := s.loadStoredFileConfigLocked()
	if err != nil {
		return Preset{}, err
	}
	promptsDir, err := resolvePromptsDir(fileCfg, currentEnv())
	if err != nil {
		return Preset{}, err
	}
	req.Name = strings.TrimSpace(req.Name)
	return CreatePreset(promptsDir, req)
}

func (s *store) UpdatePreset(presetID string, req PresetUpdateRequest) (Preset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fileCfg, _, err := s.loadStoredFileConfigLocked()
	if err != nil {
		return Preset{}, err
	}
	promptsDir, err := resolvePromptsDir(fileCfg, currentEnv())
	if err != nil {
		return Preset{}, err
	}
	return UpdatePreset(promptsDir, presetID, req)
}

func (s *store) DeletePreset(presetID string) (Preset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fileCfg, _, err := s.loadStoredFileConfigLocked()
	if err != nil {
		return Preset{}, err
	}
	promptsDir, err := resolvePromptsDir(fileCfg, currentEnv())
	if err != nil {
		return Preset{}, err
	}
	return DeletePreset(promptsDir, presetID)
}
