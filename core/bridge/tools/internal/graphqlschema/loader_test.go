package graphqlschema

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBuildsReadableSchemaIndex(t *testing.T) {
	path := writeSchemaSnapshot(t, `{
  "root_queries": [
    {
      "name": "viewer",
      "return_type": "Viewer",
      "description": "Current viewer"
    }
  ],
  "types": [
    {
      "name": "Viewer",
      "fields": [
        {
          "name": "id",
          "return_type": "ID!",
          "description": "Viewer id"
        }
      ]
    }
  ]
}`)

	schema, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(schema.RootQueryList()) != 1 {
		t.Fatalf("unexpected root query count: %d", len(schema.RootQueryList()))
	}
	item, ok := schema.TypeByName("Viewer")
	if !ok {
		t.Fatal("expected Viewer type to be indexed")
	}
	if len(item.Fields) != 1 || item.Fields[0].Name != "id" {
		t.Fatalf("unexpected Viewer fields: %+v", item.Fields)
	}
}

func TestLoadRejectsInvalidSnapshot(t *testing.T) {
	path := writeSchemaSnapshot(t, `{
  "root_queries": [{"name":"","return_type":"Viewer"}],
  "types": []
}`)

	if _, err := Load(path); err == nil {
		t.Fatal("expected invalid schema snapshot error")
	}
}

func writeSchemaSnapshot(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "schema.json")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}
