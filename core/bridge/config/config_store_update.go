package config

import "strings"

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

	persistFallback, err := updatePersistFallback(state.current, req)
	if err != nil {
		return err
	}
	normalized, runtime, err := resolveConfigForPersist(state.fileCfg, persistFallback)
	if err != nil {
		return err
	}
	if err := writeBridgeFileConfig(state.configPath, normalized); err != nil {
		return err
	}
	s.runtime = runtime
	return nil
}

func updatePersistFallback(current runtimeConfig, req UpdateRequest) (runtimeConfig, error) {
	fallback := cloneRuntimeConfig(current)
	if !hasResetStringField(req) {
		return fallback, nil
	}

	envFallback, err := runtimeFallbackFromEnv(currentEnv())
	if err != nil {
		return runtimeConfig{}, err
	}
	if resetsStringValue(req.Model) {
		fallback.Model = envFallback.Model
	}
	if resetsStringValue(req.ChatPath) {
		fallback.ChatPath = envFallback.ChatPath
	}
	if resetsStringValue(req.ProjectRoot) {
		fallback.ProjectRoot = envFallback.ProjectRoot
	}
	if resetsStringValue(req.WebSearchTavilyURL) {
		fallback.WebSearchTavilyURL = envFallback.WebSearchTavilyURL
	}
	if resetsStringValue(req.WebSearchExaURL) {
		fallback.WebSearchExaURL = envFallback.WebSearchExaURL
	}
	return fallback, nil
}

func hasResetStringField(req UpdateRequest) bool {
	return resetsStringValue(req.Model) ||
		resetsStringValue(req.ChatPath) ||
		resetsStringValue(req.ProjectRoot) ||
		resetsStringValue(req.WebSearchTavilyURL) ||
		resetsStringValue(req.WebSearchExaURL)
}

func resetsStringValue(raw *string) bool {
	return raw != nil && strings.TrimSpace(*raw) == ""
}

func applyConfigUpdatePatch(
	fileCfg *bridgeFileConfig,
	current runtimeConfig,
	req UpdateRequest,
) error {
	if err := applyProviderUpdatePatch(fileCfg, current, req); err != nil {
		return err
	}
	return applyConfigScalarUpdatePatch(fileCfg, req)
}

func applyConfigScalarUpdatePatch(fileCfg *bridgeFileConfig, req UpdateRequest) error {
	if req.Model != nil {
		fileCfg.Model = cloneOptionalStringPointer(req.Model)
	}
	if req.ChatPath != nil {
		fileCfg.ChatPath = cloneOptionalStringPointer(req.ChatPath)
	}
	if err := applyProjectRootUpdate(fileCfg, req.ProjectRoot); err != nil {
		return err
	}
	if req.MaxTurns != nil {
		fileCfg.MaxTurns = cloneIntPointer(req.MaxTurns)
	}
	if req.TaskExecutionTimeoutMS != nil {
		fileCfg.TaskExecutionTimeoutMS = cloneIntPointer(req.TaskExecutionTimeoutMS)
	}
	if req.RelayDefaultStopPolicy != nil {
		fileCfg.RelayDefaultStopPolicy = cloneOptionalStringPointer(req.RelayDefaultStopPolicy)
	}
	if req.RelayDefaultMaxRounds != nil {
		fileCfg.RelayDefaultMaxRounds = cloneIntPointer(req.RelayDefaultMaxRounds)
	}
	if req.RelayDefaultExecutionTimeoutMS != nil {
		fileCfg.RelayDefaultExecutionTimeoutMS = cloneIntPointer(req.RelayDefaultExecutionTimeoutMS)
	}
	if req.LLMCompletionRetryCount != nil {
		fileCfg.LLMCompletionRetryCount = cloneIntPointer(req.LLMCompletionRetryCount)
	}
	if req.LLMCompletionRetryIntervalMS != nil {
		fileCfg.LLMCompletionRetryIntervalMS = cloneIntPointer(req.LLMCompletionRetryIntervalMS)
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
	if req.SessionSystemPromptVisible != nil {
		fileCfg.SessionSystemPromptVisible = cloneBoolPointer(req.SessionSystemPromptVisible)
	}
	if req.AssistantMarkdownEnabled != nil {
		fileCfg.AssistantMarkdownEnabled = cloneBoolPointer(req.AssistantMarkdownEnabled)
	}
	if req.ToolCallCompactOutputEnabled != nil {
		fileCfg.ToolCallCompactOutputEnabled = cloneBoolPointer(req.ToolCallCompactOutputEnabled)
	}
	if req.MemoryModeEnabled != nil {
		fileCfg.MemoryModeEnabled = cloneBoolPointer(req.MemoryModeEnabled)
	}
	if req.MicrocompactEnabled != nil {
		fileCfg.MicrocompactEnabled = cloneBoolPointer(req.MicrocompactEnabled)
	}
	if req.SessionTitleMode != nil {
		fileCfg.SessionTitleMode = cloneOptionalStringPointer(req.SessionTitleMode)
	}
	return nil
}

// SetProjectRoot 更新并持久化 project_root，并刷新运行态快照。
func (s *store) SetProjectRoot(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fileCfg, configPath, err := s.loadStoredFileConfigLocked()
	if err != nil {
		return err
	}
	normalized, err := NormalizeProjectRoot(path)
	if err != nil {
		return err
	}
	fileCfg.ProjectRoot = stringPointer(normalized)
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
