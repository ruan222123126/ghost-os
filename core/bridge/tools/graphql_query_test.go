package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGraphQLQueryToolUsesDefaultSourceAndLogsSuccess(t *testing.T) {
	registry := testGraphQLRegistry(t, GraphQLRegistryConfig{
		DefaultSource: "crm",
		Sources: []GraphQLSourceConfig{
			testGraphQLSourceConfig(t, "crm", "https://crm.test/query", GraphQLDomainConfig{
				Name:        "people",
				RootQueries: []string{"viewer"},
				Types:       []string{"Viewer"},
			}),
			testGraphQLSourceConfig(t, "billing", "https://billing.test/query", GraphQLDomainConfig{
				Name:        "orders",
				RootQueries: []string{"order"},
				Types:       []string{"Order"},
			}),
		},
	})

	tool := &GraphQLQueryTool{
		httpClient: &http.Client{
			Transport: graphQLRoundTripper(func(req *http.Request) (*http.Response, error) {
				if req.URL.String() != "https://crm.test/query" {
					t.Fatalf("unexpected endpoint: %s", req.URL.String())
				}
				body, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatalf("ReadAll: %v", err)
				}
				var payload struct {
					OperationName string         `json:"operationName"`
					Variables     map[string]any `json:"variables"`
				}
				if err := json.Unmarshal(body, &payload); err != nil {
					t.Fatalf("Unmarshal: %v", err)
				}
				if payload.OperationName != "ViewerQuery" {
					t.Fatalf("unexpected operation: %q", payload.OperationName)
				}
				if payload.Variables["id"] != "user-1" {
					t.Fatalf("unexpected variables: %+v", payload.Variables)
				}
				return newGraphQLResponse(http.StatusOK, `{"data":{"viewer":{"id":"user-1"}}}`), nil
			}),
		},
		registry: registry,
	}

	logs := captureGraphQLLogs(t)
	output, err := tool.Execute(context.Background(), json.RawMessage(`{
		"domain":"people",
		"query":"query ViewerQuery($id: ID!) { viewer(id: $id) { id } }",
		"variables":{"id":"user-1"},
		"operation_name":"ViewerQuery"
	}`), "trace-graphql-success")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if output != `{"data":{"viewer":{"id":"user-1"}}}` {
		t.Fatalf("unexpected output: %s", output)
	}
	if !strings.Contains(logs.String(), "trace_id=trace-graphql-success tool=graphql_query source=crm domain=people") {
		t.Fatalf("expected success log, got %q", logs.String())
	}
	if !strings.Contains(logs.String(), "operation=ViewerQuery") || !strings.Contains(logs.String(), "status=success") {
		t.Fatalf("expected query stats in log, got %q", logs.String())
	}
}

func TestGraphQLQueryToolRequiresSourceWithoutDefault(t *testing.T) {
	registry := testGraphQLRegistry(t, GraphQLRegistryConfig{
		Sources: []GraphQLSourceConfig{
			testGraphQLSourceConfig(t, "crm", "https://crm.test/query"),
			testGraphQLSourceConfig(t, "billing", "https://billing.test/query"),
		},
	})
	tool := NewGraphQLQueryTool(registry)

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"query { viewer { id } }"}`), "trace-graphql-source")
	if err == nil || !strings.Contains(err.Error(), "source is required") {
		t.Fatalf("expected missing source error, got %v", err)
	}
}

func TestGraphQLQueryToolLogsFailures(t *testing.T) {
	registry := testGraphQLRegistry(t, GraphQLRegistryConfig{
		Sources: []GraphQLSourceConfig{
			testGraphQLSourceConfig(t, "crm", "https://crm.test/query", GraphQLDomainConfig{
				Name:        "people",
				RootQueries: []string{"viewer"},
				Types:       []string{"Viewer"},
			}),
		},
	})
	tool := &GraphQLQueryTool{
		httpClient: &http.Client{
			Transport: graphQLRoundTripper(func(*http.Request) (*http.Response, error) {
				return newGraphQLResponse(http.StatusInternalServerError, `{"message":"boom"}`), nil
			}),
		},
		registry: registry,
	}

	logs := captureGraphQLLogs(t)
	_, err := tool.Execute(context.Background(), json.RawMessage(`{
		"source":"crm",
		"domain":"people",
		"query":"query ViewerQuery { viewer { id } }",
		"operation_name":"ViewerQuery"
	}`), "trace-graphql-failure")
	if err == nil || !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("expected HTTP failure, got %v", err)
	}
	if !strings.Contains(logs.String(), "trace_id=trace-graphql-failure tool=graphql_query source=crm domain=people") {
		t.Fatalf("expected failure log source/domain, got %q", logs.String())
	}
	if !strings.Contains(logs.String(), "status=error") || !strings.Contains(logs.String(), `error="graphql request returned status 500`) {
		t.Fatalf("expected failure log details, got %q", logs.String())
	}
}

type graphQLRoundTripper func(*http.Request) (*http.Response, error)

func (fn graphQLRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func newGraphQLResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header: http.Header{
			"Content-Type": []string{"application/json; charset=utf-8"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}
}

func captureGraphQLLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	buffer := &bytes.Buffer{}
	original := log.Writer()
	log.SetOutput(buffer)
	t.Cleanup(func() {
		log.SetOutput(original)
	})
	return buffer
}

func testGraphQLRegistry(t *testing.T, cfg GraphQLRegistryConfig) *GraphQLSourceRegistry {
	t.Helper()
	registry, err := NewGraphQLSourceRegistry(cfg)
	if err != nil {
		t.Fatalf("NewGraphQLSourceRegistry: %v", err)
	}
	return registry
}

func testGraphQLSourceConfig(
	t *testing.T,
	name string,
	endpoint string,
	domains ...GraphQLDomainConfig,
) GraphQLSourceConfig {
	t.Helper()
	return GraphQLSourceConfig{
		Name:             name,
		Endpoint:         endpoint,
		SchemaPath:       writeGraphQLSchema(t, name),
		TimeoutMS:        3000,
		MaxResponseBytes: 4096,
		MaxDepth:         6,
		MaxFields:        16,
		MaxRootFields:    2,
		MaxFragments:     2,
		Domains:          domains,
	}
}

func writeGraphQLSchema(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name+"-schema.json")
	contents := `{
  "root_queries": [
    {"name":"viewer","return_type":"Viewer"},
    {"name":"order","return_type":"Order","args":[{"name":"id","type":"ID!"}]}
  ],
  "types": [
    {
      "name":"Order",
      "fields":[
        {"name":"id","return_type":"ID!"},
        {"name":"status","return_type":"String!"}
      ]
    },
    {
      "name":"Viewer",
      "fields":[
        {"name":"id","return_type":"ID!"},
        {"name":"name","return_type":"String!"},
        {"name":"manager","return_type":"Viewer"}
      ]
    }
  ]
}`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}
