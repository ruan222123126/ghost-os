package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"strings"
	"testing"
)

func TestGraphQLTextExecutorRejectsQueryToolCallInMutationOnlyMode(t *testing.T) {
	executor := NewGraphQLTextExecutor(graphQLToolRuntimeTestCatalog()).(*graphQLTextExecutor)

	result, err := executor.Execute(
		context.Background(),
		`query { web_search(query: "OpenAI latest news", max_results: 5) }`,
		"trace-graphql-query",
	)
	if err == nil || !strings.Contains(err.Error(), `tool "web_search" must use mutation`) {
		t.Fatalf("expected mutation-only protocol error, got %v", err)
	}
	if !result.Recognized {
		t.Fatalf("expected recognized graphql tool call, got %+v", result)
	}
	if result.ToolName != "web_search" {
		t.Fatalf("unexpected tool name: %+v", result)
	}
	args := decodeGraphQLToolArgs(t, result.Arguments)
	if args["query"] != "OpenAI latest news" {
		t.Fatalf("unexpected args: %+v", args)
	}
	if args["max_results"] != float64(5) {
		t.Fatalf("unexpected args: %+v", args)
	}
	protocolErr, ok := AsGraphQLTextProtocolError(err)
	if !ok {
		t.Fatalf("expected structured protocol error, got %T", err)
	}
	feedback := protocolErr.Feedback()
	if feedback.Expected != "mutation" || feedback.Received != "query" {
		t.Fatalf("unexpected feedback payload: %+v", feedback)
	}
}

func TestGraphQLTextExecutorRecognizesMutationToolCall(t *testing.T) {
	executor := NewGraphQLTextExecutor(graphQLToolRuntimeTestCatalog()).(*graphQLTextExecutor)

	result, err := executor.Execute(
		context.Background(),
		`mutation { ask_human(prompt: "Need approval?", options: [{label: "Approve", allow_custom: false}, {label: "Other", allow_custom: true}]) }`,
		"trace-graphql-mutation",
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.ToolName != "ask_human" {
		t.Fatalf("unexpected tool name: %+v", result)
	}
	args := decodeGraphQLToolArgs(t, result.Arguments)
	if args["prompt"] != "Need approval?" {
		t.Fatalf("unexpected args: %+v", args)
	}
	options, ok := args["options"].([]any)
	if !ok || len(options) != 2 {
		t.Fatalf("unexpected options payload: %+v", args)
	}
}

func TestGraphQLTextExecutorRejectsMultipleTopLevelFields(t *testing.T) {
	executor := NewGraphQLTextExecutor(graphQLToolRuntimeTestCatalog())

	_, err := executor.Execute(
		context.Background(),
		`query { web_search(query: "a") ask_human(prompt: "b") }`,
		"trace-graphql-multi-field",
	)
	if err == nil || !strings.Contains(err.Error(), "exactly one top-level field") {
		t.Fatalf("expected multi-field error, got %v", err)
	}
}

func TestGraphQLTextExecutorRejectsAliasesFragmentsVariablesAndDirectives(t *testing.T) {
	tests := []struct {
		name     string
		document string
		want     string
	}{
		{
			name:     "alias",
			document: `query { search: web_search(query: "a") }`,
			want:     "does not support aliases",
		},
		{
			name:     "fragment",
			document: `query { ...SearchFrag } fragment SearchFrag on Query { web_search(query: "a") }`,
			want:     "does not support fragments",
		},
		{
			name:     "variables",
			document: `query Search($query: String!) { web_search(query: $query) }`,
			want:     "does not support variables",
		},
		{
			name:     "directives",
			document: `query @skip(if: false) { web_search(query: "a") }`,
			want:     "does not support directives",
		},
	}

	executor := NewGraphQLTextExecutor(graphQLToolRuntimeTestCatalog())
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := executor.Execute(context.Background(), testCase.document, "trace-graphql-invalid")
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("expected %q error, got %v", testCase.want, err)
			}
		})
	}
}

func TestGraphQLTextExecutorRejectsUnknownTool(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&graphQLToolRuntimeToolSearchTool{})
	executor := NewGraphQLTextExecutor(registry)

	_, err := executor.Execute(
		context.Background(),
		`query { missing_tool(query: "a") }`,
		"trace-graphql-missing",
	)
	if err == nil || !strings.Contains(err.Error(), `tool "missing_tool" not found`) {
		t.Fatalf("expected tool not found error, got %v", err)
	}
	protocolErr, ok := AsGraphQLTextProtocolError(err)
	if !ok {
		t.Fatalf("expected structured protocol error, got %T", err)
	}
	feedback := protocolErr.Feedback()
	if feedback.Kind != "unknown_tool" {
		t.Fatalf("unexpected feedback kind: %+v", feedback)
	}
	if !strings.Contains(feedback.Hint, "`tfind`") {
		t.Fatalf("expected tfind hint in feedback, got %+v", feedback)
	}
	if feedback.Example != `mutation { tfind(action: search, query: "browser control") }` {
		t.Fatalf("unexpected example: %+v", feedback)
	}
}

func TestGraphQLTextExecutorRejectsOperationMismatch(t *testing.T) {
	executor := NewGraphQLTextExecutor(graphQLToolRuntimeTestCatalog())

	_, err := executor.Execute(
		context.Background(),
		`query { ask_human(prompt: "Need approval?") }`,
		"trace-graphql-mismatch",
	)
	if err == nil || !strings.Contains(err.Error(), `tool "ask_human" must use mutation`) {
		t.Fatalf("expected operation mismatch error, got %v", err)
	}
	protocolErr, ok := AsGraphQLTextProtocolError(err)
	if !ok {
		t.Fatalf("expected structured protocol error, got %T", err)
	}
	feedback := protocolErr.Feedback()
	if feedback.Kind != "wrong_operation" {
		t.Fatalf("unexpected feedback kind: %+v", feedback)
	}
	if feedback.Tool != "ask_human" {
		t.Fatalf("unexpected feedback tool: %+v", feedback)
	}
	if feedback.Expected != "mutation" || feedback.Received != "query" {
		t.Fatalf("unexpected feedback operation payload: %+v", feedback)
	}
	if !protocolErr.Recoverable() {
		t.Fatalf("expected wrong operation to be recoverable")
	}
}

func TestGraphQLTextExecutorDefaultsUndeclaredToolsToMutation(t *testing.T) {
	executor := NewGraphQLTextExecutor(graphQLToolRuntimeTestCatalog())

	_, err := executor.Execute(
		context.Background(),
		`query { script_exec(script: "print('hi')") }`,
		"trace-graphql-default-mutation",
	)
	if err == nil || !strings.Contains(err.Error(), `tool "script_exec" must use mutation`) {
		t.Fatalf("expected default mutation mismatch error, got %v", err)
	}
}

func TestGraphQLTextExecutorSanitizesKnownGraphQLToolResultSuffix(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&graphQLToolRuntimeToolSearchTool{})
	executor := NewGraphQLTextExecutorWithOptions(registry, GraphQLTextExecutorOptions{
		SanitizeKnownArtifacts: true,
	})

	var logs bytes.Buffer
	originalWriter := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(originalWriter)

	result, err := executor.Execute(
		context.Background(),
		"mutation { tfind(action: list) }[GRAPHQL_TOOL_RESULT]\n{\"status\":\"error\"}",
		"trace-sanitize",
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Recognized || result.ToolName != ToolSearchToolName {
		t.Fatalf("unexpected result: %+v", result)
	}
	args := decodeGraphQLToolArgs(t, result.Arguments)
	if args["action"] != "list" {
		t.Fatalf("unexpected args: %+v", args)
	}
	if !strings.Contains(logs.String(), "trace_id=trace-sanitize") ||
		!strings.Contains(logs.String(), "kind=strip_graphql_tool_result_suffix") {
		t.Fatalf("expected sanitize log, got %q", logs.String())
	}
}

func TestGraphQLTextExecutorKeepsStrictParseWhenSanitizeDisabled(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&graphQLToolRuntimeToolSearchTool{})
	executor := NewGraphQLTextExecutorWithOptions(registry, GraphQLTextExecutorOptions{
		SanitizeKnownArtifacts: false,
	})

	_, err := executor.Execute(
		context.Background(),
		"mutation { tfind(action: list) }[GRAPHQL_TOOL_RESULT]\n{\"status\":\"error\"}",
		"trace-sanitize-disabled",
	)
	if err == nil {
		t.Fatal("expected parse error when sanitize is disabled")
	}
	protocolErr, ok := AsGraphQLTextProtocolError(err)
	if !ok {
		t.Fatalf("expected structured protocol error, got %T", err)
	}
	feedback := protocolErr.Feedback()
	if feedback.Kind != "parse_error" {
		t.Fatalf("unexpected feedback kind: %+v", feedback)
	}
	if !strings.Contains(feedback.Hint, "Return only a pure GraphQL document") {
		t.Fatalf("unexpected parse hint: %+v", feedback)
	}
	if feedback.Example == "" {
		t.Fatalf("expected parse example to be populated: %+v", feedback)
	}
}

func graphQLToolRuntimeTestCatalog() ToolCatalog {
	registry := NewRegistry()
	registry.Register(NewWebSearchTool(WebSearchConfig{}))
	registry.Register(NewAskHumanTool())
	registry.Register(&graphQLToolRuntimeNestedArgsTool{})
	return registry
}

type graphQLToolRuntimeNestedArgsTool struct{}

type graphQLToolRuntimeToolSearchTool struct{}

func (*graphQLToolRuntimeNestedArgsTool) Name() string {
	return "script_exec"
}

func (*graphQLToolRuntimeNestedArgsTool) Description() string {
	return "test script exec"
}

func (*graphQLToolRuntimeNestedArgsTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"script":{"type":"string"},
			"env":{"type":"object"},
			"files":{"type":"array","items":{"type":"string"}}
		},
		"required":["script"]
	}`)
}

func (*graphQLToolRuntimeNestedArgsTool) Execute(context.Context, json.RawMessage, string) (string, error) {
	return "", nil
}

func (*graphQLToolRuntimeToolSearchTool) Name() string {
	return ToolSearchToolName
}

func (*graphQLToolRuntimeToolSearchTool) Description() string {
	return "test tool search"
}

func (*graphQLToolRuntimeToolSearchTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"action":{"type":"string"},
			"query":{"type":"string"}
		}
	}`)
}

func (*graphQLToolRuntimeToolSearchTool) Execute(context.Context, json.RawMessage, string) (string, error) {
	return "", nil
}

func decodeGraphQLToolArgs(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()

	var args map[string]any
	if err := json.Unmarshal(raw, &args); err != nil {
		t.Fatalf("decode graphql args: %v", err)
	}
	return args
}
