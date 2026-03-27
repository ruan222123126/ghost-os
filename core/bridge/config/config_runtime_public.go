package config

func snapshotFromRuntimeConfig(runtime runtimeConfig) Snapshot {
	runtime = normalizeRuntimeConfig(runtime)
	return Snapshot{
		Provider:                   activeProviderLabel(runtime),
		ProviderType:               string(runtime.Provider),
		BaseURL:                    runtime.BaseURL,
		Model:                      runtime.Model,
		ChatPath:                   runtime.ChatPath,
		APIKeySet:                  runtime.APIKey != "",
		ModelSelectionEnabled:      runtime.ModelSelectionEnabled,
		GraphQLDefaultSource:       runtime.GraphQL.DefaultSource,
		GraphQLToolRuntimeEnabled:  runtime.GraphQL.ToolRuntimeEnabled,
		GraphQLTextSanitizeEnabled: runtime.GraphQL.TextSanitizeEnabled,
		GraphQLSources:             graphQLSourceSnapshotsFromRuntime(runtime.GraphQL.Sources),
		GraphQLMutationPolicies:    graphQLMutationPolicySnapshots(runtime.GraphQL.MutationPolicies),
		WebRooterEnabled:           runtime.WebRooterEnabled,
		WebRooterAPITokenSet:       runtime.WebRooterAPIToken != "",
		WebSearchTavilyURL:         runtime.WebSearchTavilyURL,
		WebSearchExaURL:            runtime.WebSearchExaURL,
		WebSearchTavilyAPIKeySet:   runtime.WebSearchTavilyAPIKey != "",
		WebSearchExaAPIKeySet:      runtime.WebSearchExaAPIKey != "",
	}
}

func graphQLSourceSnapshotsFromRuntime(raw []GraphQLSourceConfig) []GraphQLSourceSnapshot {
	if len(raw) == 0 {
		return []GraphQLSourceSnapshot{}
	}

	out := make([]GraphQLSourceSnapshot, 0, len(raw))
	for _, source := range raw {
		out = append(out, GraphQLSourceSnapshot{
			Name:             source.Name,
			Description:      source.Description,
			Endpoint:         source.Endpoint,
			SchemaPath:       source.SchemaPath,
			TimeoutMS:        source.TimeoutMS,
			MaxResponseBytes: source.MaxResponseBytes,
			MaxDepth:         source.MaxDepth,
			MaxFields:        source.MaxFields,
			MaxRootFields:    source.MaxRootFields,
			MaxFragments:     source.MaxFragments,
			Headers:          cloneStringMap(source.Headers),
			APIKeySet:        source.APIKey != "",
			Domains:          graphQLDomainSnapshotsFromRuntime(source.Domains),
		})
	}
	return out
}

func graphQLDomainSnapshotsFromRuntime(raw []GraphQLDomainConfig) []GraphQLDomainSnapshot {
	if len(raw) == 0 {
		return nil
	}

	out := make([]GraphQLDomainSnapshot, 0, len(raw))
	for _, domain := range raw {
		out = append(out, GraphQLDomainSnapshot{
			Name:          domain.Name,
			Description:   domain.Description,
			RootQueries:   append([]string(nil), domain.RootQueries...),
			Types:         append([]string(nil), domain.Types...),
			MaxDepth:      domain.MaxDepth,
			MaxFields:     domain.MaxFields,
			MaxRootFields: domain.MaxRootFields,
		})
	}
	return out
}
