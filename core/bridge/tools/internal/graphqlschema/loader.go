package graphqlschema

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

type Schema struct {
	RootQueries []Field `json:"root_queries"`
	Types       []Type  `json:"types"`

	rootIndex  map[string]Field
	typeIndex  map[string]Type
	fieldIndex map[string][]FieldLocation
}

type Type struct {
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	Fields      []Field `json:"fields"`
}

type Field struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Args        []Argument `json:"args,omitempty"`
	ReturnType  string     `json:"return_type"`
	Deprecated  bool       `json:"deprecated,omitempty"`
}

type Argument struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

type FieldLocation struct {
	TypeName string `json:"type_name"`
	Field    Field  `json:"field"`
}

func Load(path string) (*Schema, error) {
	raw, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return nil, fmt.Errorf("read graphql schema snapshot: %w", err)
	}
	schema, err := parse(raw)
	if err != nil {
		return nil, err
	}
	return buildIndex(schema), nil
}

func parse(raw []byte) (Schema, error) {
	var schema Schema
	if err := json.Unmarshal(raw, &schema); err != nil {
		return Schema{}, fmt.Errorf("decode graphql schema snapshot: %w", err)
	}
	if err := validateRootQueries(schema.RootQueries); err != nil {
		return Schema{}, err
	}
	if err := validateTypes(schema.Types); err != nil {
		return Schema{}, err
	}
	return normalizeSchema(schema), nil
}

func validateRootQueries(rootQueries []Field) error {
	return validateFields("root_queries", rootQueries)
}

func validateTypes(types []Type) error {
	seen := make(map[string]bool, len(types))
	for _, item := range types {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			return fmt.Errorf("graphql schema snapshot type name is required")
		}
		if seen[name] {
			return fmt.Errorf("graphql schema snapshot contains duplicate type %q", name)
		}
		seen[name] = true
		if err := validateFields("type "+name, item.Fields); err != nil {
			return err
		}
	}
	return nil
}

func validateFields(scope string, fields []Field) error {
	seen := make(map[string]bool, len(fields))
	for _, field := range fields {
		name := strings.TrimSpace(field.Name)
		if name == "" {
			return fmt.Errorf("%s field name is required", scope)
		}
		if seen[name] {
			return fmt.Errorf("%s contains duplicate field %q", scope, name)
		}
		seen[name] = true
		if strings.TrimSpace(field.ReturnType) == "" {
			return fmt.Errorf("%s field %q return_type is required", scope, name)
		}
		if err := validateArguments(scope+" field "+name, field.Args); err != nil {
			return err
		}
	}
	return nil
}

func validateArguments(scope string, args []Argument) error {
	seen := make(map[string]bool, len(args))
	for _, arg := range args {
		name := strings.TrimSpace(arg.Name)
		if name == "" {
			return fmt.Errorf("%s argument name is required", scope)
		}
		if seen[name] {
			return fmt.Errorf("%s contains duplicate argument %q", scope, name)
		}
		if strings.TrimSpace(arg.Type) == "" {
			return fmt.Errorf("%s argument %q type is required", scope, name)
		}
		seen[name] = true
	}
	return nil
}

func normalizeSchema(schema Schema) Schema {
	out := schema
	out.RootQueries = normalizeFields(schema.RootQueries)
	out.Types = make([]Type, 0, len(schema.Types))
	for _, item := range schema.Types {
		out.Types = append(out.Types, Type{
			Name:        strings.TrimSpace(item.Name),
			Description: strings.TrimSpace(item.Description),
			Fields:      normalizeFields(item.Fields),
		})
	}
	return out
}

func normalizeFields(fields []Field) []Field {
	out := make([]Field, 0, len(fields))
	for _, field := range fields {
		out = append(out, Field{
			Name:        strings.TrimSpace(field.Name),
			Description: strings.TrimSpace(field.Description),
			Args:        normalizeArguments(field.Args),
			ReturnType:  strings.TrimSpace(field.ReturnType),
			Deprecated:  field.Deprecated,
		})
	}
	return out
}

func normalizeArguments(args []Argument) []Argument {
	out := make([]Argument, 0, len(args))
	for _, arg := range args {
		out = append(out, Argument{
			Name:        strings.TrimSpace(arg.Name),
			Type:        strings.TrimSpace(arg.Type),
			Description: strings.TrimSpace(arg.Description),
		})
	}
	return out
}

func buildIndex(schema Schema) *Schema {
	rootIndex := make(map[string]Field, len(schema.RootQueries))
	typeIndex := make(map[string]Type, len(schema.Types))
	fieldIndex := make(map[string][]FieldLocation)
	for _, field := range schema.RootQueries {
		rootIndex[field.Name] = cloneField(field)
	}
	for _, item := range schema.Types {
		typeIndex[item.Name] = cloneType(item)
		for _, field := range item.Fields {
			name := field.Name
			fieldIndex[name] = append(fieldIndex[name], FieldLocation{
				TypeName: item.Name,
				Field:    cloneField(field),
			})
		}
	}
	for name := range fieldIndex {
		sort.Slice(fieldIndex[name], func(i, j int) bool {
			return fieldIndex[name][i].TypeName < fieldIndex[name][j].TypeName
		})
	}
	return &Schema{
		RootQueries: cloneFields(schema.RootQueries),
		Types:       cloneTypes(schema.Types),
		rootIndex:   rootIndex,
		typeIndex:   typeIndex,
		fieldIndex:  fieldIndex,
	}
}
