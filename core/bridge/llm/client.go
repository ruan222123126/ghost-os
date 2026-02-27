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

// ClientOptions 定义 provider 与 HTTP 客户端初始化所需配置。
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

// Client 封装 provider 适配与 HTTP 调用细节。
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

// NewClientWithOptions 创建带默认值归一化的客户端实例。
func NewClientWithOptions(opts ClientOptions) *Client {
	normalized := normalizeOptions(opts)
	return &Client{
		opts: normalized,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// providerRequest 是发送到具体 provider 前的统一中间结构。
type providerRequest struct {
	path    string
	body    any
	headers map[string]string
}

// Complete 执行一次完整请求链路：构建请求 -> 发起 HTTP -> 解析响应。
func (c *Client) Complete(ctx context.Context, request CompletionRequest) (*CompletionResponse, error) {
	payload, err := c.buildProviderRequest(request)
	if err != nil {
		return nil, err
	}

	raw, statusCode, err := c.postJSON(ctx, payload.path, payload.body, payload.headers)
	if err != nil {
		return nil, err
	}
	if err := ensureSuccessStatus(statusCode, raw); err != nil {
		return nil, err
	}

	return c.parseProviderResponse(raw)
}

// buildProviderRequest 按 provider 选择协议编码。
func (c *Client) buildProviderRequest(request CompletionRequest) (providerRequest, error) {
	switch c.opts.Provider {
	case ProviderOpenAI, ProviderCustom:
		return c.buildOpenAIProviderRequest(request)
	case ProviderAnthropic:
		return c.buildAnthropicProviderRequest(request)
	default:
		return providerRequest{}, fmt.Errorf("unsupported provider %q", c.opts.Provider)
	}
}

// parseProviderResponse 按 provider 选择响应解码。
func (c *Client) parseProviderResponse(raw []byte) (*CompletionResponse, error) {
	switch c.opts.Provider {
	case ProviderOpenAI, ProviderCustom:
		return c.parseOpenAIProviderResponse(raw)
	case ProviderAnthropic:
		return c.parseAnthropicProviderResponse(raw)
	default:
		return nil, fmt.Errorf("unsupported provider %q", c.opts.Provider)
	}
}

// postJSON 负责底层 HTTP POST 与原始响应读取。
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

// normalizeOptions 统一默认值与路径/头部格式。
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

// normalizePath 在未配置时填充 provider 默认 API 路径。
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

// mergeStringHeaders 合并用户自定义 Header，并忽略空 key。
func mergeStringHeaders(dst map[string]string, src map[string]string) {
	for key, value := range src {
		k := strings.TrimSpace(key)
		if k == "" {
			continue
		}
		dst[k] = value
	}
}

// applyHeaders 将 map 形式 Header 应用到 http.Request。
func applyHeaders(dst http.Header, headers map[string]string) {
	for key, value := range headers {
		k := strings.TrimSpace(key)
		if k == "" {
			continue
		}
		dst.Set(k, value)
	}
}

// ensureSuccessStatus 在非 2xx 时返回包含响应体的错误。
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
