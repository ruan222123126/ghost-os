package config

import (
	"fmt"
	"strings"
)

func validateGraphQLMutationPolicies(cfg GraphQLConfig) error {
	if len(cfg.MutationPolicies) == 0 {
		return nil
	}

	seenNames := make(map[string]bool, len(cfg.MutationPolicies))
	seenTargets := make(map[string]bool, len(cfg.MutationPolicies))
	for _, policy := range cfg.MutationPolicies {
		if err := validateGraphQLMutationPolicy(cfg.Sources, policy, seenNames, seenTargets); err != nil {
			return err
		}
	}
	return nil
}

func validateGraphQLMutationPolicy(
	sources []GraphQLSourceConfig,
	policy GraphQLMutationPolicyConfig,
	seenNames map[string]bool,
	seenTargets map[string]bool,
) error {
	if policy.Name == "" {
		return fmt.Errorf("graphql mutation policy name is required")
	}
	if policy.Source == "" {
		return fmt.Errorf("graphql mutation policy %q source is required", policy.Name)
	}
	if policy.Domain == "" {
		return fmt.Errorf("graphql mutation policy %q domain is required", policy.Name)
	}
	if policy.RootMutation == "" {
		return fmt.Errorf("graphql mutation policy %q root_mutation is required", policy.Name)
	}
	if seenNames[policy.Name] {
		return fmt.Errorf("graphql mutation policy %q is duplicated", policy.Name)
	}
	seenNames[policy.Name] = true

	targetKey := graphQLMutationPolicyTargetKey(
		policy.Source,
		policy.Domain,
		policy.RootMutation,
	)
	if seenTargets[targetKey] {
		return fmt.Errorf(
			"graphql mutation policy for source %q domain %q root mutation %q is duplicated",
			policy.Source,
			policy.Domain,
			policy.RootMutation,
		)
	}
	seenTargets[targetKey] = true

	source, ok := findGraphQLSourceConfigByName(sources, policy.Source)
	if !ok {
		return fmt.Errorf(
			"graphql mutation policy %q source %q was not found in graphql_sources",
			policy.Name,
			policy.Source,
		)
	}
	if !graphQLSourceHasDomain(source, policy.Domain) {
		return fmt.Errorf(
			"graphql mutation policy %q domain %q was not found in source %q",
			policy.Name,
			policy.Domain,
			policy.Source,
		)
	}
	if err := validateGraphQLOverrideValue(
		policy.MaxDepth,
		policy.Source,
		policy.Domain,
		"max_depth",
	); err != nil {
		return err
	}
	if err := validateGraphQLOverrideValue(
		policy.MaxFields,
		policy.Source,
		policy.Domain,
		"max_fields",
	); err != nil {
		return err
	}
	if err := validateGraphQLOverrideValue(
		policy.MaxRootFields,
		policy.Source,
		policy.Domain,
		"max_root_fields",
	); err != nil {
		return err
	}
	return validateGraphQLOverrideValue(
		policy.MaxFragments,
		policy.Source,
		policy.Domain,
		"max_fragments",
	)
}

func findGraphQLSourceConfigByName(sources []GraphQLSourceConfig, name string) (GraphQLSourceConfig, bool) {
	sourceName := strings.TrimSpace(name)
	for _, source := range sources {
		if source.Name == sourceName {
			return source, true
		}
	}
	return GraphQLSourceConfig{}, false
}

func graphQLSourceHasDomain(source GraphQLSourceConfig, domainName string) bool {
	domain := strings.TrimSpace(domainName)
	for _, item := range source.Domains {
		if item.Name == domain {
			return true
		}
	}
	return false
}

func graphQLMutationPolicyTargetKey(source string, domain string, rootMutation string) string {
	return strings.Join([]string{
		strings.TrimSpace(source),
		strings.TrimSpace(domain),
		strings.TrimSpace(rootMutation),
	}, "\x00")
}
