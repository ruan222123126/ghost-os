package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/tools/internal/graphqlschema"
	"ghost-os/bridge/tools/internal/tooljson"
)

const (
	graphqlSchemaLookupActionListRootQueries = "list_root_queries"
	graphqlSchemaLookupActionDescribeType    = "describe_type"
	graphqlSchemaLookupActionFindField       = "find_field"
)

type GraphQLSchemaLookupTool struct {
	schema *graphqlschema.Schema
}

type graphqlSchemaLookupArgs struct {
	Action string `json:"action"`
	Name   string `json:"name,omitempty"`
}

type graphqlSchemaFieldPayload struct {
	Name        string                         `json:"name"`
	Signature   string                         `json:"signature"`
	Description string                         `json:"description,omitempty"`
	ReturnType  string                         `json:"return_type"`
	Args        []graphqlSchemaArgumentPayload `json:"args,omitempty"`
	Deprecated  bool                           `json:"deprecated,omitempty"`
}

type graphqlSchemaArgumentPayload struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

type graphqlSchemaTypePayload struct {
	Name        string                      `json:"name"`
	Description string                      `json:"description,omitempty"`
	Fields      []graphqlSchemaFieldPayload `json:"fields"`
}

type graphqlSchemaFieldMatchPayload struct {
	TypeName string                    `json:"type_name"`
	Field    graphqlSchemaFieldPayload `json:"field"`
}

func NewGraphQLSchemaLookupTool(schema *graphqlschema.Schema) Tool {
	return &GraphQLSchemaLookupTool{schema: schema}
}

func NewGraphQLSchemaLookupToolFromPath(path string) (Tool, error) {
	schema, err := graphqlschema.Load(path)
	if err != nil {
		return nil, err
	}
	return NewGraphQLSchemaLookupTool(schema), nil
}

func (GraphQLSchemaLookupTool) Name() string {
	return "graphql_schema_lookup"
}

func (GraphQLSchemaLookupTool) Description() string {
	return "Inspect the local GraphQL schema snapshot before writing a query. This tool does not access the network."
}

func (GraphQLSchemaLookupTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"action":{"type":"string","enum":["list_root_queries","describe_type","find_field"]},
			"name":{"type":"string","description":"Query, type, or field name for the selected action."}
		},
		"required":["action"],
		"additionalProperties":false
	}`)
}

func (t *GraphQLSchemaLookupTool) Execute(_ context.Context, argsJSON json.RawMessage, _ string) (string, error) {
	if t == nil || t.schema == nil {
		return "", fmt.Errorf("graphql schema snapshot is not configured")
	}

	var args graphqlSchemaLookupArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}

	switch strings.ToLower(strings.TrimSpace(args.Action)) {
	case graphqlSchemaLookupActionListRootQueries:
		return t.listRootQueries()
	case graphqlSchemaLookupActionDescribeType:
		return t.describeType(args.Name)
	case graphqlSchemaLookupActionFindField:
		return t.findField(args.Name)
	default:
		return "", fmt.Errorf("action must be one of: list_root_queries, describe_type, find_field")
	}
}

func (t *GraphQLSchemaLookupTool) listRootQueries() (string, error) {
	rootQueries := t.schema.RootQueryList()
	items := make([]graphqlSchemaFieldPayload, 0, len(rootQueries))
	for _, field := range rootQueries {
		items = append(items, newGraphQLSchemaFieldPayload(field))
	}
	return tooljson.Encode(struct {
		Action      string                      `json:"action"`
		RootQueries []graphqlSchemaFieldPayload `json:"root_queries"`
	}{
		Action:      graphqlSchemaLookupActionListRootQueries,
		RootQueries: items,
	})
}

func (t *GraphQLSchemaLookupTool) describeType(name string) (string, error) {
	typeName := strings.TrimSpace(name)
	if typeName == "" {
		return "", fmt.Errorf("name is required for describe_type")
	}
	item, ok := t.schema.TypeByName(typeName)
	if !ok {
		return "", fmt.Errorf("type %q was not found in the graphql schema snapshot", typeName)
	}
	return tooljson.Encode(struct {
		Action string                   `json:"action"`
		Type   graphqlSchemaTypePayload `json:"type"`
	}{
		Action: graphqlSchemaLookupActionDescribeType,
		Type:   newGraphQLSchemaTypePayload(item),
	})
}

func (t *GraphQLSchemaLookupTool) findField(name string) (string, error) {
	fieldName := strings.TrimSpace(name)
	if fieldName == "" {
		return "", fmt.Errorf("name is required for find_field")
	}
	matches := t.schema.FieldMatches(fieldName)
	items := make([]graphqlSchemaFieldMatchPayload, 0, len(matches))
	for _, match := range matches {
		items = append(items, graphqlSchemaFieldMatchPayload{
			TypeName: match.TypeName,
			Field:    newGraphQLSchemaFieldPayload(match.Field),
		})
	}
	return tooljson.Encode(struct {
		Action  string                           `json:"action"`
		Name    string                           `json:"name"`
		Matches []graphqlSchemaFieldMatchPayload `json:"matches"`
	}{
		Action:  graphqlSchemaLookupActionFindField,
		Name:    fieldName,
		Matches: items,
	})
}

func newGraphQLSchemaTypePayload(item graphqlschema.Type) graphqlSchemaTypePayload {
	fields := make([]graphqlSchemaFieldPayload, 0, len(item.Fields))
	for _, field := range item.Fields {
		fields = append(fields, newGraphQLSchemaFieldPayload(field))
	}
	return graphqlSchemaTypePayload{
		Name:        item.Name,
		Description: item.Description,
		Fields:      fields,
	}
}

func newGraphQLSchemaFieldPayload(field graphqlschema.Field) graphqlSchemaFieldPayload {
	args := make([]graphqlSchemaArgumentPayload, 0, len(field.Args))
	for _, arg := range field.Args {
		args = append(args, graphqlSchemaArgumentPayload{
			Name:        arg.Name,
			Type:        arg.Type,
			Description: arg.Description,
		})
	}
	return graphqlSchemaFieldPayload{
		Name:        field.Name,
		Signature:   graphQLFieldSignature(field),
		Description: field.Description,
		ReturnType:  field.ReturnType,
		Args:        args,
		Deprecated:  field.Deprecated,
	}
}

func graphQLFieldSignature(field graphqlschema.Field) string {
	args := make([]string, 0, len(field.Args))
	for _, arg := range field.Args {
		args = append(args, arg.Name+": "+arg.Type)
	}
	if len(args) == 0 {
		return field.Name + ": " + field.ReturnType
	}
	return fmt.Sprintf("%s(%s): %s", field.Name, strings.Join(args, ", "), field.ReturnType)
}
