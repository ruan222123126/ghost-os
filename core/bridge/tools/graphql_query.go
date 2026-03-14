package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	defaultGraphQLToolTimeoutMS        = 10_000
	defaultGraphQLToolMaxResponseBytes = 1 << 20
)

type GraphQLQueryConfig struct {
	Endpoint         string
	APIKey           string
	TimeoutMS        int
	MaxResponseBytes int
	Headers          map[string]string
}

type GraphQLQueryTool struct {
	httpClient *http.Client
	config     GraphQLQueryConfig
}

type graphQLQueryArgs struct {
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables,omitempty"`
	OperationName string         `json:"operation_name,omitempty"`
}

func NewGraphQLQueryTool(cfg GraphQLQueryConfig) Tool {
	normalized := normalizeGraphQLQueryConfig(cfg)
	return &GraphQLQueryTool{
		httpClient: &http.Client{Timeout: time.Duration(normalized.TimeoutMS) * time.Millisecond},
		config:     normalized,
	}
}

func (GraphQLQueryTool) Name() string {
	return "graphql_query"
}

func (GraphQLQueryTool) Description() string {
	return "Execute a read-only GraphQL query against the configured endpoint. Use graphql_schema_lookup first, and never invent results when the query fails."
}

func (GraphQLQueryTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"query":{"type":"string","description":"GraphQL query text. Only query operations are allowed."},
			"variables":{"type":"object","description":"Optional GraphQL variables object."},
			"operation_name":{"type":"string","description":"Optional operation name when the document contains multiple query operations."}
		},
		"required":["query"],
		"additionalProperties":false
	}`)
}

func (t *GraphQLQueryTool) Execute(ctx context.Context, argsJSON json.RawMessage, _ string) (string, error) {
	var args graphQLQueryArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}
	if strings.TrimSpace(args.Query) == "" {
		return "", fmt.Errorf("query is required")
	}
	if err := validateGraphQLQueryDocument(args.Query, args.OperationName); err != nil {
		return "", err
	}
	body, err := t.executeRequest(ctx, args)
	if err != nil {
		return "", err
	}
	if err := validateGraphQLResponseBody(body); err != nil {
		return "", err
	}
	return string(body), nil
}

func normalizeGraphQLQueryConfig(cfg GraphQLQueryConfig) GraphQLQueryConfig {
	out := cfg
	out.Endpoint = strings.TrimSpace(out.Endpoint)
	out.APIKey = strings.TrimSpace(out.APIKey)
	out.TimeoutMS = normalizeGraphQLQueryInt(out.TimeoutMS, defaultGraphQLToolTimeoutMS)
	out.MaxResponseBytes = normalizeGraphQLQueryInt(out.MaxResponseBytes, defaultGraphQLToolMaxResponseBytes)
	out.Headers = cloneGraphQLHeaders(out.Headers)
	return out
}

func normalizeGraphQLQueryInt(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func cloneGraphQLHeaders(raw map[string]string) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]string, len(raw))
	for key, value := range raw {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		out[trimmedKey] = strings.TrimSpace(value)
	}
	return out
}
