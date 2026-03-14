package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestGraphQLQueryToolExecutesReadOnlyQueries(t *testing.T) {
	client := &http.Client{
		Transport: graphQLRoundTripper(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", req.Method)
			}
			if req.URL.String() != "https://graphql.test/query" {
				t.Fatalf("unexpected endpoint: %s", req.URL.String())
			}
			if got := req.Header.Get("Authorization"); got != "Bearer test-api-key" {
				t.Fatalf("unexpected authorization header: %q", got)
			}
			if got := req.Header.Get("X-Trace"); got != "graphql-test" {
				t.Fatalf("unexpected custom header: %q", got)
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("ReadAll: %v", err)
			}
			var payload struct {
				Query         string         `json:"query"`
				Variables     map[string]any `json:"variables"`
				OperationName string         `json:"operationName"`
			}
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if payload.OperationName != "ViewerQuery" {
				t.Fatalf("unexpected operation name: %q", payload.OperationName)
			}
			if payload.Variables["id"] != "user-1" {
				t.Fatalf("unexpected variables: %+v", payload.Variables)
			}
			return newGraphQLResponse(http.StatusOK, `{"data":{"viewer":{"id":"user-1"}},"extensions":{"cost":1}}`), nil
		}),
	}

	tool := &GraphQLQueryTool{
		httpClient: client,
		config: GraphQLQueryConfig{
			Endpoint:         "https://graphql.test/query",
			APIKey:           "test-api-key",
			TimeoutMS:        100,
			MaxResponseBytes: 1024,
			Headers: map[string]string{
				"X-Trace": "graphql-test",
			},
		},
	}

	output, err := tool.Execute(context.Background(), json.RawMessage(`{
		"query":"query ViewerQuery($id: ID!) { viewer(id: $id) { id } }",
		"variables":{"id":"user-1"},
		"operation_name":"ViewerQuery"
	}`), "trace-graphql-1")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if output != `{"data":{"viewer":{"id":"user-1"}},"extensions":{"cost":1}}` {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestGraphQLQueryToolRejectsMutations(t *testing.T) {
	tool := NewGraphQLQueryTool(GraphQLQueryConfig{
		Endpoint:         "https://graphql.test/query",
		TimeoutMS:        100,
		MaxResponseBytes: 1024,
	})

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"mutation { updateUser { id } }"}`), "trace-graphql-2")
	if err == nil || !strings.Contains(err.Error(), "only supports query operations") {
		t.Fatalf("expected mutation rejection, got %v", err)
	}
}

func TestGraphQLQueryToolRejectsHTTPFailures(t *testing.T) {
	tool := &GraphQLQueryTool{
		httpClient: &http.Client{
			Transport: graphQLRoundTripper(func(*http.Request) (*http.Response, error) {
				return newGraphQLResponse(http.StatusInternalServerError, `{"message":"boom"}`), nil
			}),
		},
		config: GraphQLQueryConfig{
			Endpoint:         "https://graphql.test/query",
			TimeoutMS:        100,
			MaxResponseBytes: 1024,
		},
	}

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"query { viewer { id } }"}`), "trace-graphql-3")
	if err == nil || !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("expected HTTP failure, got %v", err)
	}
}

func TestGraphQLQueryToolRejectsGraphQLErrors(t *testing.T) {
	tool := &GraphQLQueryTool{
		httpClient: &http.Client{
			Transport: graphQLRoundTripper(func(*http.Request) (*http.Response, error) {
				return newGraphQLResponse(http.StatusOK, `{"errors":[{"message":"bad field"}]}`), nil
			}),
		},
		config: GraphQLQueryConfig{
			Endpoint:         "https://graphql.test/query",
			TimeoutMS:        100,
			MaxResponseBytes: 1024,
		},
	}

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"query { viewer { badField } }"}`), "trace-graphql-4")
	if err == nil || !strings.Contains(err.Error(), `{"message":"bad field"}`) {
		t.Fatalf("expected graphql errors rejection, got %v", err)
	}
}

func TestGraphQLQueryToolRejectsOversizedResponses(t *testing.T) {
	tool := &GraphQLQueryTool{
		httpClient: &http.Client{
			Transport: graphQLRoundTripper(func(*http.Request) (*http.Response, error) {
				return newGraphQLResponse(http.StatusOK, `{"data":{"blob":"`+strings.Repeat("x", 2048)+`"}}`), nil
			}),
		},
		config: GraphQLQueryConfig{
			Endpoint:         "https://graphql.test/query",
			TimeoutMS:        100,
			MaxResponseBytes: 128,
		},
	}

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"query { viewer { id } }"}`), "trace-graphql-5")
	if err == nil || !strings.Contains(err.Error(), "max_response_bytes") {
		t.Fatalf("expected oversized response rejection, got %v", err)
	}
}

func TestGraphQLQueryToolRequiresConfiguredEndpoint(t *testing.T) {
	tool := NewGraphQLQueryTool(GraphQLQueryConfig{
		TimeoutMS:        100,
		MaxResponseBytes: 1024,
	})

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"query { viewer { id } }"}`), "trace-graphql-6")
	if err == nil || !strings.Contains(err.Error(), "endpoint is not configured") {
		t.Fatalf("expected endpoint configuration error, got %v", err)
	}
}

func TestGraphQLQueryToolTimesOutRequests(t *testing.T) {
	tool := &GraphQLQueryTool{
		httpClient: &http.Client{
			Transport: graphQLRoundTripper(func(req *http.Request) (*http.Response, error) {
				<-req.Context().Done()
				return nil, req.Context().Err()
			}),
		},
		config: GraphQLQueryConfig{
			Endpoint:         "https://graphql.test/query",
			TimeoutMS:        10,
			MaxResponseBytes: 1024,
		},
	}

	start := time.Now()
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"query { viewer { id } }"}`), "trace-graphql-7")
	if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("expected timeout error, got %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("expected timeout to fail quickly, took %s", time.Since(start))
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
