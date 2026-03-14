package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ghost-os/bridge/tools/internal/graphqlschema"
)

func TestGraphQLSchemaLookupToolListsRootQueries(t *testing.T) {
	tool := NewGraphQLSchemaLookupTool(testGraphQLSchema(t)).(*GraphQLSchemaLookupTool)

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"list_root_queries"}`), "trace-schema-1")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var payload struct {
		Action      string                      `json:"action"`
		RootQueries []graphqlSchemaFieldPayload `json:"root_queries"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if payload.Action != graphqlSchemaLookupActionListRootQueries {
		t.Fatalf("unexpected action: %q", payload.Action)
	}
	if len(payload.RootQueries) != 1 || payload.RootQueries[0].Signature != "viewer(id: ID!): Viewer" {
		t.Fatalf("unexpected root queries: %+v", payload.RootQueries)
	}
}

func TestGraphQLSchemaLookupToolDescribesTypesAndFields(t *testing.T) {
	tool := NewGraphQLSchemaLookupTool(testGraphQLSchema(t)).(*GraphQLSchemaLookupTool)

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"describe_type","name":"Viewer"}`), "trace-schema-2")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var payload struct {
		Type graphqlSchemaTypePayload `json:"type"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if payload.Type.Name != "Viewer" {
		t.Fatalf("unexpected type name: %q", payload.Type.Name)
	}
	if len(payload.Type.Fields) != 2 || payload.Type.Fields[1].Signature != "projects(limit: Int): [Project!]!" {
		t.Fatalf("unexpected type fields: %+v", payload.Type.Fields)
	}
}

func TestGraphQLSchemaLookupToolFindsFieldsAndReturnsEmptyMisses(t *testing.T) {
	tool := NewGraphQLSchemaLookupTool(testGraphQLSchema(t)).(*GraphQLSchemaLookupTool)

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"find_field","name":"id"}`), "trace-schema-3")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var payload struct {
		Matches []graphqlSchemaFieldMatchPayload `json:"matches"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if len(payload.Matches) != 2 {
		t.Fatalf("unexpected field matches: %+v", payload.Matches)
	}

	miss, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"find_field","name":"missing"}`), "trace-schema-4")
	if err != nil {
		t.Fatalf("Execute miss: %v", err)
	}
	var missPayload struct {
		Matches []graphqlSchemaFieldMatchPayload `json:"matches"`
	}
	if err := json.Unmarshal([]byte(miss), &missPayload); err != nil {
		t.Fatalf("decode miss output: %v", err)
	}
	if len(missPayload.Matches) != 0 {
		t.Fatalf("expected no matches, got %+v", missPayload.Matches)
	}
}

func TestGraphQLSchemaLookupToolRejectsUnknownType(t *testing.T) {
	tool := NewGraphQLSchemaLookupTool(testGraphQLSchema(t))

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"describe_type","name":"Missing"}`), "trace-schema-5")
	if err == nil {
		t.Fatal("expected error for missing type")
	}
}

func testGraphQLSchema(t *testing.T) *graphqlschema.Schema {
	t.Helper()

	path := filepath.Join(t.TempDir(), "graphql-schema.json")
	if err := os.WriteFile(path, []byte(`{
  "root_queries": [
    {
      "name": "viewer",
      "return_type": "Viewer",
      "args": [{"name":"id","type":"ID!"}]
    }
  ],
  "types": [
    {
      "name": "Project",
      "fields": [
        {"name":"id","return_type":"ID!"}
      ]
    },
    {
      "name": "Viewer",
      "fields": [
        {"name":"id","return_type":"ID!"},
        {
          "name":"projects",
          "return_type":"[Project!]!",
          "args":[{"name":"limit","type":"Int"}]
        }
      ]
    }
  ]
}`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	schema, err := graphqlschema.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return schema
}
