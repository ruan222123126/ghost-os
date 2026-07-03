package rss

import (
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/internal/stringutil"
)

func parseRSS(raw []byte, feedURL string) (Result, error) {
	var doc rssDocument
	if err := decodeXML(raw, &doc); err != nil {
		return Result{}, fmt.Errorf("decode rss feed: %w", err)
	}

	feed := FeedInfo{
		Title:       strings.TrimSpace(doc.Channel.Title),
		Link:        strings.TrimSpace(doc.Channel.Link),
		Description: cleanText(doc.Channel.Description),
		FeedURL:     strings.TrimSpace(feedURL),
	}

	items := make([]sortableItem, 0, len(doc.Channel.Items))
	seenIDs := make(map[string]struct{}, len(doc.Channel.Items))
	seenLinks := make(map[string]struct{}, len(doc.Channel.Items))
	for i, entry := range doc.Channel.Items {
		item := buildRSSItem(entry, feed)
		if shouldSkipItem(item, seenIDs, seenLinks) {
			continue
		}
		published, ok := parsePublishedTime(entry.PubDate, entry.Published, entry.Updated)
		if ok {
			item.PublishedAt = published.UTC().Format(time.RFC3339)
		}
		items = append(items, sortableItem{item: item, publishedTime: published, hasPublished: ok, index: i})
	}

	return Result{Feed: feed, Items: finalizeItems(items)}, nil
}

func buildRSSItem(entry rssItem, feed FeedInfo) Item {
	link := strings.TrimSpace(entry.Link)
	id := stringutil.FirstNonEmpty(strings.TrimSpace(entry.GUID), link)
	if id == "" {
		id = synthesizeID(strings.TrimSpace(entry.Title), link, entry.PubDate, entry.Published, entry.Updated)
	}
	return Item{
		ID:          id,
		Title:       strings.TrimSpace(entry.Title),
		Link:        link,
		Summary:     stringutil.FirstNonEmpty(cleanText(entry.Description), cleanText(entry.Summary), cleanText(entry.Content)),
		SourceTitle: feed.Title,
		SourceLink:  feed.Link,
	}
}
