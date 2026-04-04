package llm

import (
	"encoding/json"
	"strings"
)

// decodeJSONObject 确保 schema/arguments 统一按 JSON object 解码。
func decodeJSONObject(raw json.RawMessage) (map[string]any, error) {
	normalized := normalizeJSONObject(raw)
	out := make(map[string]any)
	if err := json.Unmarshal(normalized, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func decodeJSONObjectString(raw string) (map[string]any, error) {
	return decodeJSONObject(json.RawMessage(strings.TrimSpace(raw)))
}

func sanitizeCodexToolSchema(schema map[string]any) {
	for key, value := range schema {
		switch key {
		case "properties", "patternProperties", "$defs", "definitions", "dependentSchemas":
			schema[key] = sanitizeCodexSchemaMapEntries(value)
		default:
			schema[key] = sanitizeCodexToolSchemaValue(value)
		}
	}

	ty := codexSchemaType(schema)
	if ty == "" {
		ty = inferCodexSchemaType(schema)
	}
	if ty == "" {
		ty = "string"
	}
	if ty == "integer" {
		ty = "number"
	}
	schema["type"] = ty

	if ty == "object" {
		if _, ok := schema["properties"].(map[string]any); !ok {
			schema["properties"] = map[string]any{}
		}
		if additionalProperties, ok := schema["additionalProperties"]; ok {
			if _, isBool := additionalProperties.(bool); !isBool {
				schema["additionalProperties"] = sanitizeCodexToolSchemaValue(additionalProperties)
			}
		}
	}
	if ty == "array" {
		if _, ok := schema["items"]; !ok {
			schema["items"] = map[string]any{"type": "string"}
		}
	}
}

func sanitizeCodexSchemaMapEntries(value any) any {
	entries, ok := value.(map[string]any)
	if !ok {
		return value
	}
	for key, raw := range entries {
		entries[key] = sanitizeCodexToolSchemaValue(raw)
	}
	return entries
}

func sanitizeCodexToolSchemaValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		sanitizeCodexToolSchema(typed)
		return typed
	case []any:
		for index, item := range typed {
			typed[index] = sanitizeCodexToolSchemaValue(item)
		}
		return typed
	default:
		return value
	}
}

func codexSchemaType(schema map[string]any) string {
	raw, ok := schema["type"]
	if !ok {
		return ""
	}

	switch typed := raw.(type) {
	case string:
		return normalizeCodexSchemaType(typed)
	case []any:
		for _, candidate := range typed {
			asString, ok := candidate.(string)
			if !ok {
				continue
			}
			if normalized := normalizeCodexSchemaType(asString); normalized != "" {
				return normalized
			}
		}
	}
	return ""
}

func inferCodexSchemaType(schema map[string]any) string {
	switch {
	case schema["properties"] != nil || schema["required"] != nil || schema["additionalProperties"] != nil:
		return "object"
	case schema["items"] != nil || schema["prefixItems"] != nil:
		return "array"
	case schema["enum"] != nil || schema["const"] != nil || schema["format"] != nil:
		return "string"
	case schema["minimum"] != nil || schema["maximum"] != nil || schema["exclusiveMinimum"] != nil || schema["exclusiveMaximum"] != nil || schema["multipleOf"] != nil:
		return "number"
	default:
		return ""
	}
}

func normalizeCodexSchemaType(raw string) string {
	switch strings.TrimSpace(raw) {
	case "object", "array", "string", "number", "integer", "boolean":
		return strings.TrimSpace(raw)
	default:
		return ""
	}
}
