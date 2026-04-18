package tools

import toolgraphql "ghost-os/bridge/tools/graphql"

type GraphQLTextExecutionResult = toolgraphql.GraphQLTextExecutionResult

type GraphQLTextToolCall = toolgraphql.GraphQLTextToolCall

type GraphQLTextExecutor = toolgraphql.GraphQLTextExecutor

type GraphQLTextExecutorOptions = toolgraphql.GraphQLTextExecutorOptions

type GraphQLToolRuntimeSchema = toolgraphql.GraphQLToolRuntimeSchema

type GraphQLToolRuntimeField = toolgraphql.GraphQLToolRuntimeField

type GraphQLToolRuntimeArgument = toolgraphql.GraphQLToolRuntimeArgument

type GraphQLToolRuntimeInputType = toolgraphql.GraphQLToolRuntimeInputType

type GraphQLToolRuntimeEnumType = toolgraphql.GraphQLToolRuntimeEnumType

type GraphQLToolRuntimeEnumValue = toolgraphql.GraphQLToolRuntimeEnumValue

type GraphQLToolIDEntry = toolgraphql.GraphQLToolIDEntry

type GraphQLTextProtocolFeedback = toolgraphql.GraphQLTextProtocolFeedback

type GraphQLTextProtocolError = toolgraphql.GraphQLTextProtocolError

func NewGraphQLTextExecutor(catalog ToolCatalog) GraphQLTextExecutor {
	return toolgraphql.NewGraphQLTextExecutor(catalog)
}

func NewGraphQLTextExecutorWithOptions(
	catalog ToolCatalog,
	opts GraphQLTextExecutorOptions,
) GraphQLTextExecutor {
	return toolgraphql.NewGraphQLTextExecutorWithOptions(catalog, opts)
}

func BuildGraphQLToolRuntimeSchema(catalog ToolCatalog) GraphQLToolRuntimeSchema {
	return toolgraphql.BuildGraphQLToolRuntimeSchema(catalog)
}

func FormatGraphQLToolRuntimePrompt(catalog ToolCatalog) string {
	return toolgraphql.FormatGraphQLToolRuntimePrompt(catalog)
}

func NewGraphQLTextToolCallID() string {
	return toolgraphql.NewGraphQLTextToolCallID()
}

func AsGraphQLTextProtocolError(err error) (*GraphQLTextProtocolError, bool) {
	return toolgraphql.AsGraphQLTextProtocolError(err)
}
