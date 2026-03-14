package graphqlschema

import "strings"

func (s *Schema) RootQueryList() []Field {
	if s == nil {
		return nil
	}
	return cloneFields(s.RootQueries)
}

func (s *Schema) TypeByName(name string) (Type, bool) {
	if s == nil {
		return Type{}, false
	}
	item, ok := s.typeIndex[strings.TrimSpace(name)]
	if !ok {
		return Type{}, false
	}
	return cloneType(item), true
}

func (s *Schema) FieldMatches(name string) []FieldLocation {
	if s == nil {
		return nil
	}
	matches := s.fieldIndex[strings.TrimSpace(name)]
	return cloneFieldLocations(matches)
}

func cloneTypes(types []Type) []Type {
	out := make([]Type, 0, len(types))
	for _, item := range types {
		out = append(out, cloneType(item))
	}
	return out
}

func cloneType(item Type) Type {
	return Type{
		Name:        item.Name,
		Description: item.Description,
		Fields:      cloneFields(item.Fields),
	}
}

func cloneFields(fields []Field) []Field {
	out := make([]Field, 0, len(fields))
	for _, field := range fields {
		out = append(out, cloneField(field))
	}
	return out
}

func cloneField(field Field) Field {
	return Field{
		Name:        field.Name,
		Description: field.Description,
		Args:        cloneArguments(field.Args),
		ReturnType:  field.ReturnType,
		Deprecated:  field.Deprecated,
	}
}

func cloneArguments(args []Argument) []Argument {
	out := make([]Argument, 0, len(args))
	for _, arg := range args {
		out = append(out, Argument{
			Name:        arg.Name,
			Type:        arg.Type,
			Description: arg.Description,
		})
	}
	return out
}

func cloneFieldLocations(matches []FieldLocation) []FieldLocation {
	out := make([]FieldLocation, 0, len(matches))
	for _, item := range matches {
		out = append(out, FieldLocation{
			TypeName: item.TypeName,
			Field:    cloneField(item.Field),
		})
	}
	return out
}
