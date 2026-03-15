package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"

	"ghost-os/bridge/tools/internal/graphqlschema"
)

const graphQLQuerySourceName = "graphql_query.graphql"

type graphQLResponseEnvelope struct {
	Errors json.RawMessage `json:"errors,omitempty"`
}

type graphQLQuerySummary struct {
	OperationName  string
	Depth          int
	FieldCount     int
	RootFieldCount int
	FragmentCount  int
	RootFields     []string
}

type graphQLQueryWalker struct {
	document     *ast.QueryDocument
	summary      graphQLQuerySummary
	fragmentSeen map[string]bool
	fragmentPath map[string]bool
}

func validateGraphQLQueryDocument(
	queryText string,
	operationName string,
	source *graphqlschema.Source,
	domain string,
) (graphQLQuerySummary, error) {
	document, err := parser.ParseQuery(&ast.Source{
		Name:  graphQLQuerySourceName,
		Input: queryText,
	})
	if err != nil {
		return graphQLQuerySummary{}, fmt.Errorf("parse graphql query: %w", err)
	}
	operation, err := selectGraphQLOperation(document.Operations, operationName)
	if err != nil {
		return graphQLQuerySummary{}, err
	}
	summary, err := summarizeGraphQLQuery(document, operation)
	if err != nil {
		return summary, err
	}
	if err := validateGraphQLQueryBudget(summary, source, strings.TrimSpace(domain)); err != nil {
		return summary, err
	}
	return summary, nil
}

func selectGraphQLOperation(
	operations ast.OperationList,
	operationName string,
) (*ast.OperationDefinition, error) {
	if len(operations) == 0 {
		return nil, fmt.Errorf("graphql query must include at least one query operation")
	}
	for _, operation := range operations {
		if operation == nil {
			return nil, fmt.Errorf("graphql query contains an empty operation definition")
		}
		if operation.Operation != ast.Query {
			return nil, fmt.Errorf("graphql_query only supports query operations, found %q", operation.Operation)
		}
	}
	trimmed := strings.TrimSpace(operationName)
	if len(operations) > 1 && trimmed == "" {
		return nil, fmt.Errorf("operation_name is required when multiple query operations are present")
	}
	if trimmed == "" {
		return operations[0], nil
	}
	operation := operations.ForName(trimmed)
	if operation != nil {
		return operation, nil
	}
	return nil, fmt.Errorf("graphql query does not define operation %q", trimmed)
}

func summarizeGraphQLQuery(
	document *ast.QueryDocument,
	operation *ast.OperationDefinition,
) (graphQLQuerySummary, error) {
	walker := graphQLQueryWalker{
		document:     document,
		fragmentSeen: make(map[string]bool),
		fragmentPath: make(map[string]bool),
	}
	if operation != nil {
		walker.summary.OperationName = strings.TrimSpace(operation.Name)
	}
	if err := walker.walkSelectionSet(operation.SelectionSet, 1, true); err != nil {
		return walker.summary, err
	}
	walker.summary.FragmentCount = len(walker.fragmentSeen)
	return walker.summary, nil
}

func (w *graphQLQueryWalker) walkSelectionSet(
	set ast.SelectionSet,
	depth int,
	root bool,
) error {
	for _, selection := range set {
		if err := w.walkSelection(selection, depth, root); err != nil {
			return err
		}
	}
	return nil
}

func (w *graphQLQueryWalker) walkSelection(
	selection ast.Selection,
	depth int,
	root bool,
) error {
	switch item := selection.(type) {
	case *ast.Field:
		return w.walkField(item, depth, root)
	case *ast.FragmentSpread:
		return w.walkFragmentSpread(item, depth, root)
	case *ast.InlineFragment:
		return w.walkSelectionSet(item.SelectionSet, depth, root)
	default:
		return fmt.Errorf("graphql query contains unsupported selection %T", selection)
	}
}

func (w *graphQLQueryWalker) walkField(field *ast.Field, depth int, root bool) error {
	if field == nil {
		return fmt.Errorf("graphql query contains an empty field selection")
	}
	if field.Name == "__schema" || field.Name == "__type" {
		return fmt.Errorf("graphql introspection fields are not allowed")
	}
	w.summary.FieldCount++
	if depth > w.summary.Depth {
		w.summary.Depth = depth
	}
	if root {
		w.summary.RootFieldCount++
		w.summary.RootFields = append(w.summary.RootFields, field.Name)
	}
	if len(field.SelectionSet) == 0 {
		return nil
	}
	return w.walkSelectionSet(field.SelectionSet, depth+1, false)
}

func (w *graphQLQueryWalker) walkFragmentSpread(
	fragment *ast.FragmentSpread,
	depth int,
	root bool,
) error {
	if fragment == nil {
		return fmt.Errorf("graphql query contains an empty fragment spread")
	}
	name := strings.TrimSpace(fragment.Name)
	if name == "" {
		return fmt.Errorf("graphql query contains a fragment spread without a name")
	}
	definition := w.document.Fragments.ForName(name)
	if definition == nil {
		return fmt.Errorf("graphql query references undefined fragment %q", name)
	}
	if w.fragmentPath[name] {
		return fmt.Errorf("graphql query contains cyclic fragment %q", name)
	}
	w.fragmentSeen[name] = true
	w.fragmentPath[name] = true
	defer delete(w.fragmentPath, name)
	return w.walkSelectionSet(definition.SelectionSet, depth, root)
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
	if summary.Depth > budget.MaxDepth {
		return fmt.Errorf("graphql query depth %d exceeds max_depth=%d", summary.Depth, budget.MaxDepth)
	}
	if summary.FieldCount > budget.MaxFields {
		return fmt.Errorf("graphql query field_count %d exceeds max_fields=%d", summary.FieldCount, budget.MaxFields)
	}
	if summary.RootFieldCount > budget.MaxRootFields {
		return fmt.Errorf("graphql query root_field_count %d exceeds max_root_fields=%d", summary.RootFieldCount, budget.MaxRootFields)
	}
	if summary.FragmentCount > budget.MaxFragments {
		return fmt.Errorf("graphql query fragment_count %d exceeds max_fragments=%d", summary.FragmentCount, budget.MaxFragments)
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
