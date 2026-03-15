package graphqlschema

import "testing"

func TestLoadIndexesRootMutations(t *testing.T) {
	path := writeSchemaSnapshot(t, `{
  "root_queries": [{"name":"viewer","return_type":"Viewer"}],
  "root_mutations": [{"name":"updateViewer","return_type":"MutationPayload"}],
  "types": [
    {"name":"Viewer","fields":[{"name":"id","return_type":"ID!"}]},
    {"name":"MutationPayload","fields":[{"name":"ok","return_type":"Boolean!"}]}
  ]
}`)

	schema, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(schema.RootMutationList()) != 1 {
		t.Fatalf("unexpected root mutation count: %d", len(schema.RootMutationList()))
	}
	item, ok := schema.RootMutationByName("updateViewer")
	if !ok || item.Name != "updateViewer" {
		t.Fatalf("expected indexed root mutation, got %+v ok=%v", item, ok)
	}
}

func TestRegistryRejectsMutationPolicyForMissingRootMutation(t *testing.T) {
	path := writeSchemaSnapshot(t, `{
  "root_queries": [{"name":"viewer","return_type":"Viewer"}],
  "root_mutations": [{"name":"updateViewer","return_type":"MutationPayload"}],
  "types": [
    {"name":"Viewer","fields":[{"name":"id","return_type":"ID!"}]},
    {"name":"MutationPayload","fields":[{"name":"ok","return_type":"Boolean!"}]}
  ]
}`)

	_, err := NewRegistry(RegistryConfig{
		Sources: []SourceConfig{{
			Name:             "crm",
			Endpoint:         "https://crm.test/query",
			SchemaPath:       path,
			TimeoutMS:        3000,
			MaxResponseBytes: 4096,
			MaxDepth:         6,
			MaxFields:        16,
			MaxRootFields:    2,
			MaxFragments:     4,
			Domains: []DomainConfig{{
				Name:        "people",
				RootQueries: []string{"viewer"},
				Types:       []string{"Viewer", "MutationPayload"},
			}},
		}},
		MutationPolicies: []MutationPolicyConfig{{
			Name:              "bad_policy",
			Source:            "crm",
			Domain:            "people",
			RootMutation:      "archiveViewer",
			IdempotencyMode:   mutationIdempotencyModeHeader,
			IdempotencyHeader: "Idempotency-Key",
		}},
	})
	if err == nil {
		t.Fatal("expected missing root mutation policy error")
	}
}

func TestRegistryRejectsMutationPolicyWithoutIdempotency(t *testing.T) {
	path := writeSchemaSnapshot(t, `{
  "root_queries": [{"name":"viewer","return_type":"Viewer"}],
  "root_mutations": [{"name":"updateViewer","return_type":"MutationPayload"}],
  "types": [
    {"name":"Viewer","fields":[{"name":"id","return_type":"ID!"}]},
    {"name":"MutationPayload","fields":[{"name":"ok","return_type":"Boolean!"}]}
  ]
}`)

	_, err := NewRegistry(RegistryConfig{
		Sources: []SourceConfig{{
			Name:             "crm",
			Endpoint:         "https://crm.test/query",
			SchemaPath:       path,
			TimeoutMS:        3000,
			MaxResponseBytes: 4096,
			MaxDepth:         6,
			MaxFields:        16,
			MaxRootFields:    2,
			MaxFragments:     4,
			Domains: []DomainConfig{{
				Name:        "people",
				RootQueries: []string{"viewer"},
				Types:       []string{"Viewer", "MutationPayload"},
			}},
		}},
		MutationPolicies: []MutationPolicyConfig{{
			Name:         "missing_idempotency",
			Source:       "crm",
			Domain:       "people",
			RootMutation: "updateViewer",
		}},
	})
	if err == nil {
		t.Fatal("expected missing idempotency policy error")
	}
}
