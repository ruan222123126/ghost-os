package tools

import (
	"context"
	"strings"
	"testing"
)

func TestBuildGraphQLToolRuntimeSchema_GeneratesInputAndEnumTypes(t *testing.T) {
	registry := NewRegistry()
	registry.Register(NewAskHumanTool())
	registry.Register(NewTaskManageTool(nil))
	registry.Register(NewWebRooterTool(WebRooterConfig{}))

	schema := BuildGraphQLToolRuntimeSchema(registry)

	webRooterField := findGraphQLToolRuntimeField(t, schema.MutationFields, "web_rooter")
	if got := findGraphQLToolRuntimeArgument(t, webRooterField.Arguments, "action").Type; got != "WebRooterActionEnum!" {
		t.Fatalf("expected web_rooter action enum type, got %q", got)
	}
	if got := findGraphQLToolRuntimeArgument(t, webRooterField.Arguments, "params").Type; got != "WebRooterParamsInput!" {
		t.Fatalf("expected web_rooter params input type, got %q", got)
	}

	enumType := findGraphQLToolRuntimeEnumType(t, schema.EnumTypes, "WebRooterActionEnum")
	if got := graphQLToolRuntimeEnumValueNames(enumType.Values); strings.Join(got, ",") != "internet_search,research,academic_search,site_search,fetch,extract" {
		t.Fatalf("unexpected action enum values: %v", got)
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
	registry.Register(NewWebRooterTool(WebRooterConfig{}))

	executor := NewGraphQLTextExecutor(registry)
	result, err := executor.Execute(
		context.Background(),
		`<t:1>{"action":"fetch","params":{"url":"https://example.com","use_browser":false}}</t>`,
		"trace-tag-web-rooter-object",
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	args := decodeGraphQLToolArgs(t, result.Arguments)
	if got := args["action"]; got != "fetch" {
		t.Fatalf("expected action to decode as string, got %+v", args)
	}
	params, ok := args["params"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested params object, got %+v", args)
	}
	if got := params["use_browser"]; got != false {
		t.Fatalf("expected nested use_browser to decode as bool, got %+v", params)
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
