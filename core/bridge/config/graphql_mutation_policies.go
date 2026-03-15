package config

import "sort"

func normalizeGraphQLMutationPolicies(raw []GraphQLMutationPolicyConfig) []GraphQLMutationPolicyConfig {
	if len(raw) == 0 {
		return nil
	}

	out := make([]GraphQLMutationPolicyConfig, 0, len(raw))
	for _, policy := range raw {
		out = append(out, GraphQLMutationPolicyConfig{
			Name:          normalizeOptionalString(policy.Name),
			Description:   normalizeOptionalString(policy.Description),
			Source:        normalizeOptionalString(policy.Source),
			Domain:        normalizeOptionalString(policy.Domain),
			RootMutation:  normalizeOptionalString(policy.RootMutation),
			MaxDepth:      policy.MaxDepth,
			MaxFields:     policy.MaxFields,
			MaxRootFields: policy.MaxRootFields,
			MaxFragments:  policy.MaxFragments,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Source != out[j].Source {
			return out[i].Source < out[j].Source
		}
		if out[i].Domain != out[j].Domain {
			return out[i].Domain < out[j].Domain
		}
		return out[i].RootMutation < out[j].RootMutation
	})
	return out
}

func graphQLMutationPoliciesFromFile(raw []graphQLMutationPolicyFileConfig) []GraphQLMutationPolicyConfig {
	if len(raw) == 0 {
		return nil
	}

	out := make([]GraphQLMutationPolicyConfig, 0, len(raw))
	for _, policy := range raw {
		out = append(out, GraphQLMutationPolicyConfig{
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

func graphQLMutationPoliciesToFileConfigs(raw []GraphQLMutationPolicyConfig) []graphQLMutationPolicyFileConfig {
	if len(raw) == 0 {
		return nil
	}

	out := make([]graphQLMutationPolicyFileConfig, 0, len(raw))
	for _, policy := range raw {
		out = append(out, graphQLMutationPolicyFileConfig{
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

func graphQLMutationPolicyInputsToFileConfigs(raw []GraphQLMutationPolicyInput) []graphQLMutationPolicyFileConfig {
	if len(raw) == 0 {
		return nil
	}

	out := make([]graphQLMutationPolicyFileConfig, 0, len(raw))
	for _, policy := range raw {
		out = append(out, graphQLMutationPolicyFileConfig{
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

func graphQLMutationPolicySnapshots(raw []GraphQLMutationPolicyConfig) []GraphQLMutationPolicySnapshot {
	if len(raw) == 0 {
		return nil
	}

	out := make([]GraphQLMutationPolicySnapshot, 0, len(raw))
	for _, policy := range raw {
		out = append(out, GraphQLMutationPolicySnapshot{
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

func normalizeGraphQLMutationPolicyFileConfigs(raw []graphQLMutationPolicyFileConfig) []graphQLMutationPolicyFileConfig {
	if len(raw) == 0 {
		return nil
	}

	out := make([]graphQLMutationPolicyFileConfig, 0, len(raw))
	for _, policy := range raw {
		out = append(out, graphQLMutationPolicyFileConfig{
			Name:          normalizeOptionalString(policy.Name),
			Description:   normalizeOptionalString(policy.Description),
			Source:        normalizeOptionalString(policy.Source),
			Domain:        normalizeOptionalString(policy.Domain),
			RootMutation:  normalizeOptionalString(policy.RootMutation),
			MaxDepth:      policy.MaxDepth,
			MaxFields:     policy.MaxFields,
			MaxRootFields: policy.MaxRootFields,
			MaxFragments:  policy.MaxFragments,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Source != out[j].Source {
			return out[i].Source < out[j].Source
		}
		if out[i].Domain != out[j].Domain {
			return out[i].Domain < out[j].Domain
		}
		return out[i].RootMutation < out[j].RootMutation
	})
	return out
}
