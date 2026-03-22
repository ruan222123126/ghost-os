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

	schema := BuildGraphQLToolRuntimeSchema(registry)

	browserField := findGraphQLToolRuntimeField(t, schema.MutationFields, "browser_control")
	if got := findGraphQLToolRuntimeArgument(t, browserField.Arguments, "action").Type; got != "BrowserControlActionEnum!" {
		t.Fatalf("expected browser_control action enum type, got %q", got)
	}
	if got := findGraphQLToolRuntimeArgument(t, browserField.Arguments, "params").Type; got != "BrowserControlParamsInput" {
		t.Fatalf("expected browser_control params input type, got %q", got)
	}

	taskField := findGraphQLToolRuntimeField(t, schema.MutationFields, "task_manage")
	if got := findGraphQLToolRuntimeArgument(t, taskField.Arguments, "operation").Type; got != "TaskManageOperationEnum!" {
		t.Fatalf("expected task_manage operation enum type, got %q", got)
	}

	computerField := findGraphQLToolRuntimeField(t, schema.MutationFields, "computer_use")
	if got := findGraphQLToolRuntimeArgument(t, computerField.Arguments, "mode").Type; got != "ComputerUseModeEnum" {
		t.Fatalf("expected computer_use mode enum type, got %q", got)
	}
	if got := findGraphQLToolRuntimeArgument(t, computerField.Arguments, "target").Type; got != "ComputerUseTargetInput" {
		t.Fatalf("expected computer_use target input type, got %q", got)
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

func TestFormatGraphQLToolRuntimePrompt_IncludesStructuredInputSchema(t *testing.T) {
	registry := NewRegistry()
	registry.Register(NewAskHumanTool())
	registry.Register(NewBrowserControlTool(nil))
	registry.Register(NewTaskManageTool(nil))
	registry.Register(NewComputerUseTool(nil, nil, nil))

	prompt := FormatGraphQLToolRuntimePrompt(registry)
	for _, snippet := range []string{
		"enum BrowserControlActionEnum {",
		"enum BrowserControlParamsWaitEnum {",
		"input BrowserControlParamsInput {",
		"input AskHumanOptionsItemInput {",
		"input ComputerUseTargetInput {",
		"browser_control(action: BrowserControlActionEnum!, params: BrowserControlParamsInput): JSON",
		"computer_use(display_id: Int, goal: String!, mode: ComputerUseModeEnum, target: ComputerUseTargetInput): JSON",
		"task_manage(cron_expr: String, enabled: Boolean, id: String, interval_seconds: Int, message: String, operation: TaskManageOperationEnum!, session_id: String): JSON",
	} {
		if !strings.Contains(prompt, snippet) {
			t.Fatalf("expected prompt to contain %q, got %q", snippet, prompt)
		}
	}
	if strings.Contains(prompt, "browser_control(action: String") {
		t.Fatalf("expected browser_control action to stop degrading into String, got %q", prompt)
	}
	if strings.Contains(prompt, "browser_control(action: BrowserControlActionEnum!, params: JSON)") {
		t.Fatalf("expected browser_control params to stay structured, got %q", prompt)
	}
}

func TestGraphQLTextExecutorAcceptsEnumLiteralsAndNestedInputObjects(t *testing.T) {
	registry := NewRegistry()
	registry.Register(NewBrowserControlTool(nil))

	executor := NewGraphQLTextExecutor(registry)
	result, err := executor.Execute(
		context.Background(),
		`mutation { browser_control(action: goto, params: {session_id: "sess-1", selector_type: css, wait: load, timeout_ms: 500}) }`,
		"trace-graphql-browser-enum",
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	args := decodeGraphQLToolArgs(t, result.Arguments)
	if got := args["action"]; got != "goto" {
		t.Fatalf("expected enum literal to decode as original string, got %+v", args)
	}
	params, ok := args["params"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested params object, got %+v", args)
	}
	if got := params["selector_type"]; got != "css" {
		t.Fatalf("expected nested enum literal to decode as original string, got %+v", params)
	}
	if got := params["wait"]; got != "load" {
		t.Fatalf("expected wait enum literal to decode as original string, got %+v", params)
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
