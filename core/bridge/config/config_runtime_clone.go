package config

func cloneRuntimeConfig(raw runtimeConfig) runtimeConfig {
	return runtimeConfig{
		ProviderName:               raw.ProviderName,
		Provider:                   raw.Provider,
		APIKey:                     raw.APIKey,
		BaseURL:                    raw.BaseURL,
		Model:                      raw.Model,
		ChatPath:                   raw.ChatPath,
		NativePersistent:           raw.NativePersistent,
		ProjectRoot:                raw.ProjectRoot,
		ModelSelectionEnabled:      raw.ModelSelectionEnabled,
		ContextWindowTokens:        raw.ContextWindowTokens,
		ResponseReserveTokens:      raw.ResponseReserveTokens,
		ModelContextWindowTokens:   cloneModelTokenOverrides(raw.ModelContextWindowTokens),
		ModelResponseReserveTokens: cloneModelTokenOverrides(raw.ModelResponseReserveTokens),
		WebSearchTavilyURL:         raw.WebSearchTavilyURL,
		WebSearchExaURL:            raw.WebSearchExaURL,
		WebSearchTavilyAPIKey:      raw.WebSearchTavilyAPIKey,
		WebSearchExaAPIKey:         raw.WebSearchExaAPIKey,
		WebRooterEnabled:           raw.WebRooterEnabled,
		WebRooterBaseURL:           raw.WebRooterBaseURL,
		WebRooterAPIToken:          raw.WebRooterAPIToken,
		WebRooterTimeoutMS:         raw.WebRooterTimeoutMS,
		GraphQL:                    cloneGraphQLConfig(raw.GraphQL),
	}
}

func cloneGraphQLConfig(raw GraphQLConfig) GraphQLConfig {
	return GraphQLConfig{
		ToolRuntimeEnabled:  raw.ToolRuntimeEnabled,
		TextSanitizeEnabled: raw.TextSanitizeEnabled,
		DefaultSource:       raw.DefaultSource,
		Sources:             cloneGraphQLSources(raw.Sources),
		MutationPolicies:    cloneGraphQLMutationPolicies(raw.MutationPolicies),
	}
}

func cloneGraphQLSources(raw []GraphQLSourceConfig) []GraphQLSourceConfig {
	if raw == nil {
		return nil
	}

	out := make([]GraphQLSourceConfig, 0, len(raw))
	for _, source := range raw {
		out = append(out, GraphQLSourceConfig{
			Name:             source.Name,
			Description:      source.Description,
			Endpoint:         source.Endpoint,
			APIKey:           source.APIKey,
			SchemaPath:       source.SchemaPath,
			TimeoutMS:        source.TimeoutMS,
			MaxResponseBytes: source.MaxResponseBytes,
			Headers:          cloneStringMap(source.Headers),
			MaxDepth:         source.MaxDepth,
			MaxFields:        source.MaxFields,
			MaxRootFields:    source.MaxRootFields,
			MaxFragments:     source.MaxFragments,
			Domains:          cloneGraphQLDomains(source.Domains),
		})
	}
	return out
}

func cloneGraphQLDomains(raw []GraphQLDomainConfig) []GraphQLDomainConfig {
	if raw == nil {
		return nil
	}

	out := make([]GraphQLDomainConfig, 0, len(raw))
	for _, domain := range raw {
		out = append(out, GraphQLDomainConfig{
			Name:          domain.Name,
			Description:   domain.Description,
			RootQueries:   cloneStringSlice(domain.RootQueries),
			Types:         cloneStringSlice(domain.Types),
			MaxDepth:      domain.MaxDepth,
			MaxFields:     domain.MaxFields,
			MaxRootFields: domain.MaxRootFields,
		})
	}
	return out
}

func cloneGraphQLMutationPolicies(raw []GraphQLMutationPolicyConfig) []GraphQLMutationPolicyConfig {
	if raw == nil {
		return nil
	}

	out := make([]GraphQLMutationPolicyConfig, 0, len(raw))
	for _, policy := range raw {
		out = append(out, GraphQLMutationPolicyConfig{
			Name:                    policy.Name,
			Description:             policy.Description,
			Source:                  policy.Source,
			Domain:                  policy.Domain,
			RootMutation:            policy.RootMutation,
			ApprovalRequired:        policy.ApprovalRequired,
			IdempotencyMode:         policy.IdempotencyMode,
			IdempotencyHeader:       policy.IdempotencyHeader,
			IdempotencyVariablePath: policy.IdempotencyVariablePath,
			MaxDepth:                policy.MaxDepth,
			MaxFields:               policy.MaxFields,
			MaxRootFields:           policy.MaxRootFields,
			MaxFragments:            policy.MaxFragments,
		})
	}
	return out
}

func cloneStringSlice(raw []string) []string {
	if raw == nil {
		return nil
	}

	out := make([]string, len(raw))
	copy(out, raw)
	return out
}
