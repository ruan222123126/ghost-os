package tools

import (
	"fmt"
	"strings"

	"ghost-os/bridge/tools/internal/graphqlschema"
)

type GraphQLDomainConfig struct {
	Name          string
	Description   string
	RootQueries   []string
	Types         []string
	MaxDepth      int
	MaxFields     int
	MaxRootFields int
}

type GraphQLSourceConfig struct {
	Name             string
	Description      string
	Endpoint         string
	APIKey           string
	SchemaPath       string
	TimeoutMS        int
	MaxResponseBytes int
	Headers          map[string]string
	MaxDepth         int
	MaxFields        int
	MaxRootFields    int
	MaxFragments     int
	Domains          []GraphQLDomainConfig
}

type GraphQLMutationPolicyConfig struct {
	Name                    string
	Description             string
	Source                  string
	Domain                  string
	RootMutation            string
	IdempotencyMode         string
	IdempotencyHeader       string
	IdempotencyVariablePath string
	MaxDepth                int
	MaxFields               int
	MaxRootFields           int
	MaxFragments            int
}

type GraphQLRegistryConfig struct {
	DefaultSource    string
	Sources          []GraphQLSourceConfig
	MutationPolicies []GraphQLMutationPolicyConfig
}

type GraphQLSourceRegistry struct {
	inner *graphqlschema.Registry
}

func NewGraphQLSourceRegistry(cfg GraphQLRegistryConfig) (*GraphQLSourceRegistry, error) {
	sources := make([]graphqlschema.SourceConfig, 0, len(cfg.Sources))
	for _, source := range cfg.Sources {
		sources = append(sources, graphqlschema.SourceConfig{
			Name:             strings.TrimSpace(source.Name),
			Description:      strings.TrimSpace(source.Description),
			Endpoint:         strings.TrimSpace(source.Endpoint),
			APIKey:           strings.TrimSpace(source.APIKey),
			SchemaPath:       strings.TrimSpace(source.SchemaPath),
			TimeoutMS:        source.TimeoutMS,
			MaxResponseBytes: source.MaxResponseBytes,
			Headers:          cloneGraphQLHeaders(source.Headers),
			MaxDepth:         source.MaxDepth,
			MaxFields:        source.MaxFields,
			MaxRootFields:    source.MaxRootFields,
			MaxFragments:     source.MaxFragments,
			Domains:          graphQLDomainConfigs(source.Domains),
		})
	}
	registry, err := graphqlschema.NewRegistry(graphqlschema.RegistryConfig{
		DefaultSource: strings.TrimSpace(cfg.DefaultSource),
		Sources:       sources,
		MutationPolicies: graphQLMutationPolicyConfigs(
			cfg.MutationPolicies,
		),
	})
	if err != nil {
		return nil, err
	}
	return &GraphQLSourceRegistry{inner: registry}, nil
}

func (r *GraphQLSourceRegistry) resolveSource(name string) (*graphqlschema.Source, error) {
	if r == nil || r.inner == nil {
		return nil, fmt.Errorf("graphql source registry is not configured")
	}
	return r.inner.ResolveSource(strings.TrimSpace(name))
}

func (r *GraphQLSourceRegistry) listSources() []graphqlschema.Source {
	if r == nil || r.inner == nil {
		return nil
	}
	return r.inner.ListSources()
}

func graphQLDomainConfigs(raw []GraphQLDomainConfig) []graphqlschema.DomainConfig {
	if len(raw) == 0 {
		return nil
	}

	out := make([]graphqlschema.DomainConfig, 0, len(raw))
	for _, domain := range raw {
		out = append(out, graphqlschema.DomainConfig{
			Name:          strings.TrimSpace(domain.Name),
			Description:   strings.TrimSpace(domain.Description),
			RootQueries:   append([]string(nil), domain.RootQueries...),
			Types:         append([]string(nil), domain.Types...),
			MaxDepth:      domain.MaxDepth,
			MaxFields:     domain.MaxFields,
			MaxRootFields: domain.MaxRootFields,
		})
	}
	return out
}

func graphQLMutationPolicyConfigs(raw []GraphQLMutationPolicyConfig) []graphqlschema.MutationPolicyConfig {
	if len(raw) == 0 {
		return nil
	}

	out := make([]graphqlschema.MutationPolicyConfig, 0, len(raw))
	for _, policy := range raw {
		out = append(out, graphqlschema.MutationPolicyConfig{
			Name:                    strings.TrimSpace(policy.Name),
			Description:             strings.TrimSpace(policy.Description),
			Source:                  strings.TrimSpace(policy.Source),
			Domain:                  strings.TrimSpace(policy.Domain),
			RootMutation:            strings.TrimSpace(policy.RootMutation),
			IdempotencyMode:         strings.TrimSpace(policy.IdempotencyMode),
			IdempotencyHeader:       strings.TrimSpace(policy.IdempotencyHeader),
			IdempotencyVariablePath: strings.TrimSpace(policy.IdempotencyVariablePath),
			MaxDepth:                policy.MaxDepth,
			MaxFields:               policy.MaxFields,
			MaxRootFields:           policy.MaxRootFields,
			MaxFragments:            policy.MaxFragments,
		})
	}
	return out
}
