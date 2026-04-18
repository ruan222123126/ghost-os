package config

import (
	"errors"
	"strings"
)

// Update 按固定流水线执行：load current file -> apply patch -> normalize -> resolve/validate -> persist。
func (s *store) Update(req UpdateRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.loadUpdateStateLocked(req)
	if err != nil {
		return err
	}
	if err := applyConfigUpdatePatch(&state.fileCfg, state.current, req); err != nil {
		return err
	}

	normalized, runtime, err := resolveConfigForPersist(state.fileCfg, state.current)
	if err != nil {
		return err
	}
	if err := writeBridgeFileConfig(state.configPath, normalized); err != nil {
		return err
	}
	s.runtime = runtime
	return nil
}

func applyConfigUpdatePatch(
	fileCfg *bridgeFileConfig,
	current runtimeConfig,
	req UpdateRequest,
) error {
	if err := applyProviderUpdatePatch(fileCfg, current, req); err != nil {
		return err
	}
	if err := applyGraphQLUpdatePatch(fileCfg, req); err != nil {
		return err
	}
	applyConfigScalarUpdatePatch(fileCfg, req)
	return nil
}

func applyConfigScalarUpdatePatch(fileCfg *bridgeFileConfig, req UpdateRequest) {
	if req.Model != nil {
		fileCfg.Model = cloneOptionalStringPointer(req.Model)
	}
	if req.ChatPath != nil {
		fileCfg.ChatPath = cloneOptionalStringPointer(req.ChatPath)
	}
	if req.WebSearchTavilyURL != nil {
		fileCfg.WebSearchTavilyURL = cloneOptionalStringPointer(req.WebSearchTavilyURL)
	}
	if req.WebSearchExaURL != nil {
		fileCfg.WebSearchExaURL = cloneOptionalStringPointer(req.WebSearchExaURL)
	}
	if req.WebSearchTavilyAPIKey != nil {
		fileCfg.WebSearchTavilyAPIKey = cloneOptionalStringPointer(req.WebSearchTavilyAPIKey)
	}
	if req.WebSearchExaAPIKey != nil {
		fileCfg.WebSearchExaAPIKey = cloneOptionalStringPointer(req.WebSearchExaAPIKey)
	}
	if req.SessionHumanLogFullEnabled != nil {
		fileCfg.SessionHumanLogFullEnabled = cloneBoolPointer(req.SessionHumanLogFullEnabled)
	}
	if req.WebRooterEnabled != nil {
		fileCfg.WebRooterEnabled = cloneBoolPointer(req.WebRooterEnabled)
	}
	if req.WebRooterBaseURL != nil {
		fileCfg.WebRooterBaseURL = cloneStringPointer(req.WebRooterBaseURL)
	}
	if req.WebRooterAPIToken != nil {
		fileCfg.WebRooterAPIToken = cloneOptionalStringPointer(req.WebRooterAPIToken)
	}
	if req.WebRooterTimeoutMS != nil {
		fileCfg.WebRooterTimeoutMS = cloneIntPointer(req.WebRooterTimeoutMS)
	}
}

// SetProjectRoot 更新并持久化 project_root，并刷新运行态快照。
func (s *store) SetProjectRoot(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return errors.New("project_root is required")
	}

	fileCfg, configPath, err := s.loadStoredFileConfigLocked()
	if err != nil {
		return err
	}
	fileCfg.ProjectRoot = stringPointer(trimmed)
	return s.persistLocked(configPath, fileCfg)
}

func (s *store) loadStoredFileConfigLocked() (bridgeFileConfig, string, error) {
	return loadBridgeFileConfig()
}

func (s *store) persistLocked(configPath string, fileCfg bridgeFileConfig) error {
	normalized, runtime, err := resolveConfigForPersist(fileCfg, s.runtime)
	if err != nil {
		return err
	}
	if err := writeBridgeFileConfig(configPath, normalized); err != nil {
		return err
	}
	s.runtime = runtime
	return nil
}

func resolveConfigForPersist(
	fileCfg bridgeFileConfig,
	fallback runtimeConfig,
) (bridgeFileConfig, runtimeConfig, error) {
	normalized, err := normalizeBridgeFileConfigForWrite(fileCfg)
	if err != nil {
		return bridgeFileConfig{}, runtimeConfig{}, err
	}
	runtime, err := resolveRuntimeConfigWithFallback(normalized, fallback)
	if err != nil {
		return bridgeFileConfig{}, runtimeConfig{}, err
	}
	return normalized, runtime, nil
}
