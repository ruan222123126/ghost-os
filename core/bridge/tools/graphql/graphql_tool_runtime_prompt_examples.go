package graphql

import "strings"

const toolParameterExampleDateISO = `"2026-03-29"`

func formatToolParameterTypeLabel(schema map[string]any) string {
	typeName := strings.ToLower(strings.TrimSpace(graphQLSchemaTypeName(schema)))
	switch typeName {
	case "string":
		return "string"
	case "integer":
		return "int"
	case "number":
		return "number"
	case "boolean":
		return "bool"
	case "array":
		return "array"
	case "object":
		return "object"
	default:
		if _, ok := schema["properties"].(map[string]any); ok {
			return "object"
		}
		if enumValues, ok := graphQLEnumValues(schema); ok && len(enumValues) > 0 {
			return "string"
		}
		return "json"
	}
}

func toolParameterExampleForName(name string, schema map[string]any) string {
	trimmed := strings.ToLower(strings.TrimSpace(name))
	switch {
	case strings.Contains(trimmed, "city"):
		return `"Beijing"`
	case strings.Contains(trimmed, "date"):
		return toolParameterExampleDateISO
	case strings.Contains(trimmed, "day"):
		return "1"
	case strings.Contains(trimmed, "email"), strings.Contains(trimmed, "to"):
		return `"boss@example.com"`
	case strings.Contains(trimmed, "title"):
		return `"Daily report"`
	case strings.Contains(trimmed, "body"):
		return `"Summary text"`
	case strings.Contains(trimmed, "query"):
		return `"OpenAI API docs"`
	case strings.Contains(trimmed, "url"):
		return `"https://example.com"`
	}
	typeName := strings.ToLower(strings.TrimSpace(graphQLSchemaTypeName(schema)))
	if typeName == "boolean" {
		return "true"
	}
	if typeName == "integer" || typeName == "number" {
		return "1"
	}
	return ""
}
