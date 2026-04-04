package tools

import (
	"context"
	"strings"
	"testing"
)

func TestBuildGraphQLToolRuntimeSchema_GeneratesInputAndEnumTypes(t *testing.T) {
	registry := NewRegistry()
	registry.Register(NewAskHumanTool())
	registry.Register(NewBrowserControlTool(nil))
	registry.Register(NewTaskManageTool(nil))
	registry.Register(NewComputerUseTool(nil, nil, nil))
	registry.Register(NewWebRooterTool(WebRooterConfig{}))

	schema := BuildGraphQLToolRuntimeSchema(registry)

	browserField := findGraphQLToolRuntimeField(t, schema.MutationFields, "browser_control")
	if got := findGraphQLToolRuntimeArgument(t, browserField.Arguments, "action").Type; got != "BrowserControlActionEnum!" {
		t.Fatalf("expected browser_control action enum type, got %q", got)
	}
	if got := findGraphQLToolRuntimeArgument(t, browserField.Arguments, "params").Type; got != "BrowserControlParamsInput" {
		t.Fatalf("expected browser_control params input type, got %q", got)
	}

	enumType := findGraphQLToolRuntimeEnumType(t, schema.EnumTypes, "BrowserControlParamsWaitEnum")
	if got := graphQLToolRuntimeEnumValueNames(enumType.Values); strings.Join(got, ",") != "none,dom,load" {
		t.Fatalf("unexpected wait enum values: %v", got)
	}

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
	registry.Register(NewBrowserControlTool(nil))

	executor := NewGraphQLTextExecutor(registry)
	result, err := executor.Execute(
		context.Background(),
		`<t:1>{"action":"goto","params":{"session_id":"sess-1","selector_type":"css","wait":"load","timeout_ms":500}}</t>`,
		"trace-tag-browser-object",
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	args := decodeGraphQLToolArgs(t, result.Arguments)
	if got := args["action"]; got != "goto" {
		t.Fatalf("expected action to decode as string, got %+v", args)
	}
	params, ok := args["params"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested params object, got %+v", args)
	}
	if got := params["selector_type"]; got != "css" {
		t.Fatalf("expected nested selector_type to decode as string, got %+v", params)
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

func findGraphQLToolRuntimeEnumType(
	t *testing.T,
	enumTypes []GraphQLToolRuntimeEnumType,
	name string,
) GraphQLToolRuntimeEnumType {
	t.Helper()
	for _, enumType := range enumTypes {
		if enumType.Name == name {
			return enumType
		}
	}
	t.Fatalf("enum type %q not found", name)
	return GraphQLToolRuntimeEnumType{}
}

func graphQLToolRuntimeEnumValueNames(values []GraphQLToolRuntimeEnumValue) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.Name)
	}
	return out
}
