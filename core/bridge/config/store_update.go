package config

import (
	"ghost-os/bridge/config/internal/providers"
	configruntime "ghost-os/bridge/config/internal/runtime"
)

type configUpdateState struct {
	configPath string
	current    runtimeConfig
	fileCfg    bridgeFileConfig
}

// Update 按固定事务执行：load current file -> validate -> apply patches -> resolve runtime -> write -> replace snapshot。
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

func (s *store) loadUpdateStateLocked(req UpdateRequest) (configUpdateState, error) {
	fileCfg, configPath, err := s.loadStoredFileConfigLocked()
	if err != nil {
		return configUpdateState{}, err
	}
	current, err := resolveRuntimeConfigWithFallback(fileCfg, s.runtime)
	if err != nil {
		return configUpdateState{}, err
	}
	if err := validateConfigUpdateRequest(req, current); err != nil {
		return configUpdateState{}, err
	}
	updateFileCfg, err := loadCurrentUpdateFileConfig(fileCfg, current, req)
	if err != nil {
		return configUpdateState{}, err
	}
	return configUpdateState{
		configPath: configPath,
		current:    current,
		fileCfg:    updateFileCfg,
	}, nil
}

func validateConfigUpdateRequest(req UpdateRequest, current runtimeConfig) error {
	return configruntime.ValidateUpdate(configruntime.UpdateRequest{Model: req.Model}, current)
}

func loadCurrentUpdateFileConfig(
	fileCfg bridgeFileConfig,
	current runtimeConfig,
	req UpdateRequest,
) (bridgeFileConfig, error) {
	out, err := normalizeBridgeFileConfigForWrite(fileCfg)
	if err != nil {
		return bridgeFileConfig{}, err
	}
	prepareProviderUpdateBase(&out, current, req)
	prepareWebSearchUpdateBase(&out, current, req)
	return normalizeBridgeFileConfigForWrite(out)
}

func prepareProviderUpdateBase(
	fileCfg *bridgeFileConfig,
	current runtimeConfig,
	req UpdateRequest,
) {
	if fileCfg == nil || !touchesProviderUpdate(req) {
		return
	}
	prepareActiveProviderUpdateBase(fileCfg, current, req.Provider)
}

func touchesProviderUpdate(req UpdateRequest) bool {
	return req.Provider != nil || req.BaseURL != nil || req.APIKey != nil
}

func prepareWebSearchUpdateBase(
	fileCfg *bridgeFileConfig,
	current runtimeConfig,
	req UpdateRequest,
) {
	if fileCfg == nil || !touchesWebSearchUpdate(req) {
		return
	}
	if fileCfg.WebSearchTavilyURL == nil {
		fileCfg.WebSearchTavilyURL = optionalStringPointer(current.WebSearchTavilyURL)
	}
	if fileCfg.WebSearchExaURL == nil {
		fileCfg.WebSearchExaURL = optionalStringPointer(current.WebSearchExaURL)
	}
	if fileCfg.WebSearchTavilyAPIKey == nil {
		fileCfg.WebSearchTavilyAPIKey = optionalStringPointer(current.WebSearchTavilyAPIKey)
	}
	if fileCfg.WebSearchExaAPIKey == nil {
		fileCfg.WebSearchExaAPIKey = optionalStringPointer(current.WebSearchExaAPIKey)
	}
}

func touchesWebSearchUpdate(req UpdateRequest) bool {
	return req.WebSearchTavilyURL != nil ||
		req.WebSearchExaURL != nil ||
		req.WebSearchTavilyAPIKey != nil ||
		req.WebSearchExaAPIKey != nil
}

func updatePersistFallback(current runtimeConfig, req UpdateRequest) (runtimeConfig, error) {
	return configruntime.PersistFallback(current, configruntime.PersistFallbackRequest{
		Model:              req.Model,
		ChatPath:           req.ChatPath,
		ProjectRoot:        req.ProjectRoot,
		WebSearchTavilyURL: req.WebSearchTavilyURL,
		WebSearchExaURL:    req.WebSearchExaURL,
	}, currentEnv())
}

func applyConfigUpdatePatch(
	fileCfg *bridgeFileConfig,
	current runtimeConfig,
	req UpdateRequest,
) error {
	if err := applyProviderUpdatePatch(fileCfg, current, req); err != nil {
		return err
	}
	if err := applyRuntimeUpdatePatch(fileCfg, req); err != nil {
		return err
	}
	applyWebSearchUpdatePatch(fileCfg, req)
	applyDisplayUpdatePatch(fileCfg, req)
	applyTaskUpdatePatch(fileCfg, req)
	return nil
}

func applyProviderUpdatePatch(
	fileCfg *bridgeFileConfig,
	current runtimeConfig,
	req UpdateRequest,
) error {
	patch, err := providers.ApplyRuntimePatch(
		providerStateFromFileConfig(*fileCfg),
		providerRuntimeSnapshot(current),
		providers.RuntimePatchRequest{
			Provider: req.Provider,
			APIKey:   req.APIKey,
			BaseURL:  req.BaseURL,
		},
	)
	if err != nil {
		return err
	}
	applyProviderPatchToFileConfig(fileCfg, patch)
	return nil
}

func prepareActiveProviderUpdateBase(
	fileCfg *bridgeFileConfig,
	current runtimeConfig,
	requestedName *string,
) {
	if fileCfg == nil {
		return
	}
	patch := providers.PrepareRuntimePatchBase(
		providerStateFromFileConfig(*fileCfg),
		providerRuntimeSnapshot(current),
		requestedName,
	)
	applyProviderPatchToFileConfig(fileCfg, patch)
}

func applyRuntimeUpdatePatch(fileCfg *bridgeFileConfig, req UpdateRequest) error {
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
	if req.LLMCompletionRetryCount != nil {
		fileCfg.LLMCompletionRetryCount = cloneIntPointer(req.LLMCompletionRetryCount)
	}
	if req.LLMCompletionRetryIntervalMS != nil {
		fileCfg.LLMCompletionRetryIntervalMS = cloneIntPointer(req.LLMCompletionRetryIntervalMS)
	}
	if req.SessionTitleMode != nil {
		fileCfg.SessionTitleMode = cloneOptionalStringPointer(req.SessionTitleMode)
	}
	return nil
}

func applyWebSearchUpdatePatch(fileCfg *bridgeFileConfig, req UpdateRequest) {
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
}

func applyDisplayUpdatePatch(fileCfg *bridgeFileConfig, req UpdateRequest) {
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
}

func applyTaskUpdatePatch(fileCfg *bridgeFileConfig, req UpdateRequest) {
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
