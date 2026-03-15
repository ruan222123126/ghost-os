package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"strings"
	"testing"
)

func TestGraphQLSchemaLookupToolListsSourcesAndDomains(t *testing.T) {
	registry := testGraphQLRegistry(t, GraphQLRegistryConfig{
		DefaultSource: "crm",
		Sources: []GraphQLSourceConfig{
			testGraphQLSourceConfig(t, "billing", "https://billing.test/query", GraphQLDomainConfig{
				Name:        "payments",
				RootQueries: []string{"order"},
				Types:       []string{"Order"},
			}),
			testGraphQLSourceConfig(t, "crm", "https://crm.test/query", GraphQLDomainConfig{
				Name:        "people",
				RootQueries: []string{"viewer"},
				Types:       []string{"Viewer"},
			}),
		},
	})
	tool := NewGraphQLSchemaLookupTool(registry).(*GraphQLSchemaLookupTool)

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"list_sources"}`), "trace-schema-1")
	if err != nil {
		t.Fatalf("Execute list_sources: %v", err)
	}
	var sources struct {
		DefaultSource string                       `json:"default_source"`
		Sources       []graphqlSchemaSourcePayload `json:"sources"`
	}
	if err := json.Unmarshal([]byte(output), &sources); err != nil {
		t.Fatalf("decode list_sources output: %v", err)
	}
	if sources.DefaultSource != "crm" {
		t.Fatalf("unexpected default source: %q", sources.DefaultSource)
	}
	if len(sources.Sources) != 2 || sources.Sources[0].Name != "billing" || sources.Sources[1].Name != "crm" {
		t.Fatalf("expected stable source order, got %+v", sources.Sources)
	}

	output, err = tool.Execute(context.Background(), json.RawMessage(`{"action":"list_domains","source":"crm"}`), "trace-schema-2")
	if err != nil {
		t.Fatalf("Execute list_domains: %v", err)
	}
	var domains struct {
		Domains []graphqlSchemaDomainPayload `json:"domains"`
	}
	if err := json.Unmarshal([]byte(output), &domains); err != nil {
		t.Fatalf("decode list_domains output: %v", err)
	}
	if len(domains.Domains) != 1 || domains.Domains[0].Name != "people" {
		t.Fatalf("unexpected domains: %+v", domains.Domains)
	}
}

func TestGraphQLSchemaLookupToolAppliesDomainFilters(t *testing.T) {
	registry := testGraphQLRegistry(t, GraphQLRegistryConfig{
		DefaultSource: "crm",
		Sources: []GraphQLSourceConfig{
			testGraphQLSourceConfig(t, "crm", "https://crm.test/query",
				GraphQLDomainConfig{
					Name:        "orders",
					RootQueries: []string{"order"},
					Types:       []string{"Order"},
				},
				GraphQLDomainConfig{
					Name:        "people",
					RootQueries: []string{"viewer"},
					Types:       []string{"Viewer"},
				},
			),
		},
	})
	tool := NewGraphQLSchemaLookupTool(registry).(*GraphQLSchemaLookupTool)

	output, err := tool.Execute(context.Background(), json.RawMessage(`{
		"action":"list_root_queries",
		"source":"crm",
		"domain":"orders"
	}`), "trace-schema-3")
	if err != nil {
		t.Fatalf("Execute list_root_queries: %v", err)
	}
	var rootQueries struct {
		RootQueries []graphqlSchemaFieldPayload `json:"root_queries"`
	}
	if err := json.Unmarshal([]byte(output), &rootQueries); err != nil {
		t.Fatalf("decode list_root_queries output: %v", err)
	}
	if len(rootQueries.RootQueries) != 1 || rootQueries.RootQueries[0].Name != "order" {
		t.Fatalf("unexpected root queries: %+v", rootQueries.RootQueries)
	}

	_, err = tool.Execute(context.Background(), json.RawMessage(`{
		"action":"describe_type",
		"source":"crm",
		"domain":"orders",
		"name":"Viewer"
	}`), "trace-schema-4")
	if err == nil || !strings.Contains(err.Error(), `domain "orders"`) {
		t.Fatalf("expected domain describe_type miss, got %v", err)
	}

	output, err = tool.Execute(context.Background(), json.RawMessage(`{
		"action":"find_field",
		"source":"crm",
		"domain":"orders",
		"name":"id"
	}`), "trace-schema-5")
	if err != nil {
		t.Fatalf("Execute find_field: %v", err)
	}
	var matches struct {
		Matches []graphqlSchemaFieldMatchPayload `json:"matches"`
	}
	if err := json.Unmarshal([]byte(output), &matches); err != nil {
		t.Fatalf("decode find_field output: %v", err)
	}
	if len(matches.Matches) != 1 || matches.Matches[0].TypeName != "Order" {
		t.Fatalf("unexpected field matches: %+v", matches.Matches)
	}
}

func TestGraphQLSchemaLookupToolListsAllowedRootMutations(t *testing.T) {
	registry := testGraphQLRegistry(t, GraphQLRegistryConfig{
		Sources: []GraphQLSourceConfig{{
			Name:             "crm",
			Endpoint:         "https://crm.test/query",
			SchemaPath:       writeGraphQLSchema(t, "crm-mutations"),
			TimeoutMS:        3000,
			MaxResponseBytes: 4096,
			MaxDepth:         6,
			MaxFields:        16,
			MaxRootFields:    2,
			MaxFragments:     2,
			Domains: []GraphQLDomainConfig{{
				Name:        "people",
				RootQueries: []string{"viewer"},
				Types:       []string{"Viewer", "MutationPayload"},
			}},
		}},
		MutationPolicies: []GraphQLMutationPolicyConfig{{
			Name:              "update_viewer",
			Source:            "crm",
			Domain:            "people",
			RootMutation:      "updateViewer",
			IdempotencyMode:   graphQLMutationIdempotencyModeHeader,
			IdempotencyHeader: "Idempotency-Key",
		}},
	})
	tool := NewGraphQLSchemaLookupTool(registry).(*GraphQLSchemaLookupTool)

	output, err := tool.Execute(context.Background(), json.RawMessage(`{
		"action":"list_root_mutations",
		"source":"crm",
		"domain":"people"
	}`), "trace-schema-mutation-list")
	if err != nil {
		t.Fatalf("Execute list_root_mutations: %v", err)
	}
	if !strings.Contains(output, `"updateViewer"`) || strings.Contains(output, `"archiveViewer"`) {
		t.Fatalf("expected only allowlisted mutation in output, got %s", output)
	}

	output, err = tool.Execute(context.Background(), json.RawMessage(`{
		"action":"describe_mutation_policy",
		"source":"crm",
		"domain":"people",
		"name":"updateViewer"
	}`), "trace-schema-mutation-policy")
	if err != nil {
		t.Fatalf("Execute describe_mutation_policy: %v", err)
	}
	if !strings.Contains(output, `"name":"update_viewer"`) ||
		!strings.Contains(output, `"root_mutation"`) ||
		!strings.Contains(output, `"idempotency_mode":"header"`) {
		t.Fatalf("unexpected mutation policy output: %s", output)
	}
}

func TestGraphQLSchemaLookupToolLogsSuccessAndFailure(t *testing.T) {
	registry := testGraphQLRegistry(t, GraphQLRegistryConfig{
		Sources: []GraphQLSourceConfig{
			testGraphQLSourceConfig(t, "crm", "https://crm.test/query", GraphQLDomainConfig{
				Name:        "people",
				RootQueries: []string{"viewer"},
				Types:       []string{"Viewer"},
			}),
		},
	})
	tool := NewGraphQLSchemaLookupTool(registry).(*GraphQLSchemaLookupTool)
	logs := &bytes.Buffer{}
	original := log.Writer()
	log.SetOutput(logs)
	t.Cleanup(func() {
		log.SetOutput(original)
	})

	if _, err := tool.Execute(context.Background(), json.RawMessage(`{
		"action":"list_root_queries",
		"source":"crm",
		"domain":"people"
	}`), "trace-schema-log-success"); err != nil {
		t.Fatalf("Execute success: %v", err)
	}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{
		"action":"list_domains",
		"source":"missing"
	}`), "trace-schema-log-failure"); err == nil {
		t.Fatal("expected missing source error")
	}
	if !strings.Contains(logs.String(), "trace_id=trace-schema-log-success tool=graphql_schema_lookup source=crm domain=people action=list_root_queries") {
		t.Fatalf("expected success lookup log, got %q", logs.String())
	}
	if !strings.Contains(logs.String(), "trace_id=trace-schema-log-failure tool=graphql_schema_lookup source=missing domain= action=list_domains") {
		t.Fatalf("expected failure lookup log, got %q", logs.String())
	}
	if !strings.Contains(logs.String(), "status=error") {
		t.Fatalf("expected error status in logs, got %q", logs.String())
	}
}
