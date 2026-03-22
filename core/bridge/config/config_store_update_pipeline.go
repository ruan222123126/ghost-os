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
	prepareGraphQLUpdateBase(&out, current, req)
	prepareWebSearchUpdateBase(&out, current, req)
	prepareWebRooterUpdateBase(&out, current, req)
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

func prepareGraphQLUpdateBase(
	fileCfg *bridgeFileConfig,
	current runtimeConfig,
	req UpdateRequest,
) {
	if fileCfg == nil || !touchesGraphQLUpdate(req) || hasGraphQLSourceLayout(*fileCfg) {
		return
	}
	if !hasRuntimeGraphQLConfig(current.GraphQL) {
		return
	}
	fileCfg.GraphQLToolRuntimeEnabled = boolPointer(current.GraphQL.ToolRuntimeEnabled)
	fileCfg.GraphQLDefaultSource = optionalStringPointer(current.GraphQL.DefaultSource)
	fileCfg.GraphQLSources = graphQLSourcesToFileConfigs(current.GraphQL.Sources)
	fileCfg.GraphQLMutationPolicies = graphQLMutationPoliciesToFileConfigs(
		current.GraphQL.MutationPolicies,
	)
}

func touchesGraphQLUpdate(req UpdateRequest) bool {
	return req.GraphQLDefaultSource != nil ||
		req.GraphQLToolRuntimeEnabled != nil ||
		req.GraphQLSources != nil ||
		req.GraphQLSourceUpsert != nil ||
		req.GraphQLMutationPolicies != nil
}

func hasRuntimeGraphQLConfig(cfg GraphQLConfig) bool {
	return cfg.ToolRuntimeEnabled ||
		cfg.DefaultSource != "" ||
		len(cfg.Sources) > 0 ||
		len(cfg.MutationPolicies) > 0
}

func prepareWebSearchUpdateBase(
	fileCfg *bridgeFileConfig,
	current runtimeConfig,
	req UpdateRequest,
) {
	if fileCfg == nil || !touchesWebSearchUpdate(req) {
		return
	}
	if fileCfg.WebSearchTavilyAPIKey == nil {
		fileCfg.WebSearchTavilyAPIKey = optionalStringPointer(current.WebSearchTavilyAPIKey)
	}
	if fileCfg.WebSearchExaAPIKey == nil {
		fileCfg.WebSearchExaAPIKey = optionalStringPointer(current.WebSearchExaAPIKey)
	}
}

func touchesWebSearchUpdate(req UpdateRequest) bool {
	return req.WebSearchTavilyAPIKey != nil || req.WebSearchExaAPIKey != nil
}

func prepareWebRooterUpdateBase(
	fileCfg *bridgeFileConfig,
	current runtimeConfig,
	req UpdateRequest,
) {
	if fileCfg == nil || !touchesWebRooterUpdate(req) {
		return
	}
	if fileCfg.WebRooterEnabled == nil {
		fileCfg.WebRooterEnabled = cloneBoolPointer(&current.WebRooterEnabled)
	}
	if fileCfg.WebRooterBaseURL == nil {
		fileCfg.WebRooterBaseURL = stringPointer(current.WebRooterBaseURL)
	}
	if fileCfg.WebRooterAPIToken == nil {
		fileCfg.WebRooterAPIToken = optionalStringPointer(current.WebRooterAPIToken)
	}
	if fileCfg.WebRooterTimeoutMS == nil {
		fileCfg.WebRooterTimeoutMS = cloneIntPointer(&current.WebRooterTimeoutMS)
	}
}

func touchesWebRooterUpdate(req UpdateRequest) bool {
	return req.WebRooterEnabled != nil ||
		req.WebRooterBaseURL != nil ||
		req.WebRooterAPIToken != nil ||
		req.WebRooterTimeoutMS != nil
}
