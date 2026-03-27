package config

import (
	"fmt"
	"strings"
)

func normalizeBridgeFileConfigForWrite(cfg bridgeFileConfig) (bridgeFileConfig, error) {
	out := cfg
	normalizeBridgeScalarFields(&out)
	if err := normalizeBridgeCollectionFields(&out); err != nil {
		return bridgeFileConfig{}, err
	}
	providers := normalizeProviderConfigs(out.Providers, stringValue(out.Model))
	out.Providers = providerConfigsToFileMap(providers)
	out.ActiveProvider = normalizedActiveProviderName(providers, out.ActiveProvider)
	return out, nil
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
	cfg.TasksPath = cloneOptionalStringPointer(cfg.TasksPath)
	cfg.SessionsPath = cloneOptionalStringPointer(cfg.SessionsPath)
	cfg.RSSFeedsPath = cloneOptionalStringPointer(cfg.RSSFeedsPath)
	cfg.RSSInboxPath = cloneOptionalStringPointer(cfg.RSSInboxPath)
	cfg.RSSBriefingsPath = cloneOptionalStringPointer(cfg.RSSBriefingsPath)
	cfg.RSSReportsPath = cloneOptionalStringPointer(cfg.RSSReportsPath)
	cfg.WebRooterEnabled = cloneBoolPointer(cfg.WebRooterEnabled)
	cfg.WebRooterBaseURL = cloneStringPointer(cfg.WebRooterBaseURL)
	cfg.WebRooterAPIToken = cloneOptionalStringPointer(cfg.WebRooterAPIToken)
	cfg.WebRooterTimeoutMS = cloneIntPointer(cfg.WebRooterTimeoutMS)
	cfg.RSSPollInterval = cloneOptionalStringPointer(cfg.RSSPollInterval)
	cfg.RSSBriefingInterval = cloneOptionalStringPointer(cfg.RSSBriefingInterval)
	cfg.GraphQLDefaultSource = cloneOptionalStringPointer(cfg.GraphQLDefaultSource)
	cfg.GraphQLToolRuntimeEnabled = cloneBoolPointer(cfg.GraphQLToolRuntimeEnabled)
	cfg.GraphQLTextSanitizeEnabled = cloneBoolPointer(cfg.GraphQLTextSanitizeEnabled)
	cfg.WebSearchTavilyURL = cloneOptionalStringPointer(cfg.WebSearchTavilyURL)
	cfg.WebSearchExaURL = cloneOptionalStringPointer(cfg.WebSearchExaURL)
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

func normalizeBridgeCollectionFields(cfg *bridgeFileConfig) error {
	cfg.PromptsCoreFiles = normalizeConfiguredPathList(cfg.PromptsCoreFiles)
	cfg.PromptsRuntimeConstraintFiles = normalizeConfiguredPathList(cfg.PromptsRuntimeConstraintFiles)
	cfg.PromptsResponseRuleFiles = normalizeConfiguredPathList(cfg.PromptsResponseRuleFiles)
	cfg.NativeBinaryRoots = normalizeConfiguredPathList(cfg.NativeBinaryRoots)
	cfg.NativeBinaryCandidates = normalizeConfiguredPathList(cfg.NativeBinaryCandidates)
	cfg.NativeAllowedReadPaths = normalizeConfiguredPathList(cfg.NativeAllowedReadPaths)
	cfg.NativeAllowedWritePaths = normalizeConfiguredPathList(cfg.NativeAllowedWritePaths)
	sources, err := normalizeGraphQLSourceFileConfigs(cfg.GraphQLSources)
	if err != nil {
		return err
	}
	cfg.GraphQLSources = sources
	cfg.GraphQLMutationPolicies = normalizeGraphQLMutationPolicyFileConfigs(cfg.GraphQLMutationPolicies)
	providerHeaders, err := normalizeProviderHeaders(cfg.ProviderHeaders)
	if err != nil {
		return err
	}
	cfg.ProviderHeaders = providerHeaders
	cfg.CORSOrigins = normalizeOrigins(cfg.CORSOrigins)
	cfg.ToolAllowlist = normalizeConfiguredToolNames(cfg.ToolAllowlist)
	cfg.ToolBlocklist = normalizeConfiguredToolNames(cfg.ToolBlocklist)
	cfg.WorkflowToolAllowlist = normalizeConfiguredToolNames(cfg.WorkflowToolAllowlist)
	return nil
}

func hasGraphQLSourceLayout(cfg bridgeFileConfig) bool {
	return cfg.GraphQLToolRuntimeEnabled != nil ||
		cfg.GraphQLTextSanitizeEnabled != nil ||
		cfg.GraphQLDefaultSource != nil ||
		len(cfg.GraphQLSources) > 0 ||
		len(cfg.GraphQLMutationPolicies) > 0
}

func normalizeGraphQLSourceFileConfigs(raw []graphQLSourceFileConfig) ([]graphQLSourceFileConfig, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	out := make([]graphQLSourceFileConfig, 0, len(raw))
	for _, source := range raw {
		normalized, err := normalizeGraphQLSourceFileConfig(source)
		if err != nil {
			return nil, err
		}
		out = append(out, normalized)
	}
	return out, nil
}

func normalizeGraphQLSourceFileConfig(source graphQLSourceFileConfig) (graphQLSourceFileConfig, error) {
	headers, err := normalizeGraphQLHeaders(source.Headers)
	if err != nil {
		return graphQLSourceFileConfig{}, fmt.Errorf(
			"invalid graphql source %q headers: %w",
			strings.TrimSpace(source.Name),
			err,
		)
	}
	return graphQLSourceFileConfig{
		Name:             strings.TrimSpace(source.Name),
		Description:      strings.TrimSpace(source.Description),
		Endpoint:         strings.TrimSpace(source.Endpoint),
		APIKey:           cloneOptionalStringPointer(source.APIKey),
		SchemaPath:       strings.TrimSpace(source.SchemaPath),
		TimeoutMS:        source.TimeoutMS,
		MaxResponseBytes: source.MaxResponseBytes,
		Headers:          headers,
		MaxDepth:         source.MaxDepth,
		MaxFields:        source.MaxFields,
		MaxRootFields:    source.MaxRootFields,
		MaxFragments:     source.MaxFragments,
		Domains:          normalizeGraphQLDomainFileConfigs(source.Domains),
	}, nil
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
