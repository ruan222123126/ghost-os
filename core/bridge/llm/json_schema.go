package llm

import (
	"encoding/json"
	"fmt"
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

func sanitizeCodexToolSchema(schema map[string]any) error {
	return sanitizeStrictGatewayToolSchema(schema, "codex responses function tools")
}

func sanitizeToolSchemaNode(schema map[string]any) {
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

func sanitizeOpenAIToolParameters(raw json.RawMessage, toolName string) (json.RawMessage, error) {
	schema, err := decodeJSONObject(raw)
	if err != nil {
		return nil, fmt.Errorf("decode schema for tool %q: %w", toolName, err)
	}
	if err := sanitizeOpenAIToolSchema(schema); err != nil {
		return nil, fmt.Errorf("sanitize schema for tool %q: %w", toolName, err)
	}
	encoded, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("encode schema for tool %q: %w", toolName, err)
	}
	return encoded, nil
}

// OpenAI 兼容 chat completions 的 function tools 要求顶层必须是 object，
// 且不接受 oneOf/anyOf/allOf/enum/not 这类顶层组合关键字。
func sanitizeOpenAIToolSchema(schema map[string]any) error {
	return sanitizeStrictGatewayToolSchema(schema, "openai chat completions")
}

func sanitizeStrictGatewayToolSchema(schema map[string]any, gatewayName string) error {
	sanitizeToolSchemaNode(schema)

	topLevelType := codexSchemaType(schema)
	if topLevelType == "" {
		topLevelType = inferCodexSchemaType(schema)
	}
	if topLevelType != "" && topLevelType != "object" {
		return fmt.Errorf("top-level tool schema type %q is not supported by %s", topLevelType, gatewayName)
	}

	delete(schema, "oneOf")
	delete(schema, "anyOf")
	delete(schema, "allOf")
	delete(schema, "enum")
	delete(schema, "not")
	schema["type"] = "object"

	if _, ok := schema["properties"].(map[string]any); !ok {
		schema["properties"] = map[string]any{}
	}
	if additionalProperties, ok := schema["additionalProperties"]; ok {
		if _, isBool := additionalProperties.(bool); !isBool {
			schema["additionalProperties"] = sanitizeCodexToolSchemaValue(additionalProperties)
		}
	}
	return nil
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
		sanitizeToolSchemaNode(typed)
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
