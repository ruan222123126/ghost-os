package tools

import (
	"fmt"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"

	"ghost-os/bridge/tools/internal/graphqlschema"
)

type graphQLOperationSummary struct {
	OperationName  string
	Depth          int
	FieldCount     int
	RootFieldCount int
	FragmentCount  int
	RootFields     []string
}

type graphQLOperationWalker struct {
	document     *ast.QueryDocument
	summary      graphQLOperationSummary
	fragmentSeen map[string]bool
	fragmentPath map[string]bool
}

func parseGraphQLDocument(sourceName string, text string) (*ast.QueryDocument, error) {
	document, err := parser.ParseQuery(&ast.Source{
		Name:  strings.TrimSpace(sourceName),
		Input: text,
	})
	if err != nil {
		return nil, fmt.Errorf("parse graphql document: %w", err)
	}
	return document, nil
}

func selectGraphQLOperationByKind(
	operations ast.OperationList,
	operationName string,
	expected ast.Operation,
	toolName string,
) (*ast.OperationDefinition, error) {
	if len(operations) == 0 {
		return nil, fmt.Errorf("%s must include at least one %s operation", toolName, expected)
	}
	for _, operation := range operations {
		if operation == nil {
			return nil, fmt.Errorf("%s contains an empty operation definition", toolName)
		}
		if operation.Operation != expected {
			return nil, fmt.Errorf("%s only supports %s operations, found %q", toolName, expected, operation.Operation)
		}
	}
	trimmed := strings.TrimSpace(operationName)
	if len(operations) > 1 && trimmed == "" {
		return nil, fmt.Errorf("operation_name is required when multiple %s operations are present", expected)
	}
	if trimmed == "" {
		return operations[0], nil
	}
	operation := operations.ForName(trimmed)
	if operation != nil {
		return operation, nil
	}
	return nil, fmt.Errorf("%s does not define operation %q", toolName, trimmed)
}

func summarizeGraphQLOperation(
	document *ast.QueryDocument,
	operation *ast.OperationDefinition,
) (graphQLOperationSummary, error) {
	walker := graphQLOperationWalker{
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

func (w *graphQLOperationWalker) walkSelectionSet(
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

func (w *graphQLOperationWalker) walkSelection(
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
		return fmt.Errorf("graphql document contains unsupported selection %T", selection)
	}
}

func (w *graphQLOperationWalker) walkField(field *ast.Field, depth int, root bool) error {
	if field == nil {
		return fmt.Errorf("graphql document contains an empty field selection")
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

func (w *graphQLOperationWalker) walkFragmentSpread(
	fragment *ast.FragmentSpread,
	depth int,
	root bool,
) error {
	if fragment == nil {
		return fmt.Errorf("graphql document contains an empty fragment spread")
	}
	name := strings.TrimSpace(fragment.Name)
	if name == "" {
		return fmt.Errorf("graphql document contains a fragment spread without a name")
	}
	definition := w.document.Fragments.ForName(name)
	if definition == nil {
		return fmt.Errorf("graphql document references undefined fragment %q", name)
	}
	if w.fragmentPath[name] {
		return fmt.Errorf("graphql document contains cyclic fragment %q", name)
	}
	w.fragmentSeen[name] = true
	w.fragmentPath[name] = true
	defer delete(w.fragmentPath, name)
	return w.walkSelectionSet(definition.SelectionSet, depth, root)
}

func validateGraphQLOperationBudget(
	summary graphQLOperationSummary,
	budget graphqlschema.Budget,
	label string,
) error {
	if summary.Depth > budget.MaxDepth {
		return fmt.Errorf("%s depth %d exceeds max_depth=%d", label, summary.Depth, budget.MaxDepth)
	}
	if summary.FieldCount > budget.MaxFields {
		return fmt.Errorf("%s field_count %d exceeds max_fields=%d", label, summary.FieldCount, budget.MaxFields)
	}
	if summary.RootFieldCount > budget.MaxRootFields {
		return fmt.Errorf("%s root_field_count %d exceeds max_root_fields=%d", label, summary.RootFieldCount, budget.MaxRootFields)
	}
	if summary.FragmentCount > budget.MaxFragments {
		return fmt.Errorf("%s fragment_count %d exceeds max_fragments=%d", label, summary.FragmentCount, budget.MaxFragments)
	}
	return nil
}
