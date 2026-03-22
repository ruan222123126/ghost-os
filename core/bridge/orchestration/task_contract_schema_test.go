package orchestration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestTaskSchemaDefinesKindSpecificContracts(t *testing.T) {
	defs := loadTaskSchemaDefs(t)
	assertSchemaOneOfRefs(t, defs, "taskCreateRequest",
		"agentMessageTaskCreateRequest",
		"rssInboxPollTaskCreateRequest",
		"rssBriefingTaskCreateRequest",
		"workflowTaskCreateRequest",
	)
	assertSchemaOneOfRefs(t, defs, "taskPayload",
		"agentMessageTaskPayload",
		"rssInboxPollTaskPayload",
		"rssBriefingTaskPayload",
		"workflowTaskPayload",
	)
	assertSchemaRequired(t, defs, "agentMessageTaskCreateRequest", "message")
	assertSchemaRequired(t, defs, "workflowTaskCreateRequest", "workflow")
	assertSchemaRequired(t, defs, "taskUpdateRequest", "id")
	assertSchemaProperty(t, defs, "taskUpdateRequest", "action")
	assertSchemaProperty(t, defs, "taskUpdateRequest", "action_params")
	assertSchemaProperty(t, defs, "workflowNode", "tool")
	assertSchemaProperty(t, defs, "workflowNode", "llm")
	assertSchemaProperty(t, defs, "workflowNode", "agent")
	assertSchemaRequired(t, defs, "workflowToolNode", "tool_name")
	assertSchemaRequired(t, defs, "workflowLLMNode", "prompt")
	assertSchemaRequired(t, defs, "workflowAgentNode", "message")
	assertSchemaConst(t, defs, "rssInboxPollTaskCreateRequest", "action", busActionRSSInboxPoll)
	assertSchemaConst(t, defs, "rssBriefingTaskPayload", "action", busActionRSSBriefingBuild)
}

func loadTaskSchemaDefs(t *testing.T) map[string]any {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve caller path")
	}
	path := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "shared", "schema", "defs", "tasks.json"))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read tasks schema: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("decode tasks schema: %v", err)
	}
	defs, ok := document["$defs"].(map[string]any)
	if !ok {
		t.Fatal("tasks schema defs missing")
	}
	return defs
}

func assertSchemaOneOfRefs(t *testing.T, defs map[string]any, name string, want ...string) {
	t.Helper()
	definition := schemaDefinition(t, defs, name)
	items, ok := definition["oneOf"].([]any)
	if !ok || len(items) != len(want) {
		t.Fatalf("unexpected oneOf in %s: %#v", name, definition["oneOf"])
	}
	for index, ref := range want {
		item, ok := items[index].(map[string]any)
		if !ok || item["$ref"] != "#/$defs/"+ref {
			t.Fatalf("unexpected oneOf[%d] in %s: %#v", index, name, items[index])
		}
	}
}

func assertSchemaRequired(t *testing.T, defs map[string]any, name string, field string) {
	t.Helper()
	required, ok := schemaDefinition(t, defs, name)["required"].([]any)
	if !ok {
		t.Fatalf("required missing in %s", name)
	}
	for _, item := range required {
		if item == field {
			return
		}
	}
	t.Fatalf("field %q is not required in %s: %#v", field, name, required)
}

func assertSchemaProperty(t *testing.T, defs map[string]any, name string, field string) {
	t.Helper()
	properties := schemaProperties(t, defs, name)
	if _, ok := properties[field]; !ok {
		t.Fatalf("field %q missing in %s", field, name)
	}
}

func assertSchemaConst(t *testing.T, defs map[string]any, name string, field string, want string) {
	t.Helper()
	property, ok := schemaProperties(t, defs, name)[field].(map[string]any)
	if !ok || property["const"] != want {
		t.Fatalf("unexpected const in %s.%s: %#v", name, field, property)
	}
}

func schemaDefinition(t *testing.T, defs map[string]any, name string) map[string]any {
	t.Helper()
	definition, ok := defs[name].(map[string]any)
	if !ok {
		t.Fatalf("schema definition %q missing", name)
	}
	return definition
}

func schemaProperties(t *testing.T, defs map[string]any, name string) map[string]any {
	t.Helper()
	properties, ok := schemaDefinition(t, defs, name)["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties missing in %s", name)
	}
	return properties
}
