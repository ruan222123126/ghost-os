package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestGraphQLTextExecutorRecognizesQueryToolCall(t *testing.T) {
	executor := NewGraphQLTextExecutor(graphQLToolRuntimeTestCatalog()).(*graphQLTextExecutor)

	result, err := executor.Execute(
		context.Background(),
		`query { web_search(query: "OpenAI latest news", max_results: 5) }`,
		"trace-graphql-query",
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
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
	executor := NewGraphQLTextExecutor(graphQLToolRuntimeTestCatalog())

	_, err := executor.Execute(
		context.Background(),
		`query { missing_tool(query: "a") }`,
		"trace-graphql-missing",
	)
	if err == nil || !strings.Contains(err.Error(), `tool "missing_tool" not found`) {
		t.Fatalf("expected tool not found error, got %v", err)
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

func graphQLToolRuntimeTestCatalog() ToolCatalog {
	registry := NewRegistry()
	registry.Register(NewWebSearchTool(WebSearchConfig{}))
	registry.Register(NewAskHumanTool())
	registry.Register(&graphQLToolRuntimeNestedArgsTool{})
	return registry
}

type graphQLToolRuntimeNestedArgsTool struct{}

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

func decodeGraphQLToolArgs(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()

	var args map[string]any
	if err := json.Unmarshal(raw, &args); err != nil {
		t.Fatalf("decode graphql args: %v", err)
	}
	return args
}
