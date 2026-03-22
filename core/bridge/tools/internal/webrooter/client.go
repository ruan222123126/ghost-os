package webrooter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	jsonContentType   = "application/json"
	authHeader        = "Authorization"
	bearerPrefix      = "Bearer "
	maxBodyBytes      = 16 << 20
	maxErrorBodyBytes = 64 << 10
)

func (c *Client) Execute(ctx context.Context, req Request) (Result, error) {
	if c == nil || c.httpClient == nil {
		return Result{}, fmt.Errorf("web_rooter http client is not configured")
	}
	if err := c.validatePinnedVersion(ctx, req.TraceID); err != nil {
		return Result{}, err
	}
	return c.executeAction(ctx, req)
}

func (c *Client) validatePinnedVersion(ctx context.Context, traceID string) error {
	var payload versionResponse
	if err := c.doRequest(ctx, http.MethodGet, "/", nil, traceID, &payload); err != nil {
		return fmt.Errorf("validate web_rooter version: %w", err)
	}
	if strings.TrimSpace(payload.Version) == PinnedVersion {
		return nil
	}

	return fmt.Errorf(
		"unexpected web-rooter version %q; bridge integration is pinned to v%s (source commit %s)",
		strings.TrimSpace(payload.Version),
		PinnedVersion,
		PinnedCommit,
	)
}

func (c *Client) executeAction(ctx context.Context, req Request) (Result, error) {
	var payload map[string]any
	if err := c.doRequest(ctx, http.MethodPost, req.Path, req.Body, req.TraceID, &payload); err != nil {
		return Result{}, fmt.Errorf("web_rooter %s request failed: %w", req.Action, err)
	}
	if payload == nil {
		return Result{}, fmt.Errorf("web_rooter %s returned empty JSON object", req.Action)
	}
	return NormalizeResponse(req.Action, payload)
}

func (c *Client) doRequest(
	ctx context.Context,
	method string,
	path string,
	body any,
	traceID string,
	target any,
) error {
	request, err := c.newRequest(ctx, method, path, body, traceID)
	if err != nil {
		return err
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("status %d: %s", response.StatusCode, readErrorBody(response.Body))
	}
	return decodeJSONResponse(response.Body, target)
}

func (c *Client) newRequest(
	ctx context.Context,
	method string,
	path string,
	body any,
	traceID string,
) (*http.Request, error) {
	endpoint, err := joinURL(c.baseURL, path)
	if err != nil {
		return nil, err
	}

	reader, err := encodeBody(body)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	request.Header.Set("Accept", jsonContentType)
	if body != nil {
		request.Header.Set("Content-Type", jsonContentType)
	}
	if c.apiToken != "" {
		request.Header.Set(authHeader, bearerPrefix+c.apiToken)
	}
	if trimmed := strings.TrimSpace(traceID); trimmed != "" {
		request.Header.Set("X-Trace-ID", trimmed)
	}
	return request, nil
}

func encodeBody(body any) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode request body: %w", err)
	}
	return bytes.NewReader(encoded), nil
}

func joinURL(baseURL string, path string) (string, error) {
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", fmt.Errorf("invalid web_rooter base url: %w", err)
	}
	if base.Scheme == "" || base.Host == "" {
		return "", fmt.Errorf("invalid web_rooter base url %q", baseURL)
	}

	ref, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("invalid web_rooter path %q: %w", path, err)
	}
	return base.ResolveReference(ref).String(), nil
}

func readErrorBody(body io.Reader) string {
	data, err := io.ReadAll(io.LimitReader(body, maxErrorBodyBytes))
	if err != nil {
		return "unable to read error body"
	}

	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return "empty error body"
	}
	return trimmed
}

func decodeJSONResponse(body io.Reader, target any) error {
	data, err := io.ReadAll(io.LimitReader(body, maxBodyBytes))
	if err != nil {
		return fmt.Errorf("decode response: read response: %w", err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return fmt.Errorf("decode response: empty response body")
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return ensureJSONEOF(decoder)
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return fmt.Errorf("decode response: unexpected trailing JSON content")
}
