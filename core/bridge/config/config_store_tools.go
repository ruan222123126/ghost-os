package config

import (
	"fmt"
	"strings"
)

func (s *store) ListTools() ([]ToolRecord, error) {
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
	overrides, err := loadToolPromptOverridesFromFiles(promptsDir)
	if err != nil {
		return nil, err
	}

	allowlisted := toolNameSetFromSlice(fileCfg.ToolAllowlist)
	records := make([]ToolRecord, 0, len(configuredToolCatalog))
	for _, raw := range configuredToolCatalog {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		records = append(records, ToolRecord{
			Name:           name,
			Enabled:        allowlisted[name],
			PromptOverride: strings.TrimSpace(overrides[name]),
		})
	}
	return records, nil
}

func (s *store) UpdateTool(req ToolUpdateRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return errToolNameRequired
	}
	if !validConfiguredToolNames()[name] {
		return fmt.Errorf("%w: %s", errToolNotFound, name)
	}
	if req.Enabled == nil && req.PromptOverride == nil {
		return errToolUpdateEmpty
	}

	fileCfg, configPath, err := s.loadStoredFileConfigLocked()
	if err != nil {
		return err
	}
	if req.Enabled != nil {
		fileCfg.ToolAllowlist = withUpdatedToolAllowlist(fileCfg.ToolAllowlist, name, *req.Enabled)
		fileCfg.ToolBlocklist = withUpdatedToolBlocklist(fileCfg.ToolBlocklist, name, *req.Enabled)
	}
	if req.PromptOverride != nil {
		promptsDir, resolveErr := resolvePromptsDir(fileCfg, currentEnv())
		if resolveErr != nil {
			return resolveErr
		}
		if writeErr := writeToolPromptOverrideToFile(promptsDir, name, *req.PromptOverride); writeErr != nil {
			return writeErr
		}
	}
	fileCfg.ToolPromptOverrides = nil
	return s.persistLocked(configPath, fileCfg)
}

func withUpdatedToolAllowlist(raw []string, name string, enabled bool) []string {
	allowlist := normalizeConfiguredToolNames(raw)
	if enabled {
		return normalizeConfiguredToolNames(append(allowlist, name))
	}

	filtered := make([]string, 0, len(allowlist))
	for _, item := range allowlist {
		if item != name {
			filtered = append(filtered, item)
		}
	}
	return normalizeConfiguredToolNames(filtered)
}

func withUpdatedToolBlocklist(raw []string, name string, enabled bool) []string {
	blocklist := normalizeConfiguredToolNames(raw)
	if !enabled {
		return normalizeConfiguredToolNames(append(blocklist, name))
	}

	filtered := make([]string, 0, len(blocklist))
	for _, item := range blocklist {
		if item != name {
			filtered = append(filtered, item)
		}
	}
	return normalizeConfiguredToolNames(filtered)
}

func toolNameSetFromSlice(raw []string) map[string]bool {
	set := make(map[string]bool, len(raw))
	for _, name := range normalizeConfiguredToolNames(raw) {
		set[name] = true
	}
	return set
}
