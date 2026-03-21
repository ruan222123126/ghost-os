package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type GraphQLQueryTool struct {
	httpClient *http.Client
	registry   *GraphQLSourceRegistry
}

type graphQLQueryArgs struct {
	Source        string         `json:"source,omitempty"`
	Domain        string         `json:"domain,omitempty"`
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables,omitempty"`
	OperationName string         `json:"operation_name,omitempty"`
}

func NewGraphQLQueryTool(registry *GraphQLSourceRegistry) Tool {
	return &GraphQLQueryTool{
		httpClient: &http.Client{},
		registry:   registry,
	}
}

func (GraphQLQueryTool) Name() string {
	return "graphql_query"
}

func (GraphQLQueryTool) Description() string {
	return "Execute a read-only GraphQL query against a configured source. Inspect the available source and domain metadata first when it is available, then query only the selected source/domain."
}

func (GraphQLQueryTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"source":{"type":"string","description":"Optional GraphQL source name. Omit only when a single source exists or graphql_default_source is configured."},
			"domain":{"type":"string","description":"Optional domain name. When set, every top-level root field must belong to that domain."},
			"query":{"type":"string","description":"GraphQL query text. Only query operations are allowed."},
			"variables":{"type":"object","description":"Optional GraphQL variables object."},
			"operation_name":{"type":"string","description":"Optional operation name when the document contains multiple query operations."}
		},
		"required":["query"],
		"additionalProperties":false
	}`)
}

func (t *GraphQLQueryTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	args, err := decodeGraphQLQueryArgs(argsJSON)
	if err != nil {
		return "", err
	}

	startedAt := time.Now()
	source, err := t.registry.resolveSource(args.Source)
	if err != nil {
		logGraphQLQuery(traceID, args.Source, args.Domain, graphQLQuerySummary{}, 0, time.Since(startedAt), err)
		return "", err
	}

	summary, err := validateGraphQLQueryDocument(args.Query, args.OperationName, source, args.Domain)
	if err != nil {
		logGraphQLQuery(traceID, source.Name, args.Domain, summary, 0, time.Since(startedAt), err)
		return "", err
	}

	body, err := t.executeRequest(ctx, source, args)
	if err != nil {
		logGraphQLQuery(traceID, source.Name, args.Domain, summary, 0, time.Since(startedAt), err)
		return "", err
	}
	if err := validateGraphQLResponseBody(body); err != nil {
		logGraphQLQuery(traceID, source.Name, args.Domain, summary, len(body), time.Since(startedAt), err)
		return "", err
	}

	logGraphQLQuery(traceID, source.Name, args.Domain, summary, len(body), time.Since(startedAt), nil)
	return string(body), nil
}

func decodeGraphQLQueryArgs(argsJSON json.RawMessage) (graphQLQueryArgs, error) {
	var args graphQLQueryArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return graphQLQueryArgs{}, fmt.Errorf("decode args: %w", err)
	}
	if strings.TrimSpace(args.Query) == "" {
		return graphQLQueryArgs{}, fmt.Errorf("query is required")
	}
	args.Source = strings.TrimSpace(args.Source)
	args.Domain = strings.TrimSpace(args.Domain)
	args.OperationName = strings.TrimSpace(args.OperationName)
	return args, nil
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
