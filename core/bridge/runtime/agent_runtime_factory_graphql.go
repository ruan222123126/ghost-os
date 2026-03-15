package runtime

import (
	"fmt"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/tools"
)

func registerOptionalGraphQLTools(registry *tools.Registry, cfg Config) error {
	if registry == nil {
		return fmt.Errorf("tool registry is not configured")
	}
	if len(cfg.GraphQL.Sources) == 0 {
		return nil
	}

	graphQLRegistry, err := tools.NewGraphQLSourceRegistry(graphQLRegistryConfig(cfg))
	if err != nil {
		return fmt.Errorf("build graphql source registry: %w", err)
	}
	registry.Register(tools.NewGraphQLQueryTool(graphQLRegistry))
	registry.Register(tools.NewGraphQLSchemaLookupTool(graphQLRegistry))
	if graphQLRegistry.HasMutationPolicies() {
		registry.Register(tools.NewGraphQLMutationTool(graphQLRegistry))
	}
	return nil
}

func graphQLRegistryConfig(cfg Config) tools.GraphQLRegistryConfig {
	sources := make([]tools.GraphQLSourceConfig, 0, len(cfg.GraphQL.Sources))
	for _, source := range cfg.GraphQL.Sources {
		sources = append(sources, tools.GraphQLSourceConfig{
			Name:             source.Name,
			Description:      source.Description,
			Endpoint:         source.Endpoint,
			APIKey:           source.APIKey,
			SchemaPath:       source.SchemaPath,
			TimeoutMS:        source.TimeoutMS,
			MaxResponseBytes: source.MaxResponseBytes,
			Headers:          cloneRuntimeHeaders(source.Headers),
			MaxDepth:         source.MaxDepth,
			MaxFields:        source.MaxFields,
			MaxRootFields:    source.MaxRootFields,
			MaxFragments:     source.MaxFragments,
			Domains:          graphQLDomainConfigs(source.Domains),
		})
	}
	return tools.GraphQLRegistryConfig{
		DefaultSource:    cfg.GraphQL.DefaultSource,
		Sources:          sources,
		MutationPolicies: graphQLMutationPolicyConfigs(cfg.GraphQL.MutationPolicies),
	}
}

func graphQLDomainConfigs(raw []bridgeconfig.GraphQLDomainConfig) []tools.GraphQLDomainConfig {
	if len(raw) == 0 {
		return nil
	}

	out := make([]tools.GraphQLDomainConfig, 0, len(raw))
	for _, domain := range raw {
		out = append(out, tools.GraphQLDomainConfig{
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

func cloneRuntimeHeaders(raw map[string]string) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]string, len(raw))
	for key, value := range raw {
		out[key] = value
	}
	return out
}

func graphQLMutationPolicyConfigs(
	raw []bridgeconfig.GraphQLMutationPolicyConfig,
) []tools.GraphQLMutationPolicyConfig {
	if len(raw) == 0 {
		return nil
	}

	out := make([]tools.GraphQLMutationPolicyConfig, 0, len(raw))
	for _, policy := range raw {
		out = append(out, tools.GraphQLMutationPolicyConfig{
			Name:          policy.Name,
			Description:   policy.Description,
			Source:        policy.Source,
			Domain:        policy.Domain,
			RootMutation:  policy.RootMutation,
			MaxDepth:      policy.MaxDepth,
			MaxFields:     policy.MaxFields,
			MaxRootFields: policy.MaxRootFields,
			MaxFragments:  policy.MaxFragments,
		})
	}
	return out
}
