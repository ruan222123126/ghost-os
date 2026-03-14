package rss

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Result struct {
	Feed      FeedInfo `json:"feed"`
	Items     []Item   `json:"items"`
	FetchedAt string   `json:"fetched_at"`
}

type FeedInfo struct {
	Title       string `json:"title,omitempty"`
	Link        string `json:"link,omitempty"`
	Description string `json:"description,omitempty"`
	FeedURL     string `json:"feed_url,omitempty"`
}

type Item struct {
	ID          string `json:"id,omitempty"`
	Title       string `json:"title,omitempty"`
	Link        string `json:"link,omitempty"`
	Summary     string `json:"summary,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
	SourceTitle string `json:"source_title,omitempty"`
	SourceLink  string `json:"source_link,omitempty"`
}

type rssDocument struct {
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	GUID        string `xml:"guid"`
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Summary     string `xml:"summary"`
	Content     string `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
	PubDate     string `xml:"pubDate"`
	Published   string `xml:"published"`
	Updated     string `xml:"updated"`
}

type atomDocument struct {
	Title    string      `xml:"title"`
	Subtitle string      `xml:"subtitle"`
	Links    []atomLink  `xml:"link"`
	Entries  []atomEntry `xml:"entry"`
}

type atomEntry struct {
	ID        string     `xml:"id"`
	Title     string     `xml:"title"`
	Summary   string     `xml:"summary"`
	Content   string     `xml:"content"`
	Published string     `xml:"published"`
	Updated   string     `xml:"updated"`
	Links     []atomLink `xml:"link"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type sortableItem struct {
	item          Item
	publishedTime time.Time
	hasPublished  bool
	index         int
}

var htmlTagPattern = regexp.MustCompile(`(?s)<[^>]+>`)

func Parse(raw []byte, feedURL string, fetchedAt time.Time) (Result, error) {
	var root struct {
		XMLName xml.Name
	}
	if err := decodeXML(raw, &root); err != nil {
		return Result{}, fmt.Errorf("decode feed root: %w", err)
	}

	var result Result
	switch strings.ToLower(strings.TrimSpace(root.XMLName.Local)) {
	case "rss":
		parsed, err := parseRSS(raw, feedURL)
		if err != nil {
			return Result{}, err
		}
		result = parsed
	case "feed":
		parsed, err := parseAtom(raw, feedURL)
		if err != nil {
			return Result{}, err
		}
		result = parsed
	default:
		return Result{}, fmt.Errorf("unsupported feed root %q", root.XMLName.Local)
	}

	result.FetchedAt = fetchedAt.UTC().Format(time.RFC3339)
	return result, nil
}

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

func buildRSSItem(entry rssItem, feed FeedInfo) Item {
	link := strings.TrimSpace(entry.Link)
	id := firstNonEmpty(strings.TrimSpace(entry.GUID), link)
	if id == "" {
		id = synthesizeID(strings.TrimSpace(entry.Title), link, entry.PubDate, entry.Published, entry.Updated)
	}
	return Item{
		ID:          id,
		Title:       strings.TrimSpace(entry.Title),
		Link:        link,
		Summary:     firstNonEmpty(cleanText(entry.Description), cleanText(entry.Summary), cleanText(entry.Content)),
		SourceTitle: feed.Title,
		SourceLink:  feed.Link,
	}
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

func shouldSkipItem(item Item, seenIDs map[string]struct{}, seenLinks map[string]struct{}) bool {
	if id := strings.TrimSpace(item.ID); id != "" {
		if _, exists := seenIDs[id]; exists {
			return true
		}
		seenIDs[id] = struct{}{}
	}
	if link := strings.TrimSpace(item.Link); link != "" {
		if _, exists := seenLinks[link]; exists {
			return true
		}
		seenLinks[link] = struct{}{}
	}
	return false
}

func finalizeItems(items []sortableItem) []Item {
	sort.SliceStable(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		if left.hasPublished && right.hasPublished {
			if left.publishedTime.Equal(right.publishedTime) {
				return left.index < right.index
			}
			return left.publishedTime.After(right.publishedTime)
		}
		if left.hasPublished != right.hasPublished {
			return left.hasPublished
		}
		return left.index < right.index
	})

	out := make([]Item, 0, len(items))
	for _, entry := range items {
		out = append(out, entry.item)
	}
	return out
}

func decodeXML(raw []byte, target any) error {
	decoder := xml.NewDecoder(bytes.NewReader(raw))
	decoder.Strict = false
	return decoder.Decode(target)
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

func cleanText(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	trimmed = html.UnescapeString(trimmed)
	trimmed = htmlTagPattern.ReplaceAllString(trimmed, " ")
	return strings.Join(strings.Fields(trimmed), " ")
}

func parsePublishedTime(values ...string) (time.Time, bool) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		time.RFC850,
		"Mon, 02 Jan 2006 15:04:05 MST",
		"Mon, 2 Jan 2006 15:04:05 MST",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05Z0700",
	}
	for _, raw := range values {
		text := strings.TrimSpace(raw)
		if text == "" {
			continue
		}
		for _, layout := range layouts {
			parsed, err := time.Parse(layout, text)
			if err == nil {
				return parsed, true
			}
		}
	}
	return time.Time{}, false
}

func synthesizeID(values ...string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, " | ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
