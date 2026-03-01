// Web search tool implementation and result normalization for agent consumption.

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	defaultWebSearchResults = 5
	maxWebSearchResults     = 10
	maxWebSearchBodyBytes   = 2 << 20
)

var (
	resultAnchorRegexp = regexp.MustCompile(`(?is)<a[^>]*class="[^"]*result__a[^"]*"[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	resultSnippetRegex = regexp.MustCompile(`(?is)<a[^>]*class="[^"]*result__snippet[^"]*"[^>]*>(.*?)</a>|<div[^>]*class="[^"]*result__snippet[^"]*"[^>]*>(.*?)</div>`)
	htmlTagRegexp      = regexp.MustCompile(`(?is)<[^>]+>`)
	spaceRegexp        = regexp.MustCompile(`\s+`)
)

type WebSearchTool struct {
	httpClient *http.Client
	endpoint   string
	userAgents []string
}

type webSearchArgs struct {
	Query      string `json:"query"`
	MaxResults int    `json:"max_results,omitempty"`
}

type webSearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// NewWebSearchTool 创建默认的网页搜索工具实例（DuckDuckGo HTML 端点）。
func NewWebSearchTool() Tool {
	return &WebSearchTool{
		httpClient: &http.Client{Timeout: 12 * time.Second},
		endpoint:   "https://html.duckduckgo.com/html/",
		userAgents: []string{
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Safari/605.1.15",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:124.0) Gecko/20100101 Firefox/124.0",
		},
	}
}

func (WebSearchTool) Name() string {
	return "web_search"
}

func (WebSearchTool) Description() string {
	return "Search the web for current information and return title, URL, and snippet results."
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
func (t *WebSearchTool) search(ctx context.Context, query string, maxResults int) ([]webSearchResult, error) {
	endpoint := strings.TrimSpace(t.endpoint)
	if endpoint == "" {
		endpoint = "https://html.duckduckgo.com/html/"
	}

	requestURL := endpoint
	if strings.Contains(requestURL, "?") {
		requestURL += "&q=" + url.QueryEscape(query)
	} else {
		requestURL += "?q=" + url.QueryEscape(query)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", t.pickUserAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "en-US,en;q=0.8")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("search request returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxWebSearchBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("read search response: %w", err)
	}

	return parseDuckDuckGoHTML(string(bodyBytes), maxResults), nil
}

// pickUserAgent 从候选列表轮换 UA，降低被动限流概率。
func (t *WebSearchTool) pickUserAgent() string {
	if len(t.userAgents) == 0 {
		return "Ghost-OS/1.0 (+https://ghost-os.dev)"
	}
	index := int(time.Now().UnixNano() % int64(len(t.userAgents)))
	if index < 0 || index >= len(t.userAgents) {
		index = 0
	}
	return t.userAgents[index]
}

// parseDuckDuckGoHTML 从结果页提取标题/链接/摘要并做 URL 去重。
func parseDuckDuckGoHTML(source string, maxResults int) []webSearchResult {
	if maxResults <= 0 {
		return nil
	}

	matches := resultAnchorRegexp.FindAllStringSubmatchIndex(source, maxResults*4)
	results := make([]webSearchResult, 0, maxResults)
	seen := make(map[string]struct{}, maxResults)

	for _, loc := range matches {
		if len(loc) < 6 {
			continue
		}

		rawURL := source[loc[2]:loc[3]]
		titleHTML := source[loc[4]:loc[5]]
		linkURL := normalizeResultURL(rawURL)
		if linkURL == "" {
			continue
		}
		if _, exists := seen[linkURL]; exists {
			continue
		}

		anchorEnd := loc[1]
		snippetWindowEnd := anchorEnd + 1200
		if snippetWindowEnd > len(source) {
			snippetWindowEnd = len(source)
		}
		snippet := extractSnippet(source[anchorEnd:snippetWindowEnd])

		result := webSearchResult{
			Title:   cleanHTMLText(titleHTML),
			URL:     linkURL,
			Snippet: snippet,
		}
		if result.Title == "" {
			continue
		}

		seen[linkURL] = struct{}{}
		results = append(results, result)
		if len(results) >= maxResults {
			break
		}
	}

	return results
}

// extractSnippet 从结果锚点附近片段中提取可读摘要文本。
func extractSnippet(source string) string {
	match := resultSnippetRegex.FindStringSubmatch(source)
	if len(match) < 2 {
		return ""
	}
	if snippet := cleanHTMLText(match[1]); snippet != "" {
		return snippet
	}
	if len(match) >= 3 {
		return cleanHTMLText(match[2])
	}
	return ""
}

// normalizeResultURL 还原跳转链接中的真实目标 URL，并过滤无效 scheme。
func normalizeResultURL(raw string) string {
	unescaped := html.UnescapeString(strings.TrimSpace(raw))
	if unescaped == "" {
		return ""
	}
	if strings.HasPrefix(unescaped, "//") {
		unescaped = "https:" + unescaped
	}

	parsed, err := url.Parse(unescaped)
	if err != nil {
		return ""
	}
	queryURL := parsed.Query().Get("uddg")
	if queryURL != "" {
		decoded, decodeErr := url.QueryUnescape(queryURL)
		if decodeErr == nil && strings.TrimSpace(decoded) != "" {
			return strings.TrimSpace(decoded)
		}
		return strings.TrimSpace(queryURL)
	}
	if parsed.Scheme == "" {
		return ""
	}
	return parsed.String()
}

// cleanHTMLText 去除标签与多余空白，得到可读纯文本。
func cleanHTMLText(raw string) string {
	withoutTags := htmlTagRegexp.ReplaceAllString(raw, " ")
	unescaped := html.UnescapeString(withoutTags)
	return strings.TrimSpace(spaceRegexp.ReplaceAllString(unescaped, " "))
}
