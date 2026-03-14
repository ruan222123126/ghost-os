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

func (t *GraphQLQueryTool) executeRequest(ctx context.Context, args graphQLQueryArgs) ([]byte, error) {
	if t == nil || t.httpClient == nil {
		return nil, fmt.Errorf("graphql http client is not configured")
	}
	if t.config.Endpoint == "" {
		return nil, fmt.Errorf("graphql endpoint is not configured")
	}

	requestCtx, cancel := graphQLRequestContext(ctx, t.config.TimeoutMS)
	defer cancel()

	req, err := t.buildRequest(requestCtx, args)
	if err != nil {
		return nil, err
	}
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("graphql request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := readGraphQLResponseBody(resp.Body, t.config.MaxResponseBytes)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("graphql request returned status %d: %s", resp.StatusCode, previewGraphQLBody(body))
	}
	return body, nil
}

func (t *GraphQLQueryTool) buildRequest(ctx context.Context, args graphQLQueryArgs) (*http.Request, error) {
	payload := graphQLRequestPayload{
		Query:         args.Query,
		Variables:     args.Variables,
		OperationName: strings.TrimSpace(args.OperationName),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode graphql request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.config.Endpoint, bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("build graphql request: %w", err)
	}
	applyGraphQLRequestHeaders(req, t.config)
	return req, nil
}

func graphQLRequestContext(ctx context.Context, timeoutMS int) (context.Context, context.CancelFunc) {
	timeout := time.Duration(timeoutMS) * time.Millisecond
	if timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, timeout)
}

func applyGraphQLRequestHeaders(req *http.Request, cfg GraphQLQueryConfig) {
	req.Header.Set("Accept", graphQLRequestContentType)
	req.Header.Set("Content-Type", graphQLRequestContentType)
	for key, value := range cfg.Headers {
		req.Header.Set(key, value)
	}
	if cfg.APIKey != "" && req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
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
