package rss

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	rsssubscriptions "ghost-os/bridge/rss/subscriptions"
)

const (
	defaultRSSFetchMaxItems    = 10
	maxRSSClassifierSummaryLen = 280
)

type rssInboxCandidate struct {
	FeedID       string
	FeedURL      string
	SourceTitle  string
	FeedTags     []string
	ItemTitle    string
	ItemLink     string
	RawSummary   string
	PublishedAt  time.Time
	DedupeKey    string
	ContentHash  string
	SourceFeedID string
}

type rssInboxClassification struct {
	Index      int      `json:"index"`
	Keep       bool     `json:"keep"`
	AISummary  string   `json:"ai_summary,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Importance string   `json:"importance,omitempty"`
	Reason     string   `json:"reason,omitempty"`
}

func buildRSSInboxCandidate(
	feed rsssubscriptions.FeedSubscription,
	item RSSItem,
	fetchedSourceTitle string,
) rssInboxCandidate {
	publishedAt, _ := parseRSSInboxTime(item.PublishedAt)
	sourceTitle := strings.TrimSpace(item.SourceTitle)
	if sourceTitle == "" {
		sourceTitle = strings.TrimSpace(feed.Title)
	}
	if sourceTitle == "" {
		sourceTitle = strings.TrimSpace(fetchedSourceTitle)
	}
	return rssInboxCandidate{
		FeedID:       feed.ID,
		SourceFeedID: feed.ID,
		FeedURL:      feed.URL,
		SourceTitle:  sourceTitle,
		FeedTags:     append([]string(nil), feed.Tags...),
		ItemTitle:    strings.TrimSpace(item.Title),
		ItemLink:     strings.TrimSpace(item.Link),
		RawSummary:   strings.TrimSpace(item.Summary),
		PublishedAt:  publishedAt,
		DedupeKey:    buildRSSInboxDedupeKey(feed.ID, item),
		ContentHash:  buildRSSInboxContentHash(feed.ID, item),
	}
}

func buildRSSInboxItems(
	batch []rssInboxCandidate,
	decisions []rssInboxClassification,
	traceID string,
	savedAt time.Time,
) []RSSInboxItem {
	decisionByIndex := make(map[int]rssInboxClassification, len(decisions))
	for _, decision := range decisions {
		decisionByIndex[decision.Index] = decision
	}
	items := make([]RSSInboxItem, 0, len(batch))
	for index, candidate := range batch {
		decision, ok := decisionByIndex[index]
		if !ok || !decision.Keep {
			continue
		}
		items = append(items, RSSInboxItem{
			FeedID:       candidate.FeedID,
			SourceFeedID: candidate.SourceFeedID,
			FeedURL:      candidate.FeedURL,
			SourceTitle:  candidate.SourceTitle,
			ItemTitle:    candidate.ItemTitle,
			ItemLink:     candidate.ItemLink,
			PublishedAt:  candidate.PublishedAt,
			RawSummary:   candidate.RawSummary,
			AISummary:    truncateRunes(strings.TrimSpace(decision.AISummary), maxRSSClassifierSummaryLen),
			Tags:         normalizeRSSInboxTags(append(append([]string(nil), candidate.FeedTags...), decision.Tags...)),
			Importance:   normalizeRSSInboxImportance(decision.Importance),
			Reason:       truncateRunes(strings.TrimSpace(decision.Reason), 180),
			ContentHash:  candidate.ContentHash,
			DedupeKey:    candidate.DedupeKey,
			TraceID:      strings.TrimSpace(traceID),
			SavedAt:      savedAt.UTC(),
		})
	}
	return items
}

func buildRSSInboxDedupeKey(feedID string, item RSSItem) string {
	base := strings.TrimSpace(item.ID)
	if base != "" {
		return strings.TrimSpace(feedID) + "::id::" + base
	}
	if link := strings.TrimSpace(item.Link); link != "" {
		return strings.TrimSpace(feedID) + "::link::" + link
	}
	hash := sha256.Sum256([]byte(strings.TrimSpace(item.Title) + "\n" + strings.TrimSpace(item.PublishedAt)))
	return strings.TrimSpace(feedID) + "::hash::" + hex.EncodeToString(hash[:])
}

func buildRSSInboxContentHash(feedID string, item RSSItem) string {
	payload := []string{
		strings.TrimSpace(feedID),
		strings.TrimSpace(item.ID),
		strings.TrimSpace(item.Title),
		strings.TrimSpace(item.Link),
		strings.TrimSpace(item.PublishedAt),
		strings.TrimSpace(item.Summary),
	}
	hash := sha256.Sum256([]byte(strings.Join(payload, "\n")))
	return hex.EncodeToString(hash[:])
}

func parseRSSInboxTime(raw string) (time.Time, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}
