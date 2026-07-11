package tools

import (
	"encoding/json"
	"fmt"

	"ghost-os/bridge/llm"
)

func CollectSchemas(defs []llm.ToolDef) (map[string]map[string]any, error) {
	schemas := make(map[string]map[string]any, len(defs))
	for _, def := range defs {
		schema, err := DecodeSchema(def.Parameters, def.Name)
		if err != nil {
			return nil, err
		}
		schemas[def.Name] = schema
	}
	return schemas, nil
}

func DecodeSchema(raw json.RawMessage, toolName string) (map[string]any, error) {
	var schema map[string]any
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil, fmt.Errorf("decode schema for tool %q: %w", toolName, err)
	}
	if schema == nil {
		return map[string]any{}, nil
	}
	return schema, nil
}

func SchemaByToolName(schemasByName map[string]map[string]any, toolName string) map[string]any {
	if len(schemasByName) == 0 {
		return nil
	}
	schema, ok := schemasByName[toolName]
	if !ok || len(schema) == 0 {
		return nil
	}
	return schema
}
