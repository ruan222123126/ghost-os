package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestGraphQLQueryToolRejectsBudgetAndDomainViolations(t *testing.T) {
	tests := []struct {
		name      string
		sourceCfg GraphQLSourceConfig
		args      string
		want      string
	}{
		{
			name: "depth",
			sourceCfg: GraphQLSourceConfig{
				Name:             "crm",
				Endpoint:         "https://crm.test/query",
				SchemaPath:       writeGraphQLSchema(t, "crm"),
				TimeoutMS:        3000,
				MaxResponseBytes: 4096,
				MaxDepth:         2,
				MaxFields:        16,
				MaxRootFields:    2,
				MaxFragments:     2,
			},
			args: `{"source":"crm","query":"query { viewer { manager { id } } }"}`,
			want: "max_depth=2",
		},
		{
			name: "fields",
			sourceCfg: GraphQLSourceConfig{
				Name:             "crm",
				Endpoint:         "https://crm.test/query",
				SchemaPath:       writeGraphQLSchema(t, "crm"),
				TimeoutMS:        3000,
				MaxResponseBytes: 4096,
				MaxDepth:         6,
				MaxFields:        2,
				MaxRootFields:    2,
				MaxFragments:     2,
			},
			args: `{"source":"crm","query":"query { viewer { id name } }"}`,
			want: "max_fields=2",
		},
		{
			name: "root fields",
			sourceCfg: GraphQLSourceConfig{
				Name:             "crm",
				Endpoint:         "https://crm.test/query",
				SchemaPath:       writeGraphQLSchema(t, "crm"),
				TimeoutMS:        3000,
				MaxResponseBytes: 4096,
				MaxDepth:         6,
				MaxFields:        16,
				MaxRootFields:    1,
				MaxFragments:     2,
			},
			args: `{"source":"crm","query":"query { viewer { id } order(id: \"1\") { id } }"}`,
			want: "max_root_fields=1",
		},
		{
			name: "introspection",
			sourceCfg: GraphQLSourceConfig{
				Name:             "crm",
				Endpoint:         "https://crm.test/query",
				SchemaPath:       writeGraphQLSchema(t, "crm"),
				TimeoutMS:        3000,
				MaxResponseBytes: 4096,
				MaxDepth:         6,
				MaxFields:        16,
				MaxRootFields:    2,
				MaxFragments:     2,
			},
			args: `{"source":"crm","query":"query { __schema { types { name } } }"}`,
			want: "introspection fields are not allowed",
		},
		{
			name: "domain root field",
			sourceCfg: GraphQLSourceConfig{
				Name:             "crm",
				Endpoint:         "https://crm.test/query",
				SchemaPath:       writeGraphQLSchema(t, "crm"),
				TimeoutMS:        3000,
				MaxResponseBytes: 4096,
				MaxDepth:         6,
				MaxFields:        16,
				MaxRootFields:    2,
				MaxFragments:     2,
				Domains: []GraphQLDomainConfig{{
					Name:        "orders",
					RootQueries: []string{"order"},
					Types:       []string{"Order"},
				}},
			},
			args: `{"source":"crm","domain":"orders","query":"query { viewer { id } }"}`,
			want: `outside domain "orders"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			registry := testGraphQLRegistry(t, GraphQLRegistryConfig{
				Sources: []GraphQLSourceConfig{tc.sourceCfg},
			})
			tool := NewGraphQLQueryTool(registry)
			_, err := tool.Execute(context.Background(), json.RawMessage(tc.args), "trace-budget")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q error, got %v", tc.want, err)
			}
		})
	}
}

func TestGraphQLQueryToolTimesOutRequests(t *testing.T) {
	registry := testGraphQLRegistry(t, GraphQLRegistryConfig{
		Sources: []GraphQLSourceConfig{{
			Name:             "crm",
			Endpoint:         "https://crm.test/query",
			SchemaPath:       writeGraphQLSchema(t, "crm"),
			TimeoutMS:        10,
			MaxResponseBytes: 4096,
			MaxDepth:         6,
			MaxFields:        16,
			MaxRootFields:    2,
			MaxFragments:     2,
		}},
	})
	tool := &GraphQLQueryTool{
		httpClient: &http.Client{
			Transport: graphQLRoundTripper(func(req *http.Request) (*http.Response, error) {
				<-req.Context().Done()
				return nil, req.Context().Err()
			}),
		},
		registry: registry,
	}

	start := time.Now()
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"source":"crm","query":"query { viewer { id } }"}`), "trace-graphql-timeout")
	if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("expected timeout error, got %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("expected timeout to fail quickly, took %s", time.Since(start))
	}
}
