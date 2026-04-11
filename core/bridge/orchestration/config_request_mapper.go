package orchestration

import (
	"strings"

	bridgeconfig "ghost-os/bridge/config"
)

func toolUpdateRequestToStoreRequest(name string, req toolUpdateRequest) bridgeconfig.ToolUpdateRequest {
	return bridgeconfig.ToolUpdateRequest{
		Name:           strings.TrimSpace(name),
		Enabled:        req.Enabled,
		PromptOverride: req.PromptOverride,
	}
}

func configUpdateRequestToStoreRequest(req configUpdateRequest) bridgeconfig.UpdateRequest {
	return bridgeconfig.UpdateRequest{
		Provider:                   req.Provider,
		APIKey:                     req.APIKey,
		BaseURL:                    req.BaseURL,
		Model:                      req.Model,
		ChatPath:                   req.ChatPath,
		GraphQLDefaultSource:       req.GraphqlDefaultSource,
		GraphQLToolRuntimeEnabled:  req.GraphqlToolRuntimeEnabled,
		GraphQLTextSanitizeEnabled: req.GraphqlTextSanitizeEnabled,
		GraphQLSources:             graphQLSourceInputs(req.GraphqlSources),
		GraphQLSourceUpsert:        graphQLSourceInputPointer(req.GraphqlSourceUpsert),
		GraphQLMutationPolicies:    graphQLMutationPolicyInputs(req.GraphqlMutationPolicies),
		SessionHumanLogFullEnabled: req.SessionHumanLogFullEnabled,
		WebRooterEnabled:           req.WebRooterEnabled,
		WebRooterBaseURL:           req.WebRooterBaseURL,
		WebRooterAPIToken:          req.WebRooterAPIToken,
		WebRooterTimeoutMS:         req.WebRooterTimeoutMs,
		WebSearchTavilyURL:         req.WebSearchTavilyURL,
		WebSearchExaURL:            req.WebSearchExaURL,
		WebSearchTavilyAPIKey:      req.WebSearchTavilyAPIKey,
		WebSearchExaAPIKey:         req.WebSearchExaAPIKey,
		TraceID:                    req.TraceID,
	}
}

func cloneOptionalStringPointer(raw *string) *string {
	if raw == nil {
		return nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil
	}
	return &value
}

func stringValue(raw *string) string {
	if raw == nil {
		return ""
	}
	return strings.TrimSpace(*raw)
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

func cloneModelTokenOverrides(raw map[string]int) map[string]int {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]int, len(raw))
	for key, value := range raw {
		out[key] = value
	}
	return out
}

func graphQLSourceInputs(raw []graphqlSourceInput) []bridgeconfig.GraphQLSourceInput {
	if raw == nil {
		return nil
	}
	out := make([]bridgeconfig.GraphQLSourceInput, 0, len(raw))
	for _, source := range raw {
		out = append(out, bridgeconfig.GraphQLSourceInput{
			Name:             source.Name,
			Description:      source.Description,
			Endpoint:         source.Endpoint,
			APIKey:           source.APIKey,
			SchemaPath:       source.SchemaPath,
			TimeoutMS:        source.TimeoutMs,
			MaxResponseBytes: source.MaxResponseBytes,
			Headers:          cloneStringMap(source.Headers),
			MaxDepth:         source.MaxDepth,
			MaxFields:        source.MaxFields,
			MaxRootFields:    source.MaxRootFields,
			MaxFragments:     source.MaxFragments,
			Domains:          graphQLDomainInputs(source.Domains),
		})
	}
	return out
}

func graphQLSourceInputPointer(raw graphqlSourceInput) *bridgeconfig.GraphQLSourceInput {
	if raw.Name == "" {
		return nil
	}
	value := bridgeconfig.GraphQLSourceInput{
		Name:             raw.Name,
		Description:      raw.Description,
		Endpoint:         raw.Endpoint,
		APIKey:           raw.APIKey,
		SchemaPath:       raw.SchemaPath,
		TimeoutMS:        raw.TimeoutMs,
		MaxResponseBytes: raw.MaxResponseBytes,
		Headers:          cloneStringMap(raw.Headers),
		MaxDepth:         raw.MaxDepth,
		MaxFields:        raw.MaxFields,
		MaxRootFields:    raw.MaxRootFields,
		MaxFragments:     raw.MaxFragments,
		Domains:          graphQLDomainInputs(raw.Domains),
	}
	return &value
}

func graphQLDomainInputs(raw []graphqlDomainInput) []bridgeconfig.GraphQLDomainInput {
	if len(raw) == 0 {
		return nil
	}
	out := make([]bridgeconfig.GraphQLDomainInput, 0, len(raw))
	for _, domain := range raw {
		out = append(out, bridgeconfig.GraphQLDomainInput{
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

func graphQLMutationPolicyInputs(raw []graphqlMutationPolicyInput) []bridgeconfig.GraphQLMutationPolicyInput {
	if len(raw) == 0 {
		return nil
	}
	out := make([]bridgeconfig.GraphQLMutationPolicyInput, 0, len(raw))
	for _, policy := range raw {
		out = append(out, bridgeconfig.GraphQLMutationPolicyInput{
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
