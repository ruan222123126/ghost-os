package config

type configUpdateState struct {
	configPath string
	current    runtimeConfig
	fileCfg    bridgeFileConfig
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
	if req.Model != nil && !current.ModelSelectionEnabled {
		return errModelSelectionDisabled
	}
	return nil
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
