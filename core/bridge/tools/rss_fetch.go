package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	internalrss "ghost-os/bridge/tools/internal/rss"
)

type RSSResult = internalrss.Result
type RSSFeedInfo = internalrss.FeedInfo
type RSSItem = internalrss.Item

type RSSFetchOptions struct {
	MaxItems       int
	IncludeSummary bool
}

const (
	defaultRSSFetchTimeout   = 10 * time.Second
	defaultRSSBodyLimitBytes = 2 << 20
	defaultRSSMaxItems       = 10
	maxRSSMaxItems           = 50
	maxRSSRedirects          = 5
)

type RSSFetchTool struct {
	httpClient  *http.Client
	validateURL func(context.Context, *url.URL) error
	now         func() time.Time
	bodyLimit   int64
}

type rssFetchArgs struct {
	URL            string `json:"url"`
	MaxItems       int    `json:"max_items,omitempty"`
	IncludeSummary *bool  `json:"include_summary,omitempty"`
}

func NewRSSFetchTool() Tool {
	return newDefaultRSSFetchTool()
}

func FetchRSS(ctx context.Context, rawURL string, opts RSSFetchOptions) (RSSResult, error) {
	tool := newDefaultRSSFetchTool()
	feedURLText := strings.TrimSpace(rawURL)
	if feedURLText == "" {
		return RSSResult{}, fmt.Errorf("url is required")
	}
	feedURL, err := url.Parse(feedURLText)
	if err != nil {
		return RSSResult{}, fmt.Errorf("invalid url: %w", err)
	}
	if err := tool.validateRequestURL(ctx, feedURL); err != nil {
		return RSSResult{}, err
	}
	maxItems := opts.MaxItems
	if maxItems <= 0 {
		maxItems = defaultRSSMaxItems
	}
	if maxItems > maxRSSMaxItems {
		maxItems = maxRSSMaxItems
	}
	result, err := tool.fetch(ctx, feedURL)
	if err != nil {
		return RSSResult{}, err
	}
	if len(result.Items) > maxItems {
		result.Items = result.Items[:maxItems]
	}
	if !opts.IncludeSummary {
		for i := range result.Items {
			result.Items[i].Summary = ""
		}
	}
	return result, nil
}

func (RSSFetchTool) Name() string {
	return "rss_fetch"
}

func (RSSFetchTool) Description() string {
	return "Fetch an HTTPS RSS or Atom feed and return normalized items sorted by newest first."
}

func (RSSFetchTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"url":{"type":"string","description":"HTTPS RSS or Atom feed URL."},
			"max_items":{"type":"integer","minimum":1,"maximum":50,"description":"Maximum number of items to return (default: 10)."},
			"include_summary":{"type":"boolean","description":"Whether to include cleaned item summaries (default: true)."}
		},
		"required":["url"],
		"additionalProperties":false
	}`)
}

func (t *RSSFetchTool) Execute(ctx context.Context, argsJSON json.RawMessage, _ string) (string, error) {
	if t == nil || t.httpClient == nil {
		return "", fmt.Errorf("http client is not configured")
	}

	var args rssFetchArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}

	feedURLText := strings.TrimSpace(args.URL)
	if feedURLText == "" {
		return "", fmt.Errorf("url is required")
	}
	feedURL, err := url.Parse(feedURLText)
	if err != nil {
		return "", fmt.Errorf("invalid url: %w", err)
	}
	if err := t.validateRequestURL(ctx, feedURL); err != nil {
		return "", err
	}

	maxItems := args.MaxItems
	if maxItems <= 0 {
		maxItems = defaultRSSMaxItems
	}
	if maxItems > maxRSSMaxItems {
		maxItems = maxRSSMaxItems
	}

	includeSummary := true
	if args.IncludeSummary != nil {
		includeSummary = *args.IncludeSummary
	}

	result, err := t.fetch(ctx, feedURL)
	if err != nil {
		return "", err
	}
	if len(result.Items) > maxItems {
		result.Items = result.Items[:maxItems]
	}
	if !includeSummary {
		for i := range result.Items {
			result.Items[i].Summary = ""
		}
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("encode feed result: %w", err)
	}
	return string(encoded), nil
}

func (t *RSSFetchTool) fetch(ctx context.Context, feedURL *url.URL) (internalrss.Result, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL.String(), nil)
	if err != nil {
		return internalrss.Result{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "Ghost-OS/1.0 (+https://ghost-os.dev)")
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml;q=0.9, */*;q=0.8")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return internalrss.Result{}, fmt.Errorf("rss request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return internalrss.Result{}, fmt.Errorf("rss request returned status %d", resp.StatusCode)
	}

	bodyLimit := t.bodyLimit
	if bodyLimit <= 0 {
		bodyLimit = defaultRSSBodyLimitBytes
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, bodyLimit))
	if err != nil {
		return internalrss.Result{}, fmt.Errorf("read rss response: %w", err)
	}

	now := time.Now
	if t.now != nil {
		now = t.now
	}
	result, err := internalrss.Parse(body, feedURL.String(), now())
	if err != nil {
		return internalrss.Result{}, fmt.Errorf("parse rss feed: %w", err)
	}
	return result, nil
}

func (t *RSSFetchTool) validateRequestURL(ctx context.Context, rawURL *url.URL) error {
	validator := validateRSSURL
	if t != nil && t.validateURL != nil {
		validator = t.validateURL
	}
	return validator(ctx, rawURL)
}

func validateRSSURL(ctx context.Context, rawURL *url.URL) error {
	if rawURL == nil {
		return fmt.Errorf("url is required")
	}
	if !strings.EqualFold(strings.TrimSpace(rawURL.Scheme), "https") {
		return fmt.Errorf("url must use https")
	}
	host := strings.TrimSpace(rawURL.Hostname())
	if host == "" {
		return fmt.Errorf("url host is required")
	}
	if err := validateRSSHost(ctx, host); err != nil {
		return err
	}
	return nil
}

func validateRSSHost(ctx context.Context, host string) error {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return fmt.Errorf("local feed hosts are not allowed")
	}
	if ip := net.ParseIP(host); ip != nil {
		if !isAllowedRSSIP(ip) {
			return fmt.Errorf("feed host resolves to a disallowed address")
		}
		return nil
	}

	resolver := net.DefaultResolver
	addrs, err := resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("resolve feed host: %w", err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("resolve feed host: no addresses found")
	}
	for _, addr := range addrs {
		if !isAllowedRSSIP(addr.IP) {
			return fmt.Errorf("feed host resolves to a disallowed address")
		}
	}
	return nil
}

func isAllowedRSSIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return false
	}
	return true
}
