package config

import "strings"

func (s *store) SetSkillEnabled(skillID string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(skillID)
	if id == "" {
		return errSkillIDRequired
	}

	fileCfg, configPath, err := s.loadStoredFileConfigLocked()
	if err != nil {
		return err
	}
	fileCfg.SkillBlocklist = withUpdatedSkillBlocklist(fileCfg.SkillBlocklist, id, enabled)
	return s.persistLocked(configPath, fileCfg)
}

func withUpdatedSkillBlocklist(raw []string, skillID string, enabled bool) []string {
	items := normalizeStringList(raw)
	if !enabled {
		return normalizeStringList(append(items, skillID))
	}

	filtered := make([]string, 0, len(items))
	for _, item := range items {
		if item != skillID {
			filtered = append(filtered, item)
		}
	}
	return normalizeStringList(filtered)
}
