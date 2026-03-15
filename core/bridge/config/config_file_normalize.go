package config

import "strings"

func normalizeBridgeFileConfigForWrite(cfg bridgeFileConfig) bridgeFileConfig {
	out := cfg
	normalizeBridgeScalarFields(&out)
	normalizeBridgeCollectionFields(&out)
	providers := normalizeProviderConfigs(out.Providers, stringValue(out.Model))
	out.Providers = providerConfigsToFileMap(providers)
	out.ActiveProvider = normalizedActiveProviderName(providers, out.ActiveProvider)
	return out
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
	cfg.GraphQLDefaultSource = cloneOptionalStringPointer(cfg.GraphQLDefaultSource)
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
	cfg.PromptsRuntimeConstraintFiles = normalizeConfiguredPathList(cfg.PromptsRuntimeConstraintFiles)
	cfg.PromptsResponseRuleFiles = normalizeConfiguredPathList(cfg.PromptsResponseRuleFiles)
	cfg.NativeBinaryRoots = normalizeConfiguredPathList(cfg.NativeBinaryRoots)
	cfg.NativeBinaryCandidates = normalizeConfiguredPathList(cfg.NativeBinaryCandidates)
	cfg.NativeAllowedReadPaths = normalizeConfiguredPathList(cfg.NativeAllowedReadPaths)
	cfg.NativeAllowedWritePaths = normalizeConfiguredPathList(cfg.NativeAllowedWritePaths)
	cfg.GraphQLSources = normalizeGraphQLSourceFileConfigs(cfg.GraphQLSources)
	cfg.GraphQLMutationPolicies = normalizeGraphQLMutationPolicyFileConfigs(cfg.GraphQLMutationPolicies)
	cfg.ProviderHeaders, _ = normalizeProviderHeaders(cfg.ProviderHeaders)
	cfg.CORSOrigins = normalizeOrigins(cfg.CORSOrigins)
	cfg.ToolAllowlist = normalizeConfiguredToolNames(cfg.ToolAllowlist)
	cfg.ToolBlocklist = normalizeConfiguredToolNames(cfg.ToolBlocklist)
}

func hasGraphQLSourceLayout(cfg bridgeFileConfig) bool {
	return cfg.GraphQLDefaultSource != nil ||
		len(cfg.GraphQLSources) > 0 ||
		len(cfg.GraphQLMutationPolicies) > 0
}

func normalizeGraphQLSourceFileConfigs(raw []graphQLSourceFileConfig) []graphQLSourceFileConfig {
	if len(raw) == 0 {
		return nil
	}

	out := make([]graphQLSourceFileConfig, 0, len(raw))
	for _, source := range raw {
		out = append(out, graphQLSourceFileConfig{
			Name:             strings.TrimSpace(source.Name),
			Description:      strings.TrimSpace(source.Description),
			Endpoint:         strings.TrimSpace(source.Endpoint),
			APIKey:           cloneOptionalStringPointer(source.APIKey),
			SchemaPath:       strings.TrimSpace(source.SchemaPath),
			TimeoutMS:        source.TimeoutMS,
			MaxResponseBytes: source.MaxResponseBytes,
			Headers:          cloneStringMap(source.Headers),
			MaxDepth:         source.MaxDepth,
			MaxFields:        source.MaxFields,
			MaxRootFields:    source.MaxRootFields,
			MaxFragments:     source.MaxFragments,
			Domains:          normalizeGraphQLDomainFileConfigs(source.Domains),
		})
	}
	return out
}

func normalizeGraphQLDomainFileConfigs(raw []graphQLDomainFileConfig) []graphQLDomainFileConfig {
	if len(raw) == 0 {
		return nil
	}

	out := make([]graphQLDomainFileConfig, 0, len(raw))
	for _, domain := range raw {
		out = append(out, graphQLDomainFileConfig{
			Name:          strings.TrimSpace(domain.Name),
			Description:   strings.TrimSpace(domain.Description),
			RootQueries:   normalizeConfiguredToolNames(domain.RootQueries),
			Types:         normalizeConfiguredToolNames(domain.Types),
			MaxDepth:      domain.MaxDepth,
			MaxFields:     domain.MaxFields,
			MaxRootFields: domain.MaxRootFields,
		})
	}
	return out
}
