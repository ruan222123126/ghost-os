package config

import (
	"fmt"
	"strings"

	"ghost-os/bridge/config/internal/providers"
	"ghost-os/bridge/config/internal/skills"
	configtools "ghost-os/bridge/config/internal/tools"
)

const scriptExecToolName = "script_exec"

func (s *store) AddProvider(cfg ProviderRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fileCfg, configPath, err := s.loadStoredFileConfigLocked()
	if err != nil {
		return err
	}
	patch, err := providers.Add(providerStateFromFileConfig(fileCfg), providerConfigFromRecord(cfg))
	if err != nil {
		return err
	}
	applyProviderPatchToFileConfig(&fileCfg, patch)
	return s.persistLocked(configPath, fileCfg)
}

func (s *store) UpdateProvider(name string, cfg ProviderRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fileCfg, configPath, err := s.loadStoredFileConfigLocked()
	if err != nil {
		return err
	}
	patch, err := providers.Update(providerStateFromFileConfig(fileCfg), name, providerConfigFromRecord(cfg))
	if err != nil {
		return err
	}
	applyProviderPatchToFileConfig(&fileCfg, patch)
	return s.persistLocked(configPath, fileCfg)
}

func (s *store) DeleteProvider(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fileCfg, configPath, err := s.loadStoredFileConfigLocked()
	if err != nil {
		return err
	}
	patch, err := providers.Delete(providerStateFromFileConfig(fileCfg), name)
	if err != nil {
		return err
	}
	applyProviderPatchToFileConfig(&fileCfg, patch)
	return s.persistLocked(configPath, fileCfg)
}

func (s *store) SetActiveProvider(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fileCfg, configPath, err := s.loadStoredFileConfigLocked()
	if err != nil {
		return err
	}
	patch, err := providers.SetActive(providerStateFromFileConfig(fileCfg), name)
	if err != nil {
		return err
	}
	applyProviderPatchToFileConfig(&fileCfg, patch)
	return s.persistLocked(configPath, fileCfg)
}

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
	scriptExecSandboxMemoryMB, err := resolveScriptExecSandboxMemoryMB(fileCfg, currentEnv())
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
		record := ToolRecord{
			Name:           name,
			Enabled:        allowlisted[name],
			PromptOverride: strings.TrimSpace(overrides[name]),
		}
		if name == scriptExecToolName {
			record.SandboxMemoryMB = intPointer(scriptExecSandboxMemoryMB)
		}
		records = append(records, record)
	}
	return records, nil
}

func (s *store) UpdateTool(req ToolUpdateRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	name, err := validateToolUpdateRequest(req)
	if err != nil {
		return err
	}

	fileCfg, configPath, err := s.loadStoredFileConfigLocked()
	if err != nil {
		return err
	}
	if err := applyToolUpdateRequest(&fileCfg, name, req); err != nil {
		return err
	}
	return s.persistLocked(configPath, fileCfg)
}

func validateToolUpdateRequest(req ToolUpdateRequest) (string, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return "", errToolNameRequired
	}
	if !validConfiguredToolNames()[name] {
		return "", fmt.Errorf("%w: %s", errToolNotFound, name)
	}
	if req.Enabled == nil && req.PromptOverride == nil && req.SandboxMemoryMB == nil {
		return "", errToolUpdateEmpty
	}
	return name, nil
}

func applyToolUpdateRequest(fileCfg *bridgeFileConfig, name string, req ToolUpdateRequest) error {
	if req.Enabled != nil {
		applyToolEnabledUpdate(fileCfg, name, *req.Enabled)
	}
	if req.PromptOverride != nil {
		if err := applyToolPromptOverride(*fileCfg, name, *req.PromptOverride); err != nil {
			return err
		}
	}
	if req.SandboxMemoryMB != nil {
		if err := applyToolSandboxMemory(fileCfg, name, *req.SandboxMemoryMB); err != nil {
			return err
		}
	}
	return nil
}

func applyToolEnabledUpdate(fileCfg *bridgeFileConfig, name string, enabled bool) {
	fileCfg.ToolAllowlist = withUpdatedToolAllowlist(fileCfg.ToolAllowlist, name, enabled)
	fileCfg.ToolBlocklist = withUpdatedToolBlocklist(fileCfg.ToolBlocklist, name, enabled)
}

func applyToolPromptOverride(fileCfg bridgeFileConfig, name string, prompt string) error {
	promptsDir, err := resolvePromptsDir(fileCfg, currentEnv())
	if err != nil {
		return err
	}
	return writeToolPromptOverrideToFile(promptsDir, name, prompt)
}

func applyToolSandboxMemory(fileCfg *bridgeFileConfig, name string, value int) error {
	if name != scriptExecToolName {
		return fmt.Errorf("%w: sandbox_memory_mb is only supported for %s", errToolConfigInvalid, scriptExecToolName)
	}
	if value <= 0 || value > maxScriptExecSandboxMemoryMB {
		return fmt.Errorf(
			"%w: sandbox_memory_mb must be between 1 and %d",
			errToolConfigInvalid,
			maxScriptExecSandboxMemoryMB,
		)
	}
	fileCfg.ScriptExecSandboxMemoryMB = intPointer(value)
	return nil
}

func withUpdatedToolAllowlist(raw []string, name string, enabled bool) []string {
	return configtools.WithUpdatedAllowlist(raw, name, enabled)
}

func withUpdatedToolBlocklist(raw []string, name string, enabled bool) []string {
	return configtools.WithUpdatedBlocklist(raw, name, enabled)
}

func intPointer(value int) *int {
	return &value
}

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
	presets, err := LoadPresets(promptsDir)
	if err != nil {
		return Preset{}, err
	}

	preset, ok := FindPresetByID(presets, presetID)
	if ok {
		return preset, nil
	}
	return Preset{}, fmt.Errorf("%w: %s", errPresetNotFound, strings.TrimSpace(presetID))
}

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
	fileCfg.SkillBlocklist = skills.WithUpdatedBlocklist(fileCfg.SkillBlocklist, id, enabled)
	return s.persistLocked(configPath, fileCfg)
}
