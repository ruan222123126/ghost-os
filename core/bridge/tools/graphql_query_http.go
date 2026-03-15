package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ghost-os/bridge/tools/internal/graphqlschema"
)

const (
	graphQLRequestContentType = "application/json"
	graphQLResponseBodyMargin = 1
	graphQLHTTPBodyPreviewLen = 256
)

type graphQLRequestPayload struct {
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables,omitempty"`
	OperationName string         `json:"operationName,omitempty"`
}

func executeGraphQLRequest(
	ctx context.Context,
	httpClient *http.Client,
	source *graphqlschema.Source,
	payload graphQLRequestPayload,
) ([]byte, error) {
	if httpClient == nil {
		return nil, fmt.Errorf("graphql http client is not configured")
	}
	if source == nil {
		return nil, fmt.Errorf("graphql source is not configured")
	}
	if source.Endpoint == "" {
		return nil, fmt.Errorf("graphql source %q endpoint is not configured", source.Name)
	}

	requestCtx, cancel := graphQLRequestContext(ctx, source.TimeoutMS)
	defer cancel()

	req, err := buildGraphQLRequest(requestCtx, source, payload)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("graphql request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := readGraphQLResponseBody(resp.Body, source.MaxResponseBytes)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("graphql request returned status %d: %s", resp.StatusCode, previewGraphQLBody(body))
	}
	return body, nil
}

func (t *GraphQLQueryTool) executeRequest(
	ctx context.Context,
	source *graphqlschema.Source,
	args graphQLQueryArgs,
) ([]byte, error) {
	var httpClient *http.Client
	if t != nil {
		httpClient = t.httpClient
	}
	return executeGraphQLRequest(ctx, httpClient, source, graphQLRequestPayload{
		Query:         args.Query,
		Variables:     args.Variables,
		OperationName: args.OperationName,
	})
}

func buildGraphQLRequest(
	ctx context.Context,
	source *graphqlschema.Source,
	payload graphQLRequestPayload,
) (*http.Request, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode graphql request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, source.Endpoint, bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("build graphql request: %w", err)
	}
	applyGraphQLRequestHeaders(req, source)
	return req, nil
}

func graphQLRequestContext(ctx context.Context, timeoutMS int) (context.Context, context.CancelFunc) {
	timeout := time.Duration(timeoutMS) * time.Millisecond
	if timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, timeout)
}

func applyGraphQLRequestHeaders(req *http.Request, source *graphqlschema.Source) {
	req.Header.Set("Accept", graphQLRequestContentType)
	req.Header.Set("Content-Type", graphQLRequestContentType)
	if source == nil {
		return
	}
	for key, value := range source.Headers {
		req.Header.Set(key, value)
	}
	if source.APIKey != "" && req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+source.APIKey)
	}
}

func readGraphQLResponseBody(body io.Reader, maxBytes int) ([]byte, error) {
	limited := io.LimitReader(body, int64(maxBytes+graphQLResponseBodyMargin))
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read graphql response: %w", err)
	}
	if len(data) > maxBytes {
		return nil, fmt.Errorf("graphql response exceeded max_response_bytes=%d", maxBytes)
	}
	return data, nil
}

func previewGraphQLBody(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return "empty body"
	}
	if len(trimmed) <= graphQLHTTPBodyPreviewLen {
		return trimmed
	}
	return trimmed[:graphQLHTTPBodyPreviewLen] + "..."
}
