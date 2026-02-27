package llm

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

type ClientOptions struct {
	Provider           Provider
	BaseURL            string
	APIKey             string
	Model              string
	ChatPath           string
	Headers            map[string]string
	AnthropicVersion   string
	AnthropicMaxTokens int
}

type Client struct {
	opts       ClientOptions
	httpClient *http.Client
}

func NewClient(baseURL, apiKey, model string) *Client {
	return NewClientWithOptions(ClientOptions{
		Provider: ProviderOpenAI,
		BaseURL:  baseURL,
		APIKey:   apiKey,
		Model:    model,
	})
}

func NewClientWithOptions(opts ClientOptions) *Client {
	normalized := normalizeOptions(opts)
	return &Client{
		opts: normalized,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *Client) Complete(ctx context.Context, messages []ChatMessage, tools []ToolDef) (*ChatResponse, error) {
	switch c.opts.Provider {
	case ProviderOpenAI, ProviderCustom:
		return c.completeOpenAICompatible(ctx, messages, tools)
	case ProviderAnthropic:
		return c.completeAnthropic(ctx, messages, tools)
	default:
		return nil, fmt.Errorf("unsupported provider %q", c.opts.Provider)
	}
}

func (c *Client) completeOpenAICompatible(ctx context.Context, messages []ChatMessage, tools []ToolDef) (*ChatResponse, error) {
	request := ChatRequest{
		Model:    c.opts.Model,
		Messages: messages,
		Tools:    tools,
	}

	headers := make(map[string]string, len(c.opts.Headers)+1)
	if c.opts.APIKey != "" {
		headers["Authorization"] = "Bearer " + c.opts.APIKey
	}
	mergeStringHeaders(headers, c.opts.Headers)

	raw, statusCode, err := c.postJSON(ctx, c.opts.ChatPath, request, headers)
	if err != nil {
		return nil, err
	}
	if err := ensureSuccessStatus(statusCode, raw); err != nil {
		return nil, err
	}

	var response ChatResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("decode chat response: %w", err)
	}
	return &response, nil
}

func (c *Client) postJSON(ctx context.Context, path string, requestBody any, headers map[string]string) ([]byte, int, error) {
	reqBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal request: %w", err)
	}

	url := c.opts.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	applyHeaders(req.Header, headers)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("read response body: %w", err)
	}

	return raw, resp.StatusCode, nil
}

func normalizeOptions(opts ClientOptions) ClientOptions {
	out := opts
	out.Provider = opts.Provider.Normalized()
	if out.Provider == "" {
		out.Provider = ProviderOpenAI
	}

	out.BaseURL = strings.TrimRight(strings.TrimSpace(out.BaseURL), "/")
	out.ChatPath = normalizePath(out.Provider, out.ChatPath)
	out.AnthropicVersion = strings.TrimSpace(out.AnthropicVersion)
	if out.AnthropicVersion == "" {
		out.AnthropicVersion = "2023-06-01"
	}
	if out.AnthropicMaxTokens <= 0 {
		out.AnthropicMaxTokens = 1024
	}

	out.Headers = cloneHeaders(opts.Headers)
	return out
}

func normalizePath(provider Provider, path string) string {
	p := strings.TrimSpace(path)
	if p == "" {
		switch provider {
		case ProviderAnthropic:
			p = "/v1/messages"
		default:
			p = "/chat/completions"
		}
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

func cloneHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	out := make(map[string]string, len(headers))
	for key, value := range headers {
		out[key] = value
	}
	return out
}

func mergeStringHeaders(dst map[string]string, src map[string]string) {
	for key, value := range src {
		k := strings.TrimSpace(key)
		if k == "" {
			continue
		}
		dst[k] = value
	}
}

func applyHeaders(dst http.Header, headers map[string]string) {
	for key, value := range headers {
		k := strings.TrimSpace(key)
		if k == "" {
			continue
		}
		dst.Set(k, value)
	}
}

func ensureSuccessStatus(statusCode int, raw []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	body := strings.TrimSpace(string(raw))
	if body == "" {
		body = "<empty body>"
	}
	return fmt.Errorf("chat completion failed with status %d: %s", statusCode, body)
}
