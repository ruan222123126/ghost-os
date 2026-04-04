package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
)

func TestGraphQLTextExecutorRecognizesSingleTaggedToolCall(t *testing.T) {
	executor := NewGraphQLTextExecutor(tagToolRuntimeTestCatalog())

	result, err := executor.Execute(
		context.Background(),
		`<t:3>{"query":"OpenAI latest news","max_results":5}</t>`,
		"trace-tag-single",
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Recognized {
		t.Fatalf("expected recognized result, got %+v", result)
	}
	if result.ToolName != "web_search" {
		t.Fatalf("unexpected tool name: %+v", result)
	}
	args := decodeGraphQLToolArgs(t, result.Arguments)
	if args["query"] != "OpenAI latest news" || args["max_results"] != float64(5) {
		t.Fatalf("unexpected args: %+v", args)
	}
}

func TestGraphQLTextExecutorRecognizesMultipleTaggedToolCalls(t *testing.T) {
	executor := NewGraphQLTextExecutor(tagToolRuntimeTestCatalog())

	result, err := executor.Execute(
		context.Background(),
		`<t:3>{"query":"OpenAI"}</t><t:1>{"prompt":"Need approval?"}</t>`,
		"trace-tag-multi",
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(result.Calls) != 2 {
		t.Fatalf("unexpected call count: %+v", result.Calls)
	}
	if result.Calls[0].ToolName != "web_search" || result.Calls[1].ToolName != "ask_human" {
		t.Fatalf("unexpected call tool names: %+v", result.Calls)
	}
	first := decodeGraphQLToolArgs(t, result.Calls[0].Arguments)
	second := decodeGraphQLToolArgs(t, result.Calls[1].Arguments)
	if first["query"] != "OpenAI" {
		t.Fatalf("unexpected first args: %+v", first)
	}
	if second["prompt"] != "Need approval?" {
		t.Fatalf("unexpected second args: %+v", second)
	}
}

func TestGraphQLTextExecutorRejectsUnknownToolID(t *testing.T) {
	executor := NewGraphQLTextExecutor(tagToolRuntimeTestCatalog())

	_, err := executor.Execute(
		context.Background(),
		`<t:99>{"query":"OpenAI"}</t>`,
		"trace-tag-unknown-id",
	)
	if err == nil || !strings.Contains(err.Error(), `tool id 99 not found`) {
		t.Fatalf("expected unknown tool id error, got %v", err)
	}
	protocolErr, ok := AsGraphQLTextProtocolError(err)
	if !ok {
		t.Fatalf("expected structured protocol error, got %T", err)
	}
	if protocolErr.Feedback().Kind != "unknown_tool_id" {
		t.Fatalf("unexpected feedback: %+v", protocolErr.Feedback())
	}
}

func TestGraphQLTextExecutorRejectsLegacyGraphQLMutationText(t *testing.T) {
	executor := NewGraphQLTextExecutor(tagToolRuntimeTestCatalog())

	result, err := executor.Execute(
		context.Background(),
		`mutation { web_search(query: "OpenAI") }`,
		"trace-tag-legacy-mutation",
	)
	if err == nil || !strings.Contains(err.Error(), "legacy GraphQL mutation/query tool calls are no longer supported") {
		t.Fatalf("expected legacy protocol rejection, got result=%+v err=%v", result, err)
	}
	if !result.Recognized {
		t.Fatalf("expected legacy mutation text to be recognized as protocol violation")
	}
	protocolErr, ok := AsGraphQLTextProtocolError(err)
	if !ok || protocolErr.Feedback().Kind != "parse_error" {
		t.Fatalf("expected parse_error feedback, got %T %+v", err, protocolErr)
	}
}

func TestGraphQLTextExecutorRejectsNonObjectArguments(t *testing.T) {
	executor := NewGraphQLTextExecutor(tagToolRuntimeTestCatalog())

	_, err := executor.Execute(
		context.Background(),
		`<t:3>["not-object"]</t>`,
		"trace-tag-bad-args",
	)
	if err == nil || !strings.Contains(err.Error(), "tool arguments must be a JSON object") {
		t.Fatalf("expected JSON object error, got %v", err)
	}
}

func TestGraphQLTextExecutorAllowsTagWithoutClosingTerminatorAtStreamEnd(t *testing.T) {
	executor := NewGraphQLTextExecutor(tagToolRuntimeTestCatalog())

	result, err := executor.Execute(
		context.Background(),
		`<t:3>{"query":"OpenAI"}`,
		"trace-tag-stream-end",
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(result.Calls) != 1 || result.Calls[0].ToolName != "web_search" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestGraphQLTextExecutorKeepsFlatPreferredAndAllowsSchemaObject(t *testing.T) {
	executor := NewGraphQLTextExecutor(tagToolRuntimeTestCatalog())

	_, err := executor.Execute(
		context.Background(),
		`<t:3>{"query":{"text":"OpenAI"}}</t>`,
		"trace-tag-flat-preferred",
	)
	if err == nil || !strings.Contains(err.Error(), "should stay flat") {
		t.Fatalf("expected flat argument error, got %v", err)
	}

	result, err := executor.Execute(
		context.Background(),
		`<t:2>{"location":{"city":"Beijing"}}</t>`,
		"trace-tag-nested-allowed",
	)
	if err != nil {
		t.Fatalf("expected nested object to be allowed by schema, got %v", err)
	}
	args := decodeGraphQLToolArgs(t, result.Arguments)
	location, ok := args["location"].(map[string]any)
	if !ok || location["city"] != "Beijing" {
		t.Fatalf("unexpected nested args: %+v", args)
	}
}

func decodeGraphQLToolArgs(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode args: %v", err)
	}
	return out
}

func tagToolRuntimeTestCatalog() ToolCatalog {
	registry := NewRegistry()
	registry.Register(NewAskHumanTool())
	registry.Register(&tagToolRuntimeNestedArgsTool{})
	registry.Register(NewWebSearchTool(WebSearchConfig{}))
	return registry
}

type tagToolRuntimeNestedArgsTool struct{}

func (*tagToolRuntimeNestedArgsTool) Name() string {
	return "send_email"
}

func (*tagToolRuntimeNestedArgsTool) Description() string {
	return "send email"
}

func (*tagToolRuntimeNestedArgsTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"location":{
				"type":"object",
				"properties":{
					"city":{"type":"string"}
				},
				"required":["city"]
			}
		},
		"required":["location"]
	}`)
}

func (*tagToolRuntimeNestedArgsTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}

func (*tagToolRuntimeNestedArgsTool) Execute(context.Context, json.RawMessage, string) (string, error) {
	return `{"status":"ok"}`, nil
}
