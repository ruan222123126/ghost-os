package rss

import (
	"fmt"
	"strings"
	"time"
)

func parseAtom(raw []byte, feedURL string) (Result, error) {
	var doc atomDocument
	if err := decodeXML(raw, &doc); err != nil {
		return Result{}, fmt.Errorf("decode atom feed: %w", err)
	}

	feed := FeedInfo{
		Title:       strings.TrimSpace(doc.Title),
		Link:        pickAtomLink(doc.Links),
		Description: cleanText(doc.Subtitle),
		FeedURL:     strings.TrimSpace(feedURL),
	}

	items := make([]sortableItem, 0, len(doc.Entries))
	seenIDs := make(map[string]struct{}, len(doc.Entries))
	seenLinks := make(map[string]struct{}, len(doc.Entries))
	for i, entry := range doc.Entries {
		item := buildAtomItem(entry, feed)
		if shouldSkipItem(item, seenIDs, seenLinks) {
			continue
		}
		published, ok := parsePublishedTime(entry.Published, entry.Updated)
		if ok {
			item.PublishedAt = published.UTC().Format(time.RFC3339)
		}
		items = append(items, sortableItem{item: item, publishedTime: published, hasPublished: ok, index: i})
	}

	return Result{Feed: feed, Items: finalizeItems(items)}, nil
}

func buildAtomItem(entry atomEntry, feed FeedInfo) Item {
	link := pickAtomLink(entry.Links)
	id := firstNonEmpty(strings.TrimSpace(entry.ID), link)
	if id == "" {
		id = synthesizeID(strings.TrimSpace(entry.Title), link, entry.Published, entry.Updated)
	}
	return Item{
		ID:          id,
		Title:       strings.TrimSpace(entry.Title),
		Link:        link,
		Summary:     firstNonEmpty(cleanText(entry.Summary), cleanText(entry.Content)),
		SourceTitle: feed.Title,
		SourceLink:  feed.Link,
	}
}

func pickAtomLink(links []atomLink) string {
	for _, link := range links {
		href := strings.TrimSpace(link.Href)
		rel := strings.ToLower(strings.TrimSpace(link.Rel))
		if href != "" && (rel == "" || rel == "alternate") {
			return href
		}
	}
	for _, link := range links {
		if href := strings.TrimSpace(link.Href); href != "" {
			return href
		}
	}
	return ""
}
