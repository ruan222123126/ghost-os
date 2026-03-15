package tools

import (
	"fmt"
	"strings"

	"ghost-os/bridge/tools/internal/graphqlschema"
	"ghost-os/bridge/tools/internal/tooljson"
)

func listGraphQLRootMutations(
	source *graphqlschema.Source,
	domain string,
	registry *GraphQLSourceRegistry,
) (string, error) {
	if source == nil {
		return "", fmt.Errorf("graphql source is not configured")
	}
	policies, err := registry.listMutationPolicies(source.Name, strings.TrimSpace(domain))
	if err != nil {
		return "", err
	}
	items := make([]graphqlSchemaMutationPolicyPayload, 0, len(policies))
	for _, policy := range policies {
		rootField, ok := source.Schema.RootMutationByName(policy.RootMutation)
		if !ok {
			return "", fmt.Errorf(
				"graphql root mutation %q was not found in source %q schema snapshot",
				policy.RootMutation,
				source.Name,
			)
		}
		items = append(items, newGraphQLSchemaMutationPolicyPayload(policy, rootField))
	}
	return tooljson.Encode(struct {
		Action        string                               `json:"action"`
		Source        string                               `json:"source"`
		Domain        string                               `json:"domain"`
		RootMutations []graphqlSchemaMutationPolicyPayload `json:"root_mutations"`
	}{
		Action:        graphqlSchemaLookupActionListRootMutations,
		Source:        source.Name,
		Domain:        strings.TrimSpace(domain),
		RootMutations: items,
	})
}

func describeGraphQLMutationPolicy(
	source *graphqlschema.Source,
	domain string,
	name string,
	registry *GraphQLSourceRegistry,
) (string, error) {
	if source == nil {
		return "", fmt.Errorf("graphql source is not configured")
	}
	rootMutation := strings.TrimSpace(name)
	if rootMutation == "" {
		return "", fmt.Errorf("name is required for describe_mutation_policy")
	}
	policy, err := registry.resolveMutationPolicy(source.Name, strings.TrimSpace(domain), rootMutation)
	if err != nil {
		return "", err
	}
	rootField, ok := source.Schema.RootMutationByName(policy.RootMutation)
	if !ok {
		return "", fmt.Errorf(
			"graphql root mutation %q was not found in source %q schema snapshot",
			policy.RootMutation,
			source.Name,
		)
	}
	return tooljson.Encode(struct {
		Action string                             `json:"action"`
		Source string                             `json:"source"`
		Domain string                             `json:"domain"`
		Policy graphqlSchemaMutationPolicyPayload `json:"policy"`
	}{
		Action: graphqlSchemaLookupActionDescribeMutationPolicy,
		Source: source.Name,
		Domain: strings.TrimSpace(domain),
		Policy: newGraphQLSchemaMutationPolicyPayload(policy, rootField),
	})
}

func newGraphQLSchemaMutationPolicyPayload(
	policy graphqlschema.MutationPolicy,
	rootField graphqlschema.Field,
) graphqlSchemaMutationPolicyPayload {
	return graphqlSchemaMutationPolicyPayload{
		Name:                    policy.Name,
		Description:             policy.Description,
		Domain:                  policy.Domain,
		RootMutation:            newGraphQLSchemaFieldPayload(rootField),
		IdempotencyMode:         policy.IdempotencyMode,
		IdempotencyHeader:       policy.IdempotencyHeader,
		IdempotencyVariablePath: policy.IdempotencyVariablePath,
		MaxDepth:                policy.Budget.MaxDepth,
		MaxFields:               policy.Budget.MaxFields,
		MaxRootFields:           policy.Budget.MaxRootFields,
		MaxFragments:            policy.Budget.MaxFragments,
		Summary: buildGraphQLMutationSummary(
			policy.Source,
			policy.Domain,
			policy.Name,
			policy.RootMutation,
			nil,
		),
	}
}
