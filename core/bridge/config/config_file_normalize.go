package config

import "strings"

func normalizeBridgeFileConfigForWrite(cfg bridgeFileConfig) bridgeFileConfig {
	out := cfg
	normalizeBridgeScalarFields(&out)
	normalizeBridgeCollectionFields(&out)

	providers := materializeProvidersForWrite(out)
	out.Providers = providerConfigsToFileMap(providers)
	out.ActiveProvider = normalizedActiveProviderName(providers, out.ActiveProvider, out.ModelProvider)
	clearLegacyProviderFields(&out)
	return out
}

func hasLegacyProviderLayout(cfg bridgeFileConfig) bool {
	return cfg.ModelProvider != nil || len(cfg.ModelProviders) > 0 || cfg.Provider != nil || cfg.APIKey != nil || cfg.BaseURL != nil
}

func normalizedActiveProviderName(providers []providerConfig, preferred ...*string) *string {
	if len(providers) == 0 {
		return nil
	}
	for _, candidate := range preferred {
		name := strings.TrimSpace(stringValue(candidate))
		if providerIndexByName(providers, name) >= 0 {
			return stringPointer(name)
		}
	}
	return stringPointer(providers[0].Name)
}

func normalizeBridgeScalarFields(cfg *bridgeFileConfig) {
	cfg.ActiveProvider = cloneOptionalStringPointer(cfg.ActiveProvider)
	cfg.Model = cloneOptionalStringPointer(cfg.Model)
	cfg.ChatPath = cloneOptionalStringPointer(cfg.ChatPath)
	cfg.ProjectRoot = cloneOptionalStringPointer(cfg.ProjectRoot)
	cfg.WorkerModel = cloneOptionalStringPointer(cfg.WorkerModel)
	cfg.PromptsPath = cloneOptionalStringPointer(cfg.PromptsPath)
	cfg.PromptsDir = cloneOptionalStringPointer(cfg.PromptsDir)
	cfg.SessionsPath = cloneOptionalStringPointer(cfg.SessionsPath)
	cfg.RSSFeedsPath = cloneOptionalStringPointer(cfg.RSSFeedsPath)
	cfg.RSSInboxPath = cloneOptionalStringPointer(cfg.RSSInboxPath)
	cfg.RSSBriefingsPath = cloneOptionalStringPointer(cfg.RSSBriefingsPath)
	cfg.RSSReportsPath = cloneOptionalStringPointer(cfg.RSSReportsPath)
	cfg.RSSPollInterval = cloneOptionalStringPointer(cfg.RSSPollInterval)
	cfg.RSSBriefingInterval = cloneOptionalStringPointer(cfg.RSSBriefingInterval)
	cfg.GraphQLEndpoint = cloneOptionalStringPointer(cfg.GraphQLEndpoint)
	cfg.GraphQLAPIKey = cloneOptionalStringPointer(cfg.GraphQLAPIKey)
	cfg.GraphQLSchemaPath = cloneOptionalStringPointer(cfg.GraphQLSchemaPath)
	cfg.WebSearchTavilyAPIKey = cloneOptionalStringPointer(cfg.WebSearchTavilyAPIKey)
	cfg.WebSearchExaAPIKey = cloneOptionalStringPointer(cfg.WebSearchExaAPIKey)
	cfg.AnthropicVersion = cloneOptionalStringPointer(cfg.AnthropicVersion)
	cfg.ToolSelectorMode = cloneOptionalStringPointer(cfg.ToolSelectorMode)
	cfg.ToolSelectorModel = cloneOptionalStringPointer(cfg.ToolSelectorModel)
	cfg.MemoryAugmentationLLMModel = cloneOptionalStringPointer(cfg.MemoryAugmentationLLMModel)
	cfg.MemoryAugmentationUserScopeID = cloneOptionalStringPointer(cfg.MemoryAugmentationUserScopeID)
	cfg.BindAddr = cloneOptionalStringPointer(cfg.BindAddr)
	cfg.APIToken = cloneOptionalStringPointer(cfg.APIToken)
	cfg.NativeBinaryPath = cloneOptionalStringPointer(cfg.NativeBinaryPath)
}

func normalizeBridgeCollectionFields(cfg *bridgeFileConfig) {
	cfg.PromptsCoreFiles = normalizeConfiguredPathList(cfg.PromptsCoreFiles)
	cfg.NativeBinaryRoots = normalizeConfiguredPathList(cfg.NativeBinaryRoots)
	cfg.NativeBinaryCandidates = normalizeConfiguredPathList(cfg.NativeBinaryCandidates)
	cfg.NativeAllowedReadPaths = normalizeConfiguredPathList(cfg.NativeAllowedReadPaths)
	cfg.NativeAllowedWritePaths = normalizeConfiguredPathList(cfg.NativeAllowedWritePaths)
	cfg.GraphQLHeaders, _ = normalizeGraphQLHeaders(cfg.GraphQLHeaders)
	cfg.ProviderHeaders, _ = normalizeProviderHeaders(cfg.ProviderHeaders)
	cfg.CORSOrigins = normalizeOrigins(cfg.CORSOrigins)
	cfg.ToolAllowlist = normalizeConfiguredToolNames(cfg.ToolAllowlist)
	cfg.ToolBlocklist = normalizeConfiguredToolNames(cfg.ToolBlocklist)
}

func materializeProvidersForWrite(cfg bridgeFileConfig) []providerConfig {
	providers := normalizeProviderConfigs(cfg.Providers, stringValue(cfg.Model))
	if len(providers) > 0 {
		return providers
	}

	providers = normalizeLegacyProviderConfigs(cfg.ModelProviders, stringValue(cfg.Model))
	if len(providers) > 0 {
		return providers
	}
	return legacySingleProviderConfig(cfg)
}

func legacySingleProviderConfig(cfg bridgeFileConfig) []providerConfig {
	name := strings.TrimSpace(stringValue(cfg.Provider))
	baseURL := strings.TrimSpace(stringValue(cfg.BaseURL))
	apiKey := cloneOptionalStringPointer(cfg.APIKey)
	if name == "" && baseURL == "" && apiKey == nil {
		return nil
	}
	if name == "" {
		name = string(defaultProvider)
	}

	providerType := inferProviderType(name, baseURL, stringValue(cfg.Model))
	if providerType == "" {
		providerType = defaultProvider
	}
	if baseURL == "" {
		baseURL = defaultBaseURLForProvider(providerType)
	}
	return []providerConfig{{
		Name:    name,
		Type:    providerType,
		BaseURL: baseURL,
		APIKey:  apiKey,
	}}
}

func clearLegacyProviderFields(cfg *bridgeFileConfig) {
	cfg.ModelProvider = nil
	cfg.ModelProviders = nil
	cfg.Provider = nil
	cfg.APIKey = nil
	cfg.BaseURL = nil
}
