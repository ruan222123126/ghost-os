package tools

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"

	"ghost-os/bridge/llm"
)

type graphQLToolRuntimeExample struct {
	Note     string
	Document string
}

func writeGraphQLToolExamplesBlock(builder *strings.Builder, defs []llm.ToolDef) {
	if len(defs) == 0 {
		return
	}

	builder.WriteString("\n\nMinimal successful examples:\n")
	for _, def := range defs {
		example := graphQLToolRuntimeExampleForDef(def)
		builder.WriteString("- `")
		builder.WriteString(def.Name)
		builder.WriteString("`")
		builder.WriteString(":")
		if example.Note != "" {
			builder.WriteString(" ")
			builder.WriteString(example.Note)
		}
		builder.WriteString(" `")
		builder.WriteString(example.Document)
		builder.WriteString("`\n")
	}
}

func graphQLToolRuntimeExampleForDef(def llm.ToolDef) graphQLToolRuntimeExample {
	if example, ok := graphQLSpecialToolRuntimeExample(strings.TrimSpace(def.Name)); ok {
		return example
	}

	return graphQLToolRuntimeExample{
		Document: graphQLExampleDocument(graphQLToolOperation(def), strings.TrimSpace(def.Name), def.Parameters),
	}
}

func graphQLSpecialToolRuntimeExample(name string) (graphQLToolRuntimeExample, bool) {
	switch name {
	case AskHumanToolName:
		return graphQLToolRuntimeExample{
			Note:     "include a final custom option so the user can type their own answer.",
			Document: `mutation { ask_human(prompt: "Which environment should I use?", options: [{label: "staging"}, {label: "Other", allow_custom: true}]) }`,
		}, true
	case ToolSearchToolName:
		return graphQLToolRuntimeExample{
			Note:     "after `action: load`, the loaded tool is available next turn, not in the same response.",
			Document: `mutation { tfind(action: load, tool_names: ["browser_control"]) }`,
		}, true
	case "script_exec":
		return graphQLToolRuntimeExample{
			Note:     "minimal sandbox execution mutation.",
			Document: `mutation { script_exec(script: "print(\"ok\")") }`,
		}, true
	default:
		return graphQLToolRuntimeExample{}, false
	}
}

func graphQLExampleDocument(
	operation ast.Operation,
	fieldName string,
	params json.RawMessage,
) string {
	signature := graphQLExampleFieldCall(fieldName, params)
	if operation == ast.Query {
		return "query { " + signature + " }"
	}
	return "mutation { " + signature + " }"
}

func graphQLExampleFieldCall(fieldName string, params json.RawMessage) string {
	root := decodeGraphQLToolSchemaObject(params)
	properties, _ := root["properties"].(map[string]any)
	if len(properties) == 0 {
		return fieldName
	}

	keys := graphQLRequiredExampleKeys(properties, graphQLRequiredSet(root["required"]))
	if len(keys) == 0 {
		return fieldName
	}
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		schema, _ := properties[key].(map[string]any)
		parts = append(parts, key+": "+graphQLExampleLiteral(key, schema))
	}
	return fieldName + "(" + strings.Join(parts, ", ") + ")"
}

func graphQLRequiredExampleKeys(properties map[string]any, required map[string]bool) []string {
	keys := make([]string, 0, len(required))
	for key := range required {
		if _, ok := properties[key]; ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func graphQLExampleLiteral(name string, schema map[string]any) string {
	if enumValue := graphQLEnumExample(schema); enumValue != "" {
		return enumValue
	}
	switch graphQLSchemaTypeName(schema) {
	case "string":
		return graphQLQuotedString(graphQLExampleString(name, schema))
	case "integer":
		return "1"
	case "number":
		return "1.0"
	case "boolean":
		return "true"
	case "array":
		return "[" + graphQLExampleLiteral(name, graphQLSchemaChild(schema, "items")) + "]"
	case "object":
		return graphQLExampleObjectLiteral(schema)
	default:
		return "{}"
	}
}

func graphQLExampleObjectLiteral(schema map[string]any) string {
	properties, _ := schema["properties"].(map[string]any)
	if len(properties) == 0 {
		return "{}"
	}

	required := graphQLRequiredSet(schema["required"])
	keys := graphQLExampleObjectKeys(properties, required)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		child, _ := properties[key].(map[string]any)
		parts = append(parts, key+": "+graphQLExampleLiteral(key, child))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func graphQLExampleObjectKeys(properties map[string]any, required map[string]bool) []string {
	requiredKeys := make([]string, 0, len(required))
	for key := range required {
		if _, ok := properties[key]; ok {
			requiredKeys = append(requiredKeys, key)
		}
	}
	sort.Strings(requiredKeys)
	if len(requiredKeys) > 0 {
		return requiredKeys
	}

	keys := make([]string, 0, len(properties))
	for key := range properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys[:1]
}

func graphQLExampleString(name string, schema map[string]any) string {
	trimmed := strings.ToLower(strings.TrimSpace(name))
	switch {
	case strings.Contains(trimmed, "prompt"):
		return "What should I do next?"
	case strings.Contains(trimmed, "query"):
		return "OpenAI API docs"
	case strings.Contains(trimmed, "script"):
		return "print(\"ok\")"
	case strings.Contains(trimmed, "path"), strings.Contains(trimmed, "dir"):
		return "."
	case strings.Contains(trimmed, "url"):
		return "https://example.com"
	case strings.Contains(trimmed, "text"), strings.Contains(trimmed, "message"):
		return "hello"
	case strings.Contains(trimmed, "name"), strings.Contains(trimmed, "id"):
		return "example"
	default:
		return "example"
	}
}

func graphQLEnumExample(schema map[string]any) string {
	values, _ := schema["enum"].([]any)
	for _, value := range values {
		text, _ := value.(string)
		if strings.TrimSpace(text) != "" {
			return text
		}
	}
	return ""
}

func graphQLSchemaChild(schema map[string]any, key string) map[string]any {
	child, _ := schema[key].(map[string]any)
	return child
}

func graphQLQuotedString(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(encoded)
}
