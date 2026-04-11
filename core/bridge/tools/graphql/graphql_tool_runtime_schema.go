package graphql

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"ghost-os/bridge/llm"
)

type GraphQLToolRuntimeSchema struct {
	EnumTypes      []GraphQLToolRuntimeEnumType
	InputTypes     []GraphQLToolRuntimeInputType
	MutationFields []GraphQLToolRuntimeField
}

type GraphQLToolRuntimeField struct {
	Name        string
	Description string
	Arguments   []GraphQLToolRuntimeArgument
}

type GraphQLToolRuntimeArgument struct {
	Name        string
	Type        string
	Description string
}

type GraphQLToolRuntimeInputType struct {
	Name        string
	Description string
	Fields      []GraphQLToolRuntimeArgument
}

type GraphQLToolRuntimeEnumType struct {
	Name        string
	Description string
	Values      []GraphQLToolRuntimeEnumValue
}

type GraphQLToolRuntimeEnumValue struct {
	Name string
}

type GraphQLToolIDEntry struct {
	ID  int
	Def llm.ToolDef
}

func BuildGraphQLToolRuntimeSchema(catalog ToolCatalog) GraphQLToolRuntimeSchema {
	defs := visibleGraphQLToolDefs(catalog)
	builder := newGraphQLToolRuntimeTypeBuilder()
	schema := GraphQLToolRuntimeSchema{}

	for _, def := range defs {
		field := buildGraphQLToolRuntimeField(def, builder)
		schema.MutationFields = append(schema.MutationFields, field)
	}

	schema.EnumTypes = builder.enumTypesInOrder()
	schema.InputTypes = builder.inputTypesInOrder()
	return schema
}

func graphQLVisibleToolIDEntries(catalog ToolCatalog) []GraphQLToolIDEntry {
	defs := visibleGraphQLToolDefs(catalog)
	if len(defs) == 0 {
		return nil
	}
	entries := make([]GraphQLToolIDEntry, 0, len(defs))
	for index, def := range defs {
		entries = append(entries, GraphQLToolIDEntry{
			ID:  index + 1,
			Def: def,
		})
	}
	return entries
}

func FormatGraphQLToolRuntimePrompt(catalog ToolCatalog) string {
	entries := graphQLVisibleToolIDEntries(catalog)
	var builder strings.Builder

	builder.WriteString("[System Instruction]\n")
	builder.WriteString("You have the following tools. When calling a tool, strictly use <t:TOOL_ID>JSON_ARGS</t>.\n")
	builder.WriteString("The JSON args must be a single object. Prefer flat top-level arguments; use nested objects only when the parameter schema explicitly requires them.\n")
	builder.WriteString("You may place multiple tool calls in one reply by concatenating tags, for example: <t:1>{\"city\":\"Beijing\"}</t><t:2>{\"to\":\"boss@example.com\"}</t>.\n\n")

	if len(entries) == 0 {
		builder.WriteString("No tools are available for this turn.")
		return strings.TrimSpace(builder.String())
	}

	for _, entry := range entries {
		builder.WriteString("ID: ")
		builder.WriteString(strconv.Itoa(entry.ID))
		builder.WriteString("\n")
		builder.WriteString("Tool name: ")
		builder.WriteString(strings.TrimSpace(entry.Def.Name))
		if desc := strings.TrimSpace(entry.Def.Description); desc != "" {
			builder.WriteString(" (")
			builder.WriteString(desc)
			builder.WriteString(")")
		}
		builder.WriteString("\n")
		builder.WriteString("Parameter format: ")
		builder.WriteString(formatToolParameterShape(entry.Def.Parameters))
		builder.WriteString("\n\n")
	}

	return strings.TrimSpace(builder.String())
}

func formatToolParameterShape(params json.RawMessage) string {
	root := decodeGraphQLToolSchemaObject(params)
	properties, _ := root["properties"].(map[string]any)
	if len(properties) == 0 {
		return `{}`
	}
	required := graphQLRequiredSet(root["required"])
	keys := graphQLSortedPropertyKeys(properties)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		schema := graphQLSchemaMap(properties[key])
		parts = append(parts, formatToolParameterShapeEntry(key, schema, required[key]))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func formatToolParameterShapeEntry(name string, schema map[string]any, required bool) string {
	typeLabel := formatToolParameterTypeLabel(schema)
	metaParts := make([]string, 0, 2)
	if required {
		metaParts = append(metaParts, "required")
	} else {
		metaParts = append(metaParts, "optional")
	}
	if example := toolParameterExampleForName(name, schema); example != "" {
		metaParts = append(metaParts, fmt.Sprintf("e.g. %s", example))
	}
	if desc := strings.TrimSpace(graphQLArgumentDescription(schema)); desc != "" {
		metaParts = append(metaParts, desc)
	}
	return fmt.Sprintf("\"%s\": %s (%s)", name, typeLabel, strings.Join(metaParts, ", "))
}

func visibleGraphQLToolDefs(catalog ToolCatalog) []llm.ToolDef {
	if catalog == nil {
		return nil
	}
	defs := catalog.ToolDefs()
	if source, ok := catalog.(GraphQLToolDefSource); ok {
		defs = source.GraphQLToolDefs()
	}
	out := make([]llm.ToolDef, 0, len(defs))
	for _, def := range defs {
		if strings.TrimSpace(def.Name) == "" {
			continue
		}
		out = append(out, def)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func buildGraphQLToolRuntimeField(
	def llm.ToolDef,
	builder *graphQLToolRuntimeTypeBuilder,
) GraphQLToolRuntimeField {
	return GraphQLToolRuntimeField{
		Name:        strings.TrimSpace(def.Name),
		Description: strings.TrimSpace(def.Description),
		Arguments:   buildGraphQLToolRuntimeArguments(def.Parameters, def.Name, builder),
	}
}

func buildGraphQLToolRuntimeArguments(
	params json.RawMessage,
	toolName string,
	builder *graphQLToolRuntimeTypeBuilder,
) []GraphQLToolRuntimeArgument {
	root := decodeGraphQLToolSchemaObject(params)
	properties, _ := root["properties"].(map[string]any)
	if len(properties) == 0 {
		return nil
	}
	required := graphQLRequiredSet(root["required"])
	keys := graphQLSortedPropertyKeys(properties)
	prefix := graphQLToolTypePrefix(toolName)
	out := make([]GraphQLToolRuntimeArgument, 0, len(keys))
	for _, key := range keys {
		property := graphQLSchemaMap(properties[key])
		out = append(out, GraphQLToolRuntimeArgument{
			Name:        key,
			Type:        builder.argumentType(graphQLTypePath(prefix, key), property, required[key]),
			Description: graphQLArgumentDescription(property),
		})
	}
	return out
}

func decodeGraphQLToolSchemaObject(params json.RawMessage) map[string]any {
	if len(params) == 0 {
		return nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(params, &decoded); err != nil {
		return nil
	}
	return decoded
}

func graphQLRequiredSet(raw any) map[string]bool {
	items, _ := raw.([]any)
	out := make(map[string]bool, len(items))
	for _, item := range items {
		name, _ := item.(string)
		name = strings.TrimSpace(name)
		if name != "" {
			out[name] = true
		}
	}
	return out
}

func graphQLArgumentDescription(schema map[string]any) string {
	if len(schema) == 0 {
		return ""
	}
	description, _ := schema["description"].(string)
	return strings.TrimSpace(description)
}

func graphQLSortedPropertyKeys(properties map[string]any) []string {
	keys := make([]string, 0, len(properties))
	for key := range properties {
		if strings.TrimSpace(key) != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}
