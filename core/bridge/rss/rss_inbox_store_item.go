package rss

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

func normalizeRSSInboxItem(item RSSInboxItem, fallbackSavedAt time.Time) (RSSInboxItem, error) {
	item.FeedID = strings.TrimSpace(item.FeedID)
	item.SourceFeedID = strings.TrimSpace(item.SourceFeedID)
	item.FeedURL = strings.TrimSpace(item.FeedURL)
	item.SourceTitle = strings.TrimSpace(item.SourceTitle)
	item.ItemTitle = strings.TrimSpace(item.ItemTitle)
	item.ItemLink = strings.TrimSpace(item.ItemLink)
	item.RawSummary = strings.TrimSpace(item.RawSummary)
	item.AISummary = strings.TrimSpace(item.AISummary)
	item.Reason = strings.TrimSpace(item.Reason)
	item.TraceID = strings.TrimSpace(item.TraceID)
	item.ContentHash = strings.TrimSpace(item.ContentHash)
	item.DedupeKey = strings.TrimSpace(item.DedupeKey)
	item.Importance = normalizeRSSInboxImportance(item.Importance)
	item.Tags = normalizeRSSInboxTags(item.Tags)

	if item.FeedID == "" {
		return RSSInboxItem{}, fmt.Errorf("feed_id is required")
	}
	if item.SourceFeedID == "" {
		item.SourceFeedID = item.FeedID
	}
	if item.DedupeKey == "" {
		return RSSInboxItem{}, fmt.Errorf("dedupe_key is required")
	}
	if item.ContentHash == "" {
		return RSSInboxItem{}, fmt.Errorf("content_hash is required")
	}
	if item.SavedAt.IsZero() {
		item.SavedAt = fallbackSavedAt
	}
	item.SavedAt = item.SavedAt.UTC()
	if !item.PublishedAt.IsZero() {
		item.PublishedAt = item.PublishedAt.UTC()
	}
	if strings.TrimSpace(item.ID) == "" {
		item.ID = newRSSInboxItemID(item.DedupeKey, item.ContentHash)
	}
	return item, nil
}

func cloneRSSInboxItem(item RSSInboxItem) RSSInboxItem {
	item.Tags = append([]string(nil), item.Tags...)
	return item
}

func newRSSInboxItemID(dedupeKey string, contentHash string) string {
	payload := strings.TrimSpace(dedupeKey) + "\n" + strings.TrimSpace(contentHash)
	hash := sha256.Sum256([]byte(payload))
	return "rss_" + hex.EncodeToString(hash[:12])
}
