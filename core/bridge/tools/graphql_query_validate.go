package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"

	"ghost-os/bridge/tools/internal/graphqlschema"
)

const graphQLQuerySourceName = "graphql_query.graphql"

type graphQLResponseEnvelope struct {
	Errors json.RawMessage `json:"errors,omitempty"`
}

type graphQLQuerySummary = graphQLOperationSummary

func validateGraphQLQueryDocument(
	queryText string,
	operationName string,
	source *graphqlschema.Source,
	domain string,
) (graphQLQuerySummary, error) {
	document, err := parseGraphQLDocument(graphQLQuerySourceName, queryText)
	if err != nil {
		return graphQLQuerySummary{}, err
	}
	operation, err := selectGraphQLOperationByKind(
		document.Operations,
		operationName,
		ast.Query,
		"graphql_query",
	)
	if err != nil {
		return graphQLQuerySummary{}, err
	}
	summary, err := summarizeGraphQLOperation(document, operation)
	if err != nil {
		return summary, err
	}
	if err := validateGraphQLQueryBudget(summary, source, strings.TrimSpace(domain)); err != nil {
		return summary, err
	}
	return summary, nil
}

func validateGraphQLQueryBudget(
	summary graphQLQuerySummary,
	source *graphqlschema.Source,
	domain string,
) error {
	if source == nil {
		return fmt.Errorf("graphql source is not configured")
	}
	budget, err := source.DomainBudget(domain)
	if err != nil {
		return err
	}
	if err := validateGraphQLOperationBudget(summary, budget, "graphql query"); err != nil {
		return err
	}
	return validateGraphQLDomainRootFields(source, domain, summary.RootFields)
}

func validateGraphQLDomainRootFields(
	source *graphqlschema.Source,
	domain string,
	rootFields []string,
) error {
	if source == nil || domain == "" {
		return nil
	}
	for _, rootField := range rootFields {
		allowed, err := source.DomainAllowsRootQuery(domain, rootField)
		if err != nil {
			return err
		}
		if allowed {
			continue
		}
		return fmt.Errorf("graphql root field %q is outside domain %q", rootField, domain)
	}
	return nil
}

func validateGraphQLResponseBody(body []byte) error {
	var envelope graphQLResponseEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("decode graphql response: %w", err)
	}
	if graphQLErrorsPresent(envelope.Errors) {
		return fmt.Errorf("graphql errors: %s", strings.TrimSpace(string(envelope.Errors)))
	}
	return nil
}

func graphQLErrorsPresent(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return false
	}
	return !bytes.Equal(trimmed, []byte("null")) && !bytes.Equal(trimmed, []byte("[]"))
}
