package tools

import "strings"

func writeGraphQLToolTypeDefinitions(
	builder *strings.Builder,
	enumTypes []GraphQLToolRuntimeEnumType,
	inputTypes []GraphQLToolRuntimeInputType,
) {
	wroteBlock := false
	for _, enumType := range enumTypes {
		if wroteBlock {
			builder.WriteString("\n\n")
		}
		writeGraphQLEnumTypeBlock(builder, enumType)
		wroteBlock = true
	}
	for _, inputType := range inputTypes {
		if wroteBlock {
			builder.WriteString("\n\n")
		}
		writeGraphQLInputTypeBlock(builder, inputType)
		wroteBlock = true
	}
	if wroteBlock {
		builder.WriteString("\n\n")
	}
}

func writeGraphQLEnumTypeBlock(builder *strings.Builder, enumType GraphQLToolRuntimeEnumType) {
	if enumType.Description != "" {
		builder.WriteString("# ")
		builder.WriteString(enumType.Description)
		builder.WriteString("\n")
	}
	builder.WriteString("enum ")
	builder.WriteString(enumType.Name)
	builder.WriteString(" {\n")
	for _, value := range enumType.Values {
		builder.WriteString("  ")
		builder.WriteString(value.Name)
		builder.WriteString("\n")
	}
	builder.WriteString("}")
}

func writeGraphQLInputTypeBlock(builder *strings.Builder, inputType GraphQLToolRuntimeInputType) {
	if inputType.Description != "" {
		builder.WriteString("# ")
		builder.WriteString(inputType.Description)
		builder.WriteString("\n")
	}
	builder.WriteString("input ")
	builder.WriteString(inputType.Name)
	builder.WriteString(" {\n")
	for _, field := range inputType.Fields {
		builder.WriteString("  ")
		builder.WriteString(field.Name)
		builder.WriteString(": ")
		builder.WriteString(field.Type)
		if field.Description != "" {
			builder.WriteString(" # ")
			builder.WriteString(field.Description)
		}
		builder.WriteString("\n")
	}
	builder.WriteString("}")
}

func writeGraphQLToolFieldBlock(
	builder *strings.Builder,
	typeName string,
	fields []GraphQLToolRuntimeField,
) {
	builder.WriteString("type ")
	builder.WriteString(typeName)
	builder.WriteString(" {\n")
	if len(fields) == 0 {
		builder.WriteString("  _empty: JSON\n")
		builder.WriteString("}")
		return
	}
	for _, field := range fields {
		builder.WriteString("  ")
		builder.WriteString(graphQLToolFieldSignature(field))
		builder.WriteString(": JSON")
		if field.Description != "" {
			builder.WriteString(" # ")
			builder.WriteString(field.Description)
		}
		builder.WriteString("\n")
	}
	builder.WriteString("}")
}

func graphQLToolFieldSignature(field GraphQLToolRuntimeField) string {
	if len(field.Arguments) == 0 {
		return field.Name
	}
	parts := make([]string, 0, len(field.Arguments))
	for _, argument := range field.Arguments {
		parts = append(parts, argument.Name+": "+argument.Type)
	}
	return field.Name + "(" + strings.Join(parts, ", ") + ")"
}
