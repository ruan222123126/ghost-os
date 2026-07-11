package websearch

import (
	"encoding/json"
	"strings"
)

type tavilyResponse struct {
	Results []tavilyResult `json:"results"`
}

type tavilyResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

// ParseTavilyJSON converts Tavily search results into the shared normalized shape.
func ParseTavilyJSON(source string, maxResults int) []Result {
	if maxResults <= 0 {
		return nil
	}

	var response tavilyResponse
	if err := json.Unmarshal([]byte(source), &response); err != nil {
		return nil
	}

	results := make([]Result, 0, min(maxResults, len(response.Results)))
	seen := make(map[string]struct{}, maxResults)
	for _, item := range response.Results {
		link := strings.TrimSpace(item.URL)
		title := strings.TrimSpace(item.Title)
		if link == "" || title == "" {
			continue
		}
		if _, exists := seen[link]; exists {
			continue
		}
		seen[link] = struct{}{}
		results = append(results, Result{
			Title:   title,
			URL:     link,
			Snippet: strings.TrimSpace(item.Content),
		})
		if len(results) >= maxResults {
			break
		}
	}
	return results
}
