package websearch

import (
	"encoding/xml"
	"strings"
)

type rssFeed struct {
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
}

// ParseRSS converts RSS search items into the shared normalized result shape.
func ParseRSS(source string, maxResults int) []Result {
	if maxResults <= 0 {
		return nil
	}

	var feed rssFeed
	if err := xml.Unmarshal([]byte(source), &feed); err != nil {
		return nil
	}

	results := make([]Result, 0, min(maxResults, len(feed.Channel.Items)))
	seen := make(map[string]struct{}, maxResults)
	for _, item := range feed.Channel.Items {
		link := strings.TrimSpace(item.Link)
		title := cleanHTMLText(item.Title)
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
			Snippet: cleanHTMLText(item.Description),
		})
		if len(results) >= maxResults {
			break
		}
	}
	return results
}
