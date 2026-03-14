package websearch

import (
	"html"
	"net/url"
	"regexp"
	"strings"
)

const snippetWindowBytes = 1200

var (
	resultAnchorRegexp = regexp.MustCompile(`(?is)<a[^>]*class="[^"]*result__a[^"]*"[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	resultSnippetRegex = regexp.MustCompile(`(?is)<a[^>]*class="[^"]*result__snippet[^"]*"[^>]*>(.*?)</a>|<div[^>]*class="[^"]*result__snippet[^"]*"[^>]*>(.*?)</div>`)
	htmlTagRegexp      = regexp.MustCompile(`(?is)<[^>]+>`)
	spaceRegexp        = regexp.MustCompile(`\s+`)
)

// Result 描述单条网页搜索结果。
type Result struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// ParseHTML 从 DuckDuckGo HTML 结果页提取标题、链接和摘要。
func ParseHTML(source string, maxResults int) []Result {
	if maxResults <= 0 {
		return nil
	}

	matches := resultAnchorRegexp.FindAllStringSubmatchIndex(source, maxResults*4)
	results := make([]Result, 0, maxResults)
	seen := make(map[string]struct{}, maxResults)

	for _, loc := range matches {
		if len(loc) < 6 {
			continue
		}

		linkURL := normalizeResultURL(source[loc[2]:loc[3]])
		if linkURL == "" {
			continue
		}
		if _, exists := seen[linkURL]; exists {
			continue
		}

		anchorEnd := loc[1]
		snippetWindowEnd := anchorEnd + snippetWindowBytes
		if snippetWindowEnd > len(source) {
			snippetWindowEnd = len(source)
		}

		result := Result{
			Title:   cleanHTMLText(source[loc[4]:loc[5]]),
			URL:     linkURL,
			Snippet: extractSnippet(source[anchorEnd:snippetWindowEnd]),
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

func cleanHTMLText(raw string) string {
	withoutTags := htmlTagRegexp.ReplaceAllString(raw, " ")
	unescaped := html.UnescapeString(withoutTags)
	return strings.TrimSpace(spaceRegexp.ReplaceAllString(unescaped, " "))
}
