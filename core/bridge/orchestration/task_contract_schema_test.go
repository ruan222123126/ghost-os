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
		"workflowTaskCreateRequest",
		"orchestrationTaskCreateRequest",
	)
	assertSchemaOneOfRefs(t, defs, "taskPayload",
		"agentMessageTaskPayload",
		"workflowTaskPayload",
		"orchestrationTaskPayload",
	)
	assertSchemaRequired(t, defs, "agentMessageTaskCreateRequest", "message")
	assertSchemaRequired(t, defs, "workflowTaskCreateRequest", "workflow")
	assertSchemaRequired(t, defs, "orchestrationTaskCreateRequest", "name")
	assertSchemaRequired(t, defs, "orchestrationTaskCreateRequest", "orchestration")
	assertSchemaRequired(t, defs, "taskUpdateRequest", "id")
	assertSchemaProperty(t, defs, "taskUpdateRequest", "name")
	assertSchemaProperty(t, defs, "taskRuntimeOverrides", "provider_name")
	assertSchemaProperty(t, defs, "taskRuntimeOverrides", "model")
	assertSchemaProperty(t, defs, "taskRuntimeOverrides", "system_prompt")
	assertSchemaProperty(t, defs, "taskRuntimeOverrides", "preset_id")
	assertSchemaProperty(t, defs, "taskRuntimeOverrides", "tool_allowlist")
	assertSchemaProperty(t, defs, "taskRuntimeOverrides", "tool_allowlist_only")
	assertSchemaProperty(t, defs, "taskRuntimeOverrides", "max_turns")
	assertSchemaProperty(t, defs, "agentMessageTaskCreateRequest", "runtime_overrides")
	assertSchemaProperty(t, defs, "taskUpdateRequest", "runtime_overrides")
	assertSchemaProperty(t, defs, "agentMessageTaskPayload", "runtime_overrides")
	assertSchemaProperty(t, defs, "workflowNode", "start")
	assertSchemaProperty(t, defs, "workflowNode", "tool")
	assertSchemaProperty(t, defs, "workflowNode", "llm")
	assertSchemaProperty(t, defs, "workflowNode", "agent")
	assertSchemaProperty(t, defs, "workflowNode", "if")
	assertSchemaProperty(t, defs, "workflowNode", "loop")
	assertSchemaProperty(t, defs, "orchestrationNode", "group")
	assertSchemaProperty(t, defs, "orchestrationNode", "agent")
	assertSchemaProperty(t, defs, "workflowStartNode", "inputs")
	assertSchemaRequired(t, defs, "workflowInputVariable", "name")
	assertSchemaRequired(t, defs, "workflowInputVariable", "type")
	assertSchemaProperty(t, defs, "workflowInputVariable", "default")
	assertSchemaRequired(t, defs, "workflowToolNode", "tool_name")
	assertSchemaRequired(t, defs, "workflowLLMNode", "prompt")
	assertSchemaRequired(t, defs, "workflowAgentNode", "message")
	assertSchemaProperty(t, defs, "workflowAgentNode", "runtime_overrides")
	assertSchemaRequired(t, defs, "workflowIfNode", "operator")
	assertSchemaRequired(t, defs, "workflowIfNode", "true_node_id")
	assertSchemaRequired(t, defs, "workflowIfNode", "false_node_id")
	assertSchemaRequired(t, defs, "workflowLoopNode", "max_iterations")
	assertSchemaRequired(t, defs, "workflowLoopNode", "body_node_id")
	assertSchemaRequired(t, defs, "workflowLoopNode", "exit_node_id")
	assertSchemaRequired(t, defs, "taskRelayConfig", "max_rounds")
	assertSchemaRequired(t, defs, "orchestrationGroupNode", "title")
	assertSchemaRequired(t, defs, "orchestrationGroupNode", "shared_context")
	assertSchemaRequired(t, defs, "orchestrationGroupNode", "speaking_mode")
	assertSchemaRequired(t, defs, "orchestrationGroupNode", "max_rounds")
	assertSchemaProperty(t, defs, "orchestrationGroupNode", "owner_agent_id")
	assertSchemaRequired(t, defs, "orchestrationAgentNode", "title")
	assertSchemaRequired(t, defs, "orchestrationAgentNode", "message")
	assertSchemaRequired(t, defs, "orchestrationEdge", "kind")
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
