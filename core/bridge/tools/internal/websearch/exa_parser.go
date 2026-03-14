package websearch

import (
	"encoding/json"
	"strings"
)

const exaSnippetMaxChars = 280

type exaResponse struct {
	Results []exaResult `json:"results"`
}

type exaResult struct {
	Title      string   `json:"title"`
	URL        string   `json:"url"`
	Summary    string   `json:"summary"`
	Text       string   `json:"text"`
	Highlights []string `json:"highlights"`
}

func ParseExaJSON(source string, maxResults int) []Result {
	if maxResults <= 0 {
		return nil
	}

	var response exaResponse
	if err := json.Unmarshal([]byte(source), &response); err != nil {
		return nil
	}

	results := make([]Result, 0, maxResults)
	seen := make(map[string]struct{}, maxResults)
	for _, entry := range response.Results {
		linkURL := strings.TrimSpace(entry.URL)
		title := strings.TrimSpace(entry.Title)
		if linkURL == "" || title == "" {
			continue
		}
		if _, exists := seen[linkURL]; exists {
			continue
		}

		seen[linkURL] = struct{}{}
		results = append(results, Result{
			Title:   title,
			URL:     linkURL,
			Snippet: resolveExaSnippet(entry),
		})
		if len(results) >= maxResults {
			break
		}
	}
	return results
}

func resolveExaSnippet(result exaResult) string {
	if summary := compactExaText(result.Summary); summary != "" {
		return summary
	}
	for _, highlight := range result.Highlights {
		if snippet := compactExaText(highlight); snippet != "" {
			return snippet
		}
	}
	return compactExaText(result.Text)
}

func compactExaText(raw string) string {
	trimmed := strings.TrimSpace(spaceRegexp.ReplaceAllString(raw, " "))
	if len(trimmed) <= exaSnippetMaxChars {
		return trimmed
	}
	return strings.TrimSpace(trimmed[:exaSnippetMaxChars]) + "..."
}
