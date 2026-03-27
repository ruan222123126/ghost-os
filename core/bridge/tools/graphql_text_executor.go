package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"

	"ghost-os/bridge/llm"
)

const graphQLTextSourceName = "graphql_tool_runtime.graphql"

type GraphQLTextExecutionResult struct {
	Recognized bool
	Operation  ast.Operation
	ToolName   string
	Arguments  json.RawMessage
}

type GraphQLTextExecutor interface {
	Execute(ctx context.Context, text string, traceID string) (GraphQLTextExecutionResult, error)
}

type GraphQLTextExecutorOptions struct {
	SanitizeKnownArtifacts bool
}

type graphQLTextExecutor struct {
	catalog                ToolCatalog
	sanitizeKnownArtifacts bool
}

func NewGraphQLTextExecutor(catalog ToolCatalog) GraphQLTextExecutor {
	return NewGraphQLTextExecutorWithOptions(catalog, GraphQLTextExecutorOptions{
		SanitizeKnownArtifacts: true,
	})
}

func NewGraphQLTextExecutorWithOptions(
	catalog ToolCatalog,
	opts GraphQLTextExecutorOptions,
) GraphQLTextExecutor {
	return &graphQLTextExecutor{
		catalog:                catalog,
		sanitizeKnownArtifacts: opts.SanitizeKnownArtifacts,
	}
}

func (e *graphQLTextExecutor) Execute(
	_ context.Context,
	text string,
	traceID string,
) (GraphQLTextExecutionResult, error) {
	normalized := normalizeGraphQLTextDocument(text, e.sanitizeKnownArtifacts)
	if !normalized.Recognized {
		return GraphQLTextExecutionResult{}, nil
	}
	logGraphQLTextSanitization(traceID, normalized.SanitizeKinds, e.sanitizeKnownArtifacts)
	call, err := parseGraphQLToolCallDocument(normalized.Document)
	if err != nil {
		call.Recognized = true
		return call, err
	}
	if e.catalog == nil {
		call.Recognized = true
		return call, newGraphQLTextCatalogUnavailableError()
	}
	def, ok := graphQLVisibleToolDef(e.catalog, call.ToolName)
	if !ok || e.catalog.Get(call.ToolName) == nil {
		call.Recognized = true
		return call, newGraphQLTextUnknownToolError(call.ToolName, visibleGraphQLToolDefNames(e.catalog))
	}
	if err := validateGraphQLToolOperation(call.Operation, def); err != nil {
		call.Recognized = true
		return call, err
	}
	call.Recognized = true
	return call, nil
}

func parseGraphQLToolCallDocument(text string) (GraphQLTextExecutionResult, error) {
	document, err := parseGraphQLDocument(graphQLTextSourceName, text)
	if err != nil {
		return GraphQLTextExecutionResult{}, newGraphQLTextParseError(err)
	}
	if len(document.Fragments) != 0 {
		return GraphQLTextExecutionResult{}, newGraphQLTextFeatureError(
			"unsupported_fragments",
			"graphql tool runtime does not support fragments",
		)
	}
	operation, err := singleGraphQLToolOperation(document.Operations)
	if err != nil {
		return GraphQLTextExecutionResult{}, err
	}
	field, err := singleGraphQLToolField(operation)
	if err != nil {
		return GraphQLTextExecutionResult{Operation: operation.Operation}, err
	}
	argsJSON, err := graphQLFieldArgumentsJSON(field.Arguments)
	if err != nil {
		return GraphQLTextExecutionResult{
			Operation: operation.Operation,
			ToolName:  strings.TrimSpace(field.Name),
		}, err
	}
	return GraphQLTextExecutionResult{
		Operation: operation.Operation,
		ToolName:  strings.TrimSpace(field.Name),
		Arguments: argsJSON,
	}, nil
}

func singleGraphQLToolOperation(
	operations ast.OperationList,
) (*ast.OperationDefinition, error) {
	if len(operations) != 1 {
		return nil, newGraphQLTextOperationCountError(len(operations))
	}
	operation := operations[0]
	if operation == nil {
		return nil, newGraphQLTextFeatureError(
			"empty_operation_definition",
			"graphql tool runtime found an empty operation definition",
		)
	}
	if operation.Operation != ast.Query && operation.Operation != ast.Mutation {
		return nil, newGraphQLTextOperationError(
			"unsupported_operation_type",
			"graphql tool runtime only supports mutation",
			"mutation",
			string(operation.Operation),
		)
	}
	if len(operation.VariableDefinitions) != 0 {
		return nil, newGraphQLTextFeatureError(
			"unsupported_variables",
			"graphql tool runtime does not support variables",
		)
	}
	if len(operation.Directives) != 0 {
		return nil, newGraphQLTextFeatureError(
			"unsupported_directives",
			"graphql tool runtime does not support directives",
		)
	}
	return operation, nil
}

func singleGraphQLToolField(
	operation *ast.OperationDefinition,
) (*ast.Field, error) {
	if operation == nil {
		return nil, newGraphQLTextFeatureError(
			"empty_operation_definition",
			"graphql tool runtime operation is empty",
		)
	}
	if len(operation.SelectionSet) != 1 {
		return nil, newGraphQLTextTopLevelFieldCountError(len(operation.SelectionSet))
	}
	field, ok := operation.SelectionSet[0].(*ast.Field)
	if !ok || field == nil {
		return nil, newGraphQLTextFeatureError(
			"invalid_top_level_selection",
			"graphql tool runtime only supports a top-level field selection",
		)
	}
	if alias := strings.TrimSpace(field.Alias); alias != "" && alias != strings.TrimSpace(field.Name) {
		return nil, newGraphQLTextAliasError(field.Name)
	}
	if len(field.Directives) != 0 {
		return nil, newGraphQLTextFeatureError(
			"unsupported_directives",
			"graphql tool runtime does not support directives",
		)
	}
	if len(field.SelectionSet) != 0 {
		return nil, newGraphQLTextNestedSelectionError(field.Name)
	}
	return field, nil
}

func graphQLFieldArgumentsJSON(arguments ast.ArgumentList) (json.RawMessage, error) {
	if len(arguments) == 0 {
		return json.RawMessage(`{}`), nil
	}
	payload := make(map[string]any, len(arguments))
	for _, argument := range arguments {
		if argument == nil || strings.TrimSpace(argument.Name) == "" {
			err := fmt.Errorf("graphql tool runtime contains an invalid argument")
			return nil, newGraphQLTextArgumentError("", err)
		}
		value, err := graphQLLiteralValue(argument.Value)
		if err != nil {
			return nil, newGraphQLTextArgumentError(
				argument.Name,
				fmt.Errorf("argument %q: %w", argument.Name, err),
			)
		}
		payload[strings.TrimSpace(argument.Name)] = value
	}
	return json.Marshal(payload)
}

func graphQLLiteralValue(value *ast.Value) (any, error) {
	if containsGraphQLVariable(value) {
		return nil, fmt.Errorf("graphql tool runtime does not support variables")
	}
	if value == nil {
		return nil, nil
	}
	decoded, err := value.Value(nil)
	if err != nil {
		return nil, fmt.Errorf("decode graphql literal: %w", err)
	}
	return decoded, nil
}

func containsGraphQLVariable(value *ast.Value) bool {
	if value == nil {
		return false
	}
	if value.Kind == ast.Variable {
		return true
	}
	for _, child := range value.Children {
		if child != nil && containsGraphQLVariable(child.Value) {
			return true
		}
	}
	return false
}

func validateGraphQLToolOperation(
	operation ast.Operation,
	def llm.ToolDef,
) error {
	expected := graphQLToolOperation(def)
	if operation == expected {
		return nil
	}
	return newGraphQLTextWrongOperationError(def.Name, expected, operation)
}

func logGraphQLTextSanitization(traceID string, kinds []string, enabled bool) {
	if !enabled {
		return
	}
	trimmedTraceID := strings.TrimSpace(traceID)
	for _, kind := range kinds {
		if strings.TrimSpace(kind) == "" {
			continue
		}
		log.Printf(
			"trace_id=%s action=GRAPHQL_TEXT_SANITIZE kind=%s",
			trimmedTraceID,
			strings.TrimSpace(kind),
		)
	}
}
