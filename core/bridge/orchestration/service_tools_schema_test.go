package orchestration

import "testing"

func TestSchemaByToolNameReturnsMissingWhenToolHasNoSchema(t *testing.T) {
	schema, ok := schemaByToolName(map[string]map[string]any{
		"script_exec": {"type": "object"},
	}, "web_search")
	if ok {
		t.Fatal("expected missing schema flag for tool without schema")
	}
	if schema != nil {
		t.Fatalf("expected nil schema for tool without schema, got %#v", schema)
	}
}

func TestSchemaByToolNameReturnsSchemaWhenPresent(t *testing.T) {
	schema, ok := schemaByToolName(map[string]map[string]any{
		"script_exec": {"type": "object"},
	}, "script_exec")
	if !ok {
		t.Fatal("expected schema to exist")
	}
	if got, ok := schema["type"].(string); !ok || got != "object" {
		t.Fatalf("expected object schema type, got %#v", schema["type"])
	}
}
