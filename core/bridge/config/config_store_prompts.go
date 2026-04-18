package config

func (s *store) UpdateSystemPrompts(req SystemPromptUpdateRequest) (SystemPromptFiles, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fileCfg, _, err := s.loadStoredFileConfigLocked()
	if err != nil {
		return SystemPromptFiles{}, err
	}
	promptsDir, err := resolvePromptsDir(fileCfg, currentEnv())
	if err != nil {
		return SystemPromptFiles{}, err
	}
	return UpdateSystemPromptFiles(promptsDir, req)
}
