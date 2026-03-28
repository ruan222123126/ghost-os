package tools

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"

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

func FormatGraphQLToolRuntimePrompt(catalog ToolCatalog) string {
	defs := visibleGraphQLToolDefs(catalog)
	schema := BuildGraphQLToolRuntimeSchema(catalog)
	var builder strings.Builder

	builder.WriteString("GraphQL Tool Call Protocol:\n")
	builder.WriteString("- Return exactly one GraphQL document and nothing else.\n")
	builder.WriteString("- Do not emit native tool_calls.\n")
	builder.WriteString("- Use one or more `mutation` operations.\n")
	builder.WriteString("- Each operation must contain exactly one top-level field.\n")
	builder.WriteString("- Operations execute sequentially in the order written.\n")
	builder.WriteString("- Field names must match visible tool names exactly.\n")
	builder.WriteString("- GraphQL arguments map directly to the tool JSON parameters.\n")
	builder.WriteString("- If one operation loads a dynamic tool (for example `tfind(action: load)`), later operations in the same document can call it.\n")
	builder.WriteString("- Unsupported: aliases, fragments, variables, directives, multiple top-level fields in one operation.\n")
	builder.WriteString("- `query` is not part of this protocol; use `mutation` for every tool call.\n")
	if visibleNames := visibleGraphQLToolDefNames(catalog); len(visibleNames) == 1 && visibleNames[0] == ToolSearchToolName {
		builder.WriteString("- This turn is effectively empty; bootstrap by starting with `mutation { tfind(action: search, query: \"...\") }`.\n")
	}
	if hasVisibleGraphQLToolDef(catalog, ToolSearchToolName) {
		builder.WriteString("- If you are unsure which tools are visible, prefer `mutation { tfind(action: search, query: \"...\") }`.\n")
		builder.WriteString("- Use `mutation { tfind(action: list) }` only when you need the current dynamic tool load state.\n")
	}
	builder.WriteString("- Never repeat or fabricate `[GRAPHQL_TOOL_RESULT]`; that marker is internal bridge feedback.\n")
	builder.WriteString("- Prefer copying the closest minimal successful example and editing only the arguments you need.\n\n")
	builder.WriteString("Available GraphQL tool schema:\n")
	builder.WriteString("scalar JSON\n\n")
	writeGraphQLToolTypeDefinitions(&builder, schema.EnumTypes, schema.InputTypes)
	writeGraphQLToolFieldBlock(&builder, "Mutation", schema.MutationFields)
	writeGraphQLToolExamplesBlock(&builder, defs)
	return strings.TrimSpace(builder.String())
}

func graphQLToolOperation(_ llm.ToolDef) ast.Operation {
	return ast.Mutation
}

func graphQLVisibleToolDef(catalog ToolCatalog, name string) (llm.ToolDef, bool) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return llm.ToolDef{}, false
	}
	for _, def := range visibleGraphQLToolDefs(catalog) {
		if strings.TrimSpace(def.Name) == trimmed {
			return def, true
		}
	}
	return llm.ToolDef{}, false
}

func hasVisibleGraphQLToolDef(catalog ToolCatalog, name string) bool {
	_, ok := graphQLVisibleToolDef(catalog, name)
	return ok
}

func visibleGraphQLToolDefNames(catalog ToolCatalog) []string {
	defs := visibleGraphQLToolDefs(catalog)
	if len(defs) == 0 {
		return nil
	}
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		names = append(names, strings.TrimSpace(def.Name))
	}
	return names
}

func visibleGraphQLToolDefs(catalog ToolCatalog) []llm.ToolDef {
	if catalog == nil {
		return nil
	}
	defs := catalog.ToolDefs()
	if source, ok := catalog.(graphQLToolDefSource); ok {
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
