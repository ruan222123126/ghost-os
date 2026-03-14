package rss

import (
	"sort"
	"strings"
	"time"
)

type rssInboxListQuery struct {
	feedID          string
	tag             string
	importance      string
	savedAfter      time.Time
	savedBefore     time.Time
	publishedAfter  time.Time
	publishedBefore time.Time
	limit           int
}

func newRSSInboxListQuery(filter RSSInboxListFilter) rssInboxListQuery {
	return rssInboxListQuery{
		feedID:          strings.TrimSpace(filter.FeedID),
		tag:             strings.ToLower(strings.TrimSpace(filter.Tag)),
		importance:      normalizedRSSInboxFilterImportance(filter.Importance),
		savedAfter:      utcOrZero(filter.SavedAfter),
		savedBefore:     utcOrZero(filter.SavedBefore),
		publishedAfter:  utcOrZero(filter.PublishedAfter),
		publishedBefore: utcOrZero(filter.PublishedBefore),
		limit:           normalizedRSSInboxListLimit(filter.Limit),
	}
}

func (q rssInboxListQuery) matches(item RSSInboxItem) bool {
	if q.feedID != "" && item.FeedID != q.feedID {
		return false
	}
	if q.importance != "" && item.Importance != q.importance {
		return false
	}
	if q.tag != "" && !containsRSSInboxTag(item.Tags, q.tag) {
		return false
	}
	if !q.savedAfter.IsZero() && item.SavedAt.Before(q.savedAfter) {
		return false
	}
	if !q.savedBefore.IsZero() && item.SavedAt.After(q.savedBefore) {
		return false
	}
	if !q.publishedAfter.IsZero() {
		if item.PublishedAt.IsZero() || item.PublishedAt.Before(q.publishedAfter) {
			return false
		}
	}
	if !q.publishedBefore.IsZero() {
		if item.PublishedAt.IsZero() || item.PublishedAt.After(q.publishedBefore) {
			return false
		}
	}
	return true
}

func listRSSInboxItems(items []RSSInboxItem, query rssInboxListQuery) []RSSInboxItem {
	filtered := make([]RSSInboxItem, 0, len(items))
	for _, item := range items {
		if !query.matches(item) {
			continue
		}
		filtered = append(filtered, cloneRSSInboxItem(item))
	}
	sortRSSInboxItemsNewestFirst(filtered)
	if len(filtered) > query.limit {
		filtered = filtered[:query.limit]
	}
	return filtered
}

func sortRSSInboxItemsNewestFirst(items []RSSInboxItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].SavedAt.Equal(items[j].SavedAt) {
			return items[i].ID > items[j].ID
		}
		return items[i].SavedAt.After(items[j].SavedAt)
	})
}

func containsRSSInboxTag(tags []string, expected string) bool {
	for _, tag := range tags {
		if strings.EqualFold(strings.TrimSpace(tag), expected) {
			return true
		}
	}
	return false
}

func normalizeRSSInboxTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, raw := range tags {
		tag := strings.ToLower(strings.TrimSpace(raw))
		if tag == "" {
			continue
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		normalized = append(normalized, tag)
	}
	if len(normalized) == 0 {
		return nil
	}
	sort.Strings(normalized)
	return normalized
}

func normalizeRSSInboxImportance(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "low", "high":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return "normal"
	}
}

func normalizedRSSInboxFilterImportance(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	return normalizeRSSInboxImportance(raw)
}

func normalizedRSSInboxListLimit(limit int) int {
	if limit <= 0 {
		return defaultRSSInboxLimit
	}
	return limit
}

func utcOrZero(ts time.Time) time.Time {
	if ts.IsZero() {
		return time.Time{}
	}
	return ts.UTC()
}
