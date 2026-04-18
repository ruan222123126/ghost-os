package graphql

import "strings"

type graphQLToolRuntimeTypeBuilder struct {
	enumTypes  map[string]GraphQLToolRuntimeEnumType
	enumOrder  []string
	inputTypes map[string]GraphQLToolRuntimeInputType
	inputOrder []string
}

func newGraphQLToolRuntimeTypeBuilder() *graphQLToolRuntimeTypeBuilder {
	return &graphQLToolRuntimeTypeBuilder{
		enumTypes:  make(map[string]GraphQLToolRuntimeEnumType),
		inputTypes: make(map[string]GraphQLToolRuntimeInputType),
	}
}

func (b *graphQLToolRuntimeTypeBuilder) enumTypesInOrder() []GraphQLToolRuntimeEnumType {
	out := make([]GraphQLToolRuntimeEnumType, 0, len(b.enumOrder))
	for _, name := range b.enumOrder {
		out = append(out, b.enumTypes[name])
	}
	return out
}

func (b *graphQLToolRuntimeTypeBuilder) inputTypesInOrder() []GraphQLToolRuntimeInputType {
	out := make([]GraphQLToolRuntimeInputType, 0, len(b.inputOrder))
	for _, name := range b.inputOrder {
		out = append(out, b.inputTypes[name])
	}
	return out
}

func (b *graphQLToolRuntimeTypeBuilder) argumentType(
	path []string,
	schema map[string]any,
	required bool,
) string {
	baseType := b.baseType(path, schema)
	if required {
		return baseType + "!"
	}
	return baseType
}

func (b *graphQLToolRuntimeTypeBuilder) baseType(path []string, schema map[string]any) string {
	if enumValues, ok := graphQLEnumValues(schema); ok {
		return b.registerEnumType(path, graphQLArgumentDescription(schema), enumValues)
	}
	switch graphQLSchemaTypeName(schema) {
	case "string":
		return "String"
	case "integer":
		return "Int"
	case "number":
		return "Float"
	case "boolean":
		return "Boolean"
	case "object":
		return b.registerInputType(path, schema)
	case "array":
		return b.listType(path, schema)
	default:
		return "JSON"
	}
}

func (b *graphQLToolRuntimeTypeBuilder) listType(path []string, schema map[string]any) string {
	items := graphQLSchemaMap(schema["items"])
	if len(items) == 0 {
		return "[JSON]"
	}
	itemType := b.baseType(graphQLTypePath(path, "item"), items)
	if itemType != "JSON" && !graphQLSchemaAllowsNull(items) {
		itemType += "!"
	}
	return "[" + itemType + "]"
}

func (b *graphQLToolRuntimeTypeBuilder) registerInputType(
	path []string,
	schema map[string]any,
) string {
	properties, _ := schema["properties"].(map[string]any)
	if len(properties) == 0 {
		return "JSON"
	}
	name := graphQLTypeName(path, "Input")
	if _, exists := b.inputTypes[name]; exists {
		return name
	}
	b.inputTypes[name] = GraphQLToolRuntimeInputType{
		Name:        name,
		Description: graphQLArgumentDescription(schema),
	}
	b.inputOrder = append(b.inputOrder, name)

	required := graphQLRequiredSet(schema["required"])
	keys := graphQLSortedPropertyKeys(properties)
	fields := make([]GraphQLToolRuntimeArgument, 0, len(keys))
	for _, key := range keys {
		property := graphQLSchemaMap(properties[key])
		fields = append(fields, GraphQLToolRuntimeArgument{
			Name:        key,
			Type:        b.argumentType(graphQLTypePath(path, key), property, required[key]),
			Description: graphQLArgumentDescription(property),
		})
	}
	b.inputTypes[name] = GraphQLToolRuntimeInputType{
		Name:        name,
		Description: graphQLArgumentDescription(schema),
		Fields:      fields,
	}
	return name
}

func (b *graphQLToolRuntimeTypeBuilder) registerEnumType(
	path []string,
	description string,
	values []GraphQLToolRuntimeEnumValue,
) string {
	name := graphQLTypeName(path, "Enum")
	if _, exists := b.enumTypes[name]; exists {
		return name
	}
	b.enumTypes[name] = GraphQLToolRuntimeEnumType{
		Name:        name,
		Description: description,
		Values:      values,
	}
	b.enumOrder = append(b.enumOrder, name)
	return name
}

func graphQLSchemaTypeName(schema map[string]any) string {
	if len(schema) == 0 {
		return ""
	}
	rawType, ok := schema["type"]
	if ok {
		switch value := rawType.(type) {
		case string:
			return strings.TrimSpace(value)
		case []any:
			for _, item := range value {
				name, _ := item.(string)
				name = strings.TrimSpace(name)
				if name != "" && name != "null" {
					return name
				}
			}
		}
	}
	if _, ok := schema["properties"]; ok {
		return "object"
	}
	if _, ok := schema["items"]; ok {
		return "array"
	}
	return ""
}

func graphQLSchemaAllowsNull(schema map[string]any) bool {
	rawType, ok := schema["type"]
	if !ok {
		return false
	}
	switch value := rawType.(type) {
	case string:
		return strings.TrimSpace(value) == "null"
	case []any:
		for _, item := range value {
			name, _ := item.(string)
			if strings.TrimSpace(name) == "null" {
				return true
			}
		}
	}
	return false
}

func graphQLSchemaMap(raw any) map[string]any {
	schema, _ := raw.(map[string]any)
	return schema
}

func graphQLEnumValues(schema map[string]any) ([]GraphQLToolRuntimeEnumValue, bool) {
	rawValues, _ := schema["enum"].([]any)
	if len(rawValues) == 0 {
		return nil, false
	}
	out := make([]GraphQLToolRuntimeEnumValue, 0, len(rawValues))
	seen := make(map[string]bool, len(rawValues))
	for _, item := range rawValues {
		value, _ := item.(string)
		value = strings.TrimSpace(value)
		if !graphQLIsEnumValue(value) {
			return nil, false
		}
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, GraphQLToolRuntimeEnumValue{Name: value})
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

func graphQLIsEnumValue(value string) bool {
	if value == "" || value == "true" || value == "false" || value == "null" {
		return false
	}
	for index, ch := range value {
		if ch == '_' || ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z' {
			continue
		}
		if index > 0 && ch >= '0' && ch <= '9' {
			continue
		}
		return false
	}
	return true
}

func graphQLToolTypePrefix(toolName string) []string {
	return []string{graphQLPascalCase(toolName)}
}

func graphQLTypePath(path []string, segment string) []string {
	next := append([]string{}, path...)
	return append(next, graphQLPascalCase(segment))
}

func graphQLTypeName(path []string, suffix string) string {
	return strings.Join(path, "") + suffix
}

func graphQLPascalCase(raw string) string {
	parts := strings.FieldsFunc(strings.TrimSpace(raw), func(ch rune) bool {
		if ch >= '0' && ch <= '9' {
			return false
		}
		if ch >= 'A' && ch <= 'Z' {
			return false
		}
		if ch >= 'a' && ch <= 'z' {
			return false
		}
		return true
	})
	if len(parts) == 0 {
		return "Value"
	}
	var builder strings.Builder
	for _, part := range parts {
		lower := strings.ToLower(part)
		builder.WriteString(strings.ToUpper(lower[:1]))
		builder.WriteString(lower[1:])
	}
	name := builder.String()
	if name[0] >= '0' && name[0] <= '9' {
		return "N" + name
	}
	return name
}
