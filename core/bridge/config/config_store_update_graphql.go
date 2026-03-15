package config

import "fmt"

func materializeRuntimeGraphQLIfNeeded(
	fileCfg *bridgeFileConfig,
	current runtimeConfig,
	req configUpdateRequest,
) {
	if fileCfg == nil || !requiresRuntimeGraphQLMaterialization(req) || hasGraphQLSourceLayout(*fileCfg) {
		return
	}
	fileCfg.GraphQLDefaultSource = optionalStringPointer(current.GraphQL.DefaultSource)
	fileCfg.GraphQLSources = graphQLSourcesToFileConfigs(current.GraphQL.Sources)
	fileCfg.GraphQLMutationPolicies = graphQLMutationPoliciesToFileConfigs(
		current.GraphQL.MutationPolicies,
	)
}

func requiresRuntimeGraphQLMaterialization(req configUpdateRequest) bool {
	return req.GraphQLDefaultSource != nil ||
		req.GraphQLSources != nil ||
		req.GraphQLSourceUpsert != nil ||
		req.GraphQLMutationPolicies != nil
}

func applyGraphQLRuntimeFields(fileCfg *bridgeFileConfig, req configUpdateRequest) error {
	if fileCfg == nil || !requiresRuntimeGraphQLMaterialization(req) {
		return nil
	}
	if req.GraphQLSources != nil && req.GraphQLSourceUpsert != nil {
		return fmt.Errorf("graphql_sources and graphql_source_upsert cannot be used together")
	}
	if req.GraphQLDefaultSource != nil {
		fileCfg.GraphQLDefaultSource = cloneOptionalStringPointer(req.GraphQLDefaultSource)
	}
	if req.GraphQLMutationPolicies != nil {
		fileCfg.GraphQLMutationPolicies = graphQLMutationPolicyInputsToFileConfigs(
			req.GraphQLMutationPolicies,
		)
	}
	if req.GraphQLSources != nil {
		sources, err := graphQLSourceInputsToFileConfigs(req.GraphQLSources)
		if err != nil {
			return err
		}
		fileCfg.GraphQLSources = sources
		return nil
	}
	if req.GraphQLSourceUpsert != nil {
		sources, err := upsertGraphQLSourceFileConfig(fileCfg.GraphQLSources, *req.GraphQLSourceUpsert)
		if err != nil {
			return err
		}
		fileCfg.GraphQLSources = sources
	}
	return nil
}

func graphQLSourcesToFileConfigs(raw []GraphQLSourceConfig) []graphQLSourceFileConfig {
	if len(raw) == 0 {
		return nil
	}

	out := make([]graphQLSourceFileConfig, 0, len(raw))
	for _, source := range raw {
		out = append(out, graphQLSourceFileConfig{
			Name:             source.Name,
			Description:      source.Description,
			Endpoint:         source.Endpoint,
			APIKey:           optionalStringPointer(source.APIKey),
			SchemaPath:       source.SchemaPath,
			TimeoutMS:        source.TimeoutMS,
			MaxResponseBytes: source.MaxResponseBytes,
			Headers:          cloneStringMap(source.Headers),
			MaxDepth:         source.MaxDepth,
			MaxFields:        source.MaxFields,
			MaxRootFields:    source.MaxRootFields,
			MaxFragments:     source.MaxFragments,
			Domains:          graphQLDomainsToFileConfigs(source.Domains),
		})
	}
	return out
}

func graphQLDomainsToFileConfigs(raw []GraphQLDomainConfig) []graphQLDomainFileConfig {
	if len(raw) == 0 {
		return nil
	}

	out := make([]graphQLDomainFileConfig, 0, len(raw))
	for _, domain := range raw {
		out = append(out, graphQLDomainFileConfig{
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

func graphQLSourceInputsToFileConfigs(raw []GraphQLSourceInput) ([]graphQLSourceFileConfig, error) {
	if raw == nil {
		return nil, nil
	}

	out := make([]graphQLSourceFileConfig, 0, len(raw))
	for _, source := range raw {
		encoded, err := graphQLSourceInputToFileConfig(source, nil)
		if err != nil {
			return nil, err
		}
		out = append(out, encoded)
	}
	return out, nil
}

func upsertGraphQLSourceFileConfig(
	existing []graphQLSourceFileConfig,
	input GraphQLSourceInput,
) ([]graphQLSourceFileConfig, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("graphql_source_upsert.name is required")
	}

	out := append([]graphQLSourceFileConfig(nil), existing...)
	for index, source := range out {
		if source.Name != input.Name {
			continue
		}
		updated, err := graphQLSourceInputToFileConfig(input, &source)
		if err != nil {
			return nil, err
		}
		out[index] = updated
		return out, nil
	}

	created, err := graphQLSourceInputToFileConfig(input, nil)
	if err != nil {
		return nil, err
	}
	return append(out, created), nil
}

func graphQLSourceInputToFileConfig(
	input GraphQLSourceInput,
	existing *graphQLSourceFileConfig,
) (graphQLSourceFileConfig, error) {
	if input.Name == "" {
		return graphQLSourceFileConfig{}, fmt.Errorf("graphql source name is required")
	}
	headers, err := normalizeGraphQLHeaders(input.Headers)
	if err != nil {
		return graphQLSourceFileConfig{}, err
	}
	source := graphQLSourceFileConfig{
		Name:             input.Name,
		Description:      input.Description,
		Endpoint:         input.Endpoint,
		APIKey:           cloneOptionalStringPointer(input.APIKey),
		SchemaPath:       input.SchemaPath,
		TimeoutMS:        input.TimeoutMS,
		MaxResponseBytes: input.MaxResponseBytes,
		Headers:          headers,
		MaxDepth:         input.MaxDepth,
		MaxFields:        input.MaxFields,
		MaxRootFields:    input.MaxRootFields,
		MaxFragments:     input.MaxFragments,
		Domains:          graphQLDomainInputsToFileConfigs(input.Domains),
	}
	if source.APIKey == nil && existing != nil {
		source.APIKey = cloneOptionalStringPointer(existing.APIKey)
	}
	return source, nil
}

func graphQLDomainInputsToFileConfigs(raw []GraphQLDomainInput) []graphQLDomainFileConfig {
	if len(raw) == 0 {
		return nil
	}

	out := make([]graphQLDomainFileConfig, 0, len(raw))
	for _, domain := range raw {
		out = append(out, graphQLDomainFileConfig{
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

func cloneStringMap(raw map[string]string) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]string, len(raw))
	for key, value := range raw {
		out[key] = value
	}
	return out
}

func optionalIntPointer(value int) *int {
	v := value
	return &v
}

func boolPointer(value bool) *bool {
	v := value
	return &v
}
