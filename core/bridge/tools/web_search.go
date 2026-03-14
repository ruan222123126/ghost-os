// Web search tool implementation and result normalization for agent consumption.

package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ghost-os/bridge/tools/internal/websearch"
)

const (
	defaultWebSearchEndpoint = "https://html.duckduckgo.com/html/"
	defaultWebSearchBingRSS  = "https://www.bing.com/search"
	defaultWebSearchTavily   = "https://api.tavily.com/search"
	defaultWebSearchResults  = 5
	maxWebSearchResults      = 10
	maxWebSearchBodyBytes    = 2 << 20
	maxWebSearchAttemptDelay = 4 * time.Second
)

type webSearchFormat string

const (
	webSearchFormatDuckDuckGoHTML webSearchFormat = "duckduckgo_html"
	webSearchFormatBingRSS        webSearchFormat = "bing_rss"
	webSearchFormatTavilyJSON     webSearchFormat = "tavily_json"
)

type webSearchProvider struct {
	name     string
	endpoint string
	format   webSearchFormat
	apiKey   string
}

type WebSearchConfig struct {
	TavilyAPIKey string
}

type WebSearchTool struct {
	httpClient *http.Client
	endpoint   string
	userAgents []string
	providers  []webSearchProvider
	config     WebSearchConfig
}

type webSearchArgs struct {
	Query      string `json:"query"`
	MaxResults int    `json:"max_results,omitempty"`
}

// NewWebSearchTool 创建默认的网页搜索工具实例。
func NewWebSearchTool(cfg WebSearchConfig) Tool {
	tool := &WebSearchTool{
		httpClient: &http.Client{Timeout: 12 * time.Second},
		userAgents: []string{
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Safari/605.1.15",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:124.0) Gecko/20100101 Firefox/124.0",
		},
		config: cfg,
	}
	tool.providers = tool.defaultProviders()
	return tool
}

func (WebSearchTool) Name() string {
	return "web_search"
}

func (WebSearchTool) Description() string {
	return "Search the web for current information."
}

func (WebSearchTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"query":{"type":"string","description":"Search query string."},
			"max_results":{"type":"integer","minimum":1,"maximum":10,"description":"Maximum number of search results to return (default: 5)."}
		},
		"required":["query"],
		"additionalProperties":false
	}`)
}

// Execute 校验查询参数，执行搜索并以结构化 JSON 结果返回。
func (t *WebSearchTool) Execute(ctx context.Context, argsJSON json.RawMessage, _ string) (string, error) {
	if t.httpClient == nil {
		return "", fmt.Errorf("http client is not configured")
	}

	var args webSearchArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}

	query := strings.TrimSpace(args.Query)
	if query == "" {
		return "", fmt.Errorf("query is required")
	}

	maxResults := args.MaxResults
	if maxResults <= 0 {
		maxResults = defaultWebSearchResults
	}
	if maxResults > maxWebSearchResults {
		maxResults = maxWebSearchResults
	}

	results, err := t.search(ctx, query, maxResults)
	if err != nil {
		return "", err
	}

	encoded, err := json.Marshal(results)
	if err != nil {
		return "", fmt.Errorf("encode search results: %w", err)
	}
	return string(encoded), nil
}

// search 请求搜索端点并解析 HTML，返回去重后的结果列表。
func (t *WebSearchTool) search(ctx context.Context, query string, maxResults int) ([]websearch.Result, error) {
	providers := t.providersForSearch()
	errors := make([]string, 0, len(providers))

	for index, provider := range providers {
		results, err := t.searchProvider(ctx, provider, query, maxResults, len(providers)-index)
		if err == nil {
			return results, nil
		}
		errors = append(errors, fmt.Sprintf("%s: %v", provider.name, err))
		if ctx.Err() != nil {
			break
		}
	}

	if len(errors) == 0 {
		return nil, fmt.Errorf("no search providers configured")
	}
	return nil, fmt.Errorf("web search failed after %d provider(s): %s", len(errors), strings.Join(errors, "; "))
}

func (t *WebSearchTool) providersForSearch() []webSearchProvider {
	if len(t.providers) > 0 {
		return t.providers
	}
	endpoint := strings.TrimSpace(t.endpoint)
	if endpoint == "" {
		return t.defaultProviders()
	}
	return []webSearchProvider{{name: "duckduckgo_html", endpoint: endpoint, format: webSearchFormatDuckDuckGoHTML}}
}

func (t *WebSearchTool) defaultProviders() []webSearchProvider {
	if apiKey := strings.TrimSpace(t.config.TavilyAPIKey); apiKey != "" {
		return []webSearchProvider{{
			name:     "tavily",
			endpoint: defaultWebSearchTavily,
			format:   webSearchFormatTavilyJSON,
			apiKey:   apiKey,
		}}
	}
	return []webSearchProvider{
		{name: "duckduckgo_html", endpoint: defaultWebSearchEndpoint, format: webSearchFormatDuckDuckGoHTML},
		{name: "bing_rss", endpoint: defaultWebSearchBingRSS, format: webSearchFormatBingRSS},
	}
}

func (t *WebSearchTool) searchProvider(ctx context.Context, provider webSearchProvider, query string, maxResults int, remainingProviders int) ([]websearch.Result, error) {
	requestURL, err := url.Parse(strings.TrimSpace(provider.endpoint))
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint: %w", err)
	}

	requestCtx := ctx
	cancel := func() {}
	if timeout := t.providerTimeout(ctx, remainingProviders); timeout > 0 {
		requestCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	req, err := buildWebSearchRequest(requestCtx, provider, requestURL, query, maxResults)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", t.pickUserAgent())
	req.Header.Set("Accept-Language", "en-US,en;q=0.8")
	req.Header.Set("Accept", acceptHeaderForProvider(provider.format))

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("request returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxWebSearchBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	results := parseProviderResults(provider.format, string(bodyBytes), maxResults)
	if len(results) == 0 {
		return nil, fmt.Errorf("provider returned no parseable results")
	}
	return results, nil
}

func (t *WebSearchTool) providerTimeout(ctx context.Context, remainingProviders int) time.Duration {
	budget := 12 * time.Second
	if t.httpClient != nil && t.httpClient.Timeout > 0 {
		budget = t.httpClient.Timeout
	}
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return 0
		}
		if remaining < budget {
			budget = remaining
		}
	}
	if remainingProviders <= 1 || budget <= maxWebSearchAttemptDelay {
		return budget
	}
	perProvider := budget / time.Duration(remainingProviders)
	if perProvider <= 0 {
		return budget
	}
	if perProvider > maxWebSearchAttemptDelay {
		return maxWebSearchAttemptDelay
	}
	return perProvider
}

func acceptHeaderForProvider(format webSearchFormat) string {
	if format == webSearchFormatTavilyJSON {
		return "application/json"
	}
	if format == webSearchFormatBingRSS {
		return "application/rss+xml,application/xml,text/xml;q=0.9,*/*;q=0.8"
	}
	return "text/html,application/xhtml+xml"
}

func parseProviderResults(format webSearchFormat, body string, maxResults int) []websearch.Result {
	switch format {
	case webSearchFormatTavilyJSON:
		return websearch.ParseTavilyJSON(body, maxResults)
	case webSearchFormatBingRSS:
		return websearch.ParseRSS(body, maxResults)
	default:
		return websearch.ParseHTML(body, maxResults)
	}
}

func buildWebSearchRequest(ctx context.Context, provider webSearchProvider, requestURL *url.URL, query string, maxResults int) (*http.Request, error) {
	if provider.format == webSearchFormatTavilyJSON {
		payload := map[string]any{
			"query":        query,
			"topic":        "general",
			"search_depth": "basic",
			"max_results":  maxResults,
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL.String(), bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		if apiKey := strings.TrimSpace(provider.apiKey); apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}
		return req, nil
	}

	requestQuery := requestURL.Query()
	requestQuery.Set("q", query)
	if provider.format == webSearchFormatBingRSS {
		requestQuery.Set("format", "rss")
	}
	requestURL.RawQuery = requestQuery.Encode()
	return http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
}

// pickUserAgent 从候选列表轮换 UA，降低被动限流概率。
func (t *WebSearchTool) pickUserAgent() string {
	if len(t.userAgents) == 0 {
		return "Ghost-OS/1.0 (+https://ghost-os.dev)"
	}
	return t.userAgents[int(time.Now().UnixNano()%int64(len(t.userAgents)))]
}
