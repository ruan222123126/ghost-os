package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

const graphQLQuerySourceName = "graphql_query.graphql"

type graphQLResponseEnvelope struct {
	Errors json.RawMessage `json:"errors,omitempty"`
}

func validateGraphQLQueryDocument(queryText string, operationName string) error {
	document, err := parser.ParseQuery(&ast.Source{
		Name:  graphQLQuerySourceName,
		Input: queryText,
	})
	if err != nil {
		return fmt.Errorf("parse graphql query: %w", err)
	}
	if len(document.Operations) == 0 {
		return fmt.Errorf("graphql query must include at least one query operation")
	}
	if err := validateGraphQLOperations(document.Operations); err != nil {
		return err
	}
	return validateGraphQLOperationName(document.Operations, operationName)
}

func validateGraphQLOperations(operations ast.OperationList) error {
	for _, operation := range operations {
		if operation == nil {
			return fmt.Errorf("graphql query contains an empty operation definition")
		}
		if operation.Operation != ast.Query {
			return fmt.Errorf("graphql_query only supports query operations, found %q", operation.Operation)
		}
	}
	return nil
}

func validateGraphQLOperationName(operations ast.OperationList, operationName string) error {
	trimmed := strings.TrimSpace(operationName)
	if len(operations) > 1 && trimmed == "" {
		return fmt.Errorf("operation_name is required when multiple query operations are present")
	}
	if trimmed == "" {
		return nil
	}
	for _, operation := range operations {
		if operation != nil && operation.Name == trimmed {
			return nil
		}
	}
	return fmt.Errorf("graphql query does not define operation %q", trimmed)
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
