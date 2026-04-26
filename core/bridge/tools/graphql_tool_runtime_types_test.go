package tools

import (
	"context"
	"strings"
	"testing"
)

func TestBuildGraphQLToolRuntimeSchema_GeneratesInputAndEnumTypes(t *testing.T) {
	registry := NewRegistry()
	registry.Register(NewAskHumanTool())

	schema := BuildGraphQLToolRuntimeSchema(registry)

	inputType := findGraphQLToolRuntimeInputType(t, schema.InputTypes, "AskHumanOptionsItemInput")
	if got := findGraphQLToolRuntimeArgument(t, inputType.Fields, "label").Type; got != "String!" {
		t.Fatalf("expected ask_human option label to stay required, got %q", got)
	}
}

func TestFormatGraphQLToolRuntimePrompt_UsesIDAndJSONShape(t *testing.T) {
	registry := NewRegistry()
	registry.Register(NewAskHumanTool())
	registry.Register(NewWebSearchTool(WebSearchConfig{}))

	prompt := FormatGraphQLToolRuntimePrompt(registry)
	for _, snippet := range []string{
		"[System Instruction]",
		"<t:TOOL_ID>JSON_ARGS</t>",
		"ID: 1",
		"Tool name: ask_human",
		"Tool name: web_search",
		"Parameter format:",
	} {
		if !strings.Contains(prompt, snippet) {
			t.Fatalf("expected prompt to contain %q, got %q", snippet, prompt)
		}
	}
	if strings.Contains(prompt, "type Mutation {") || strings.Contains(prompt, "enum ") {
		t.Fatalf("expected prompt to avoid GraphQL schema blocks, got %q", prompt)
	}
}

func TestGraphQLTextExecutorAcceptsTaggedNestedInputObjectsWhenSchemaAllows(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&tagToolRuntimeNestedArgsTool{})

	executor := NewGraphQLTextExecutor(registry)
	result, err := executor.Execute(
		context.Background(),
		`<t:1>{"location":{"city":"Beijing"}}</t>`,
		"trace-tag-nested-allowed",
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	args := decodeGraphQLToolArgs(t, result.Arguments)
	location, ok := args["location"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested location object, got %+v", args)
	}
	if got := location["city"]; got != "Beijing" {
		t.Fatalf("expected nested city to decode as string, got %+v", location)
	}
}

func findGraphQLToolRuntimeField(
	t *testing.T,
	fields []GraphQLToolRuntimeField,
	name string,
) GraphQLToolRuntimeField {
	t.Helper()
	for _, field := range fields {
		if field.Name == name {
			return field
		}
	}
	t.Fatalf("field %q not found", name)
	return GraphQLToolRuntimeField{}
}

func findGraphQLToolRuntimeArgument(
	t *testing.T,
	arguments []GraphQLToolRuntimeArgument,
	name string,
) GraphQLToolRuntimeArgument {
	t.Helper()
	for _, argument := range arguments {
		if argument.Name == name {
			return argument
		}
	}
	t.Fatalf("argument %q not found", name)
	return GraphQLToolRuntimeArgument{}
}

func findGraphQLToolRuntimeInputType(
	t *testing.T,
	inputTypes []GraphQLToolRuntimeInputType,
	name string,
) GraphQLToolRuntimeInputType {
	t.Helper()
	for _, inputType := range inputTypes {
		if inputType.Name == name {
			return inputType
		}
	}
	t.Fatalf("input type %q not found", name)
	return GraphQLToolRuntimeInputType{}
}
