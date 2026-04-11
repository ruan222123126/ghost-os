package orchestration

import bridgeconfig "ghost-os/bridge/config"

func configResponseFromSnapshot(snapshot bridgeconfig.Snapshot) configResponse {
	return configResponse{
		Provider:                   snapshot.Provider,
		ProviderType:               snapshot.ProviderType,
		BaseURL:                    snapshot.BaseURL,
		Model:                      snapshot.Model,
		ChatPath:                   snapshot.ChatPath,
		APIKeySet:                  snapshot.APIKeySet,
		ModelSelectionEnabled:      snapshot.ModelSelectionEnabled,
		GraphqlDefaultSource:       snapshot.GraphQLDefaultSource,
		GraphqlToolRuntimeEnabled:  snapshot.GraphQLToolRuntimeEnabled,
		GraphqlTextSanitizeEnabled: snapshot.GraphQLTextSanitizeEnabled,
		GraphqlSources:             graphQLSourceResponses(snapshot.GraphQLSources),
		GraphqlMutationPolicies:    graphQLMutationPolicyResponses(snapshot.GraphQLMutationPolicies),
		SessionHumanLogFullEnabled: snapshot.SessionHumanLogFullEnabled,
		WebRooterEnabled:           snapshot.WebRooterEnabled,
		WebRooterBaseURL:           snapshot.WebRooterBaseURL,
		WebRooterTimeoutMs:         snapshot.WebRooterTimeoutMS,
		WebRooterAPITokenSet:       snapshot.WebRooterAPITokenSet,
		WebSearchTavilyURL:         snapshot.WebSearchTavilyURL,
		WebSearchExaURL:            snapshot.WebSearchExaURL,
		WebSearchTavilyAPIKeySet:   snapshot.WebSearchTavilyAPIKeySet,
		WebSearchExaAPIKeySet:      snapshot.WebSearchExaAPIKeySet,
	}
}

func graphQLSourceResponses(raw []bridgeconfig.GraphQLSourceSnapshot) []graphqlSourceResponse {
	if len(raw) == 0 {
		return []graphqlSourceResponse{}
	}

	out := make([]graphqlSourceResponse, 0, len(raw))
	for _, source := range raw {
		out = append(out, graphqlSourceResponse{
			Name:             source.Name,
			Description:      source.Description,
			Endpoint:         source.Endpoint,
			SchemaPath:       source.SchemaPath,
			TimeoutMs:        source.TimeoutMS,
			MaxResponseBytes: source.MaxResponseBytes,
			MaxDepth:         source.MaxDepth,
			MaxFields:        source.MaxFields,
			MaxRootFields:    source.MaxRootFields,
			MaxFragments:     source.MaxFragments,
			Headers:          cloneStringMap(source.Headers),
			APIKeySet:        source.APIKeySet,
			Domains:          graphQLDomainResponses(source.Domains),
		})
	}
	return out
}

func graphQLDomainResponses(raw []bridgeconfig.GraphQLDomainSnapshot) []graphqlDomainResponse {
	if len(raw) == 0 {
		return nil
	}

	out := make([]graphqlDomainResponse, 0, len(raw))
	for _, domain := range raw {
		out = append(out, graphqlDomainResponse{
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

func graphQLMutationPolicyResponses(
	raw []bridgeconfig.GraphQLMutationPolicySnapshot,
) []graphqlMutationPolicyResponse {
	if len(raw) == 0 {
		return []graphqlMutationPolicyResponse{}
	}

	out := make([]graphqlMutationPolicyResponse, 0, len(raw))
	for _, policy := range raw {
		out = append(out, graphqlMutationPolicyResponse{
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
