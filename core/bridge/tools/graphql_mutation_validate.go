package tools

import (
	"fmt"

	"github.com/vektah/gqlparser/v2/ast"

	"ghost-os/bridge/tools/internal/graphqlschema"
)

const graphQLMutationSourceName = "graphql_mutation.graphql"

type graphQLPreparedMutation struct {
	Policy      graphqlschema.MutationPolicy
	RootField   graphqlschema.Field
	Summary     graphQLOperationSummary
	Source      *graphqlschema.Source
	Domain      string
	MutationDoc string
}

func prepareGraphQLMutationDocument(
	mutationText string,
	operationName string,
	source *graphqlschema.Source,
	domain string,
	registry *GraphQLSourceRegistry,
) (graphQLPreparedMutation, error) {
	summary, err := parseAndSummarizeGraphQLMutation(mutationText, operationName)
	if err != nil {
		return graphQLPreparedMutation{}, err
	}
	if summary.RootFieldCount != 1 {
		return graphQLPreparedMutation{}, fmt.Errorf("graphql mutation must select exactly one root mutation field")
	}
	rootMutation := summary.RootFields[0]
	rootField, ok := source.Schema.RootMutationByName(rootMutation)
	if !ok {
		return graphQLPreparedMutation{}, fmt.Errorf(
			"graphql mutation root field %q was not found in source %q schema snapshot",
			rootMutation,
			source.Name,
		)
	}
	policy, err := registry.resolveMutationPolicy(source.Name, domain, rootMutation)
	if err != nil {
		return graphQLPreparedMutation{}, err
	}
	if err := validateGraphQLOperationBudget(summary, policy.Budget, "graphql mutation"); err != nil {
		return graphQLPreparedMutation{}, err
	}
	return graphQLPreparedMutation{
		Policy:      policy,
		RootField:   rootField,
		Summary:     summary,
		Source:      source,
		Domain:      domain,
		MutationDoc: mutationText,
	}, nil
}

func parseAndSummarizeGraphQLMutation(
	mutationText string,
	operationName string,
) (graphQLOperationSummary, error) {
	document, err := parseGraphQLDocument(graphQLMutationSourceName, mutationText)
	if err != nil {
		return graphQLOperationSummary{}, err
	}
	operation, err := selectGraphQLOperationByKind(
		document.Operations,
		operationName,
		ast.Mutation,
		"graphql_mutation",
	)
	if err != nil {
		return graphQLOperationSummary{}, err
	}
	summary, err := summarizeGraphQLOperation(document, operation)
	if err != nil {
		return graphQLOperationSummary{}, err
	}
	return summary, nil
}
