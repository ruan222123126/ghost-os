package tools

import (
	"encoding/json"
	"testing"
)

func TestRelayToolParametersAreValidJSON(t *testing.T) {
	cases := []struct {
		name        string
		tool        Tool
		finalFields bool
	}{
		{name: "update", tool: NewRelayUpdateRecordTool()},
		{name: "complete", tool: NewRelayCompleteTool(), finalFields: true},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			schema := decodeRelayToolSchema(t, tt.tool.Parameters())
			properties := schemaProperties(t, schema)
			assertRelayBaseProperties(t, properties)
			_, hasFinalMessage := properties["final_message"]
			_, hasFinalChangeLog := properties["final_change_log"]
			if hasFinalMessage != tt.finalFields || hasFinalChangeLog != tt.finalFields {
				t.Fatalf("unexpected final field presence: final_message=%v final_change_log=%v", hasFinalMessage, hasFinalChangeLog)
			}
		})
	}
}

func decodeRelayToolSchema(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()

	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("relay tool schema must be valid JSON: %v\nraw=%s", err, string(raw))
	}
	if schema["type"] != "object" {
		t.Fatalf("relay tool schema type = %#v, want object", schema["type"])
	}
	return schema
}

func schemaProperties(t *testing.T, schema map[string]any) map[string]any {
	t.Helper()

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("relay tool schema properties missing: %#v", schema["properties"])
	}
	return properties
}

func assertRelayBaseProperties(t *testing.T, properties map[string]any) {
	t.Helper()

	for _, name := range []string{"did", "remaining", "failed_attempts", "next_step"} {
		if _, ok := properties[name]; !ok {
			t.Fatalf("relay tool schema missing property %q", name)
		}
	}
}
