package tools

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"ghost-os/bridge/tools/internal/graphqlschema"
)

var graphQLTextFencePattern = regexp.MustCompile("(?is)```(?:graphql|gql)?\\s*([\\s\\S]*?)```")

func inferQueryDomain(source *graphqlschema.Source, rootFields []string) (string, error) {
	if source == nil {
		return "", fmt.Errorf("graphql source is not configured")
	}
	domains := source.DomainList()
	if len(domains) == 0 {
		return "", nil
	}
	matches := filterQueryDomainsByRootFields(source, domains, rootFields)
	return resolveInferredDomain(matches, "query root fields")
}

func filterQueryDomainsByRootFields(
	source *graphqlschema.Source,
	domains []graphqlschema.Domain,
	rootFields []string,
) []string {
	matches := make([]string, 0, len(domains))
	for _, domain := range domains {
		if domainAllowsAllRootQueries(source, domain.Name, rootFields) {
			matches = append(matches, domain.Name)
		}
	}
	return matches
}

func domainAllowsAllRootQueries(source *graphqlschema.Source, domain string, rootFields []string) bool {
	for _, field := range rootFields {
		allowed, err := source.DomainAllowsRootQuery(domain, field)
		if err != nil || !allowed {
			return false
		}
	}
	return true
}

func inferMutationDomain(
	registry *GraphQLSourceRegistry,
	source *graphqlschema.Source,
	mutationText string,
) (string, error) {
	summary, err := parseAndSummarizeGraphQLMutation(mutationText, "")
	if err != nil {
		return "", err
	}
	rootMutation := summary.RootFields[0]
	matches, err := mutationPolicyDomainMatches(registry, source, rootMutation)
	if err != nil {
		return "", err
	}
	return resolveInferredDomain(matches, "mutation policy")
}

func mutationPolicyDomainMatches(
	registry *GraphQLSourceRegistry,
	source *graphqlschema.Source,
	rootMutation string,
) ([]string, error) {
	if source == nil {
		return nil, fmt.Errorf("graphql source is not configured")
	}
	domains := source.DomainList()
	matches := make([]string, 0, len(domains))
	for _, domain := range domains {
		policies, err := registry.listMutationPolicies(source.Name, domain.Name)
		if err != nil {
			return nil, err
		}
		for _, policy := range policies {
			if policy.RootMutation == rootMutation {
				matches = append(matches, domain.Name)
			}
		}
	}
	sort.Strings(matches)
	return matches, nil
}

func resolveInferredDomain(matches []string, reason string) (string, error) {
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("failed to infer graphql domain from %s", reason)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf(
			"graphql domain inference is ambiguous for %s: %s",
			reason,
			strings.Join(matches, ","),
		)
	}
}

func extractGraphQLFenceContent(text string) string {
	match := graphQLTextFencePattern.FindStringSubmatch(text)
	if len(match) != 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func looksLikeGraphQLDocument(text string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(text))
	return strings.HasPrefix(trimmed, "{") ||
		strings.HasPrefix(trimmed, "query") ||
		strings.HasPrefix(trimmed, "mutation")
}

func sourceName(source *graphqlschema.Source) string {
	if source == nil {
		return ""
	}
	return source.Name
}
