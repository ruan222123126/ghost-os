package tools

import (
	"fmt"
	"strings"

	"ghost-os/bridge/tools/internal/graphqlschema"
)

func (r *GraphQLSourceRegistry) HasMutationPolicies() bool {
	return r != nil && r.inner != nil && r.inner.HasMutationPolicies()
}

func (r *GraphQLSourceRegistry) sourceHasMutationPolicies(sourceName string) bool {
	if r == nil || r.inner == nil {
		return false
	}
	return r.inner.SourceHasMutationPolicies(strings.TrimSpace(sourceName))
}

func (r *GraphQLSourceRegistry) resolveMutationPolicy(
	sourceName string,
	domainName string,
	rootMutation string,
) (graphqlschema.MutationPolicy, error) {
	if r == nil || r.inner == nil {
		return graphqlschema.MutationPolicy{}, fmt.Errorf("graphql source registry is not configured")
	}
	return r.inner.ResolveMutationPolicy(
		strings.TrimSpace(sourceName),
		strings.TrimSpace(domainName),
		strings.TrimSpace(rootMutation),
	)
}

func (r *GraphQLSourceRegistry) listMutationPolicies(
	sourceName string,
	domainName string,
) ([]graphqlschema.MutationPolicy, error) {
	if r == nil || r.inner == nil {
		return nil, fmt.Errorf("graphql source registry is not configured")
	}
	return r.inner.ListMutationPolicies(
		strings.TrimSpace(sourceName),
		strings.TrimSpace(domainName),
	)
}
