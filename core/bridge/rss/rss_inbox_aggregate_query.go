package rss

import (
	"strings"
	"time"
)

type rssInboxAggregateSpec struct {
	feedID        string
	tag           string
	importance    string
	windowHours   int
	limit         int
	itemLimit     int
	itemsPerGroup int
	generatedAt   time.Time
	cutoff        time.Time
	bucketSize    time.Duration
}

func newRSSInboxAggregateSpec(query RSSInboxGroupQuery, now time.Time) rssInboxAggregateSpec {
	windowHours := normalizeRSSAggregateWindowHours(query.WindowHours)
	generatedAt := now.UTC()
	return rssInboxAggregateSpec{
		feedID:        strings.TrimSpace(query.FeedID),
		tag:           strings.TrimSpace(query.Tag),
		importance:    strings.TrimSpace(query.Importance),
		windowHours:   windowHours,
		limit:         normalizeRSSAggregateGroupLimit(query.Limit),
		itemLimit:     normalizeRSSAggregateItemLimit(query.ItemLimit),
		itemsPerGroup: normalizeRSSAggregateItemsPerGroup(query.ItemsPerGroup),
		generatedAt:   generatedAt,
		cutoff:        generatedAt.Add(-time.Duration(windowHours) * time.Hour),
		bucketSize:    rssAggregateBucketSize(windowHours),
	}
}

func (s rssInboxAggregateSpec) listFilter() RSSInboxListFilter {
	return RSSInboxListFilter{
		FeedID:     s.feedID,
		Tag:        s.tag,
		Importance: s.importance,
		Limit:      s.itemLimit,
	}
}

func (s rssInboxAggregateSpec) includes(item RSSInboxItem) bool {
	return !rssInboxActivityTime(item).Before(s.cutoff)
}

func normalizeRSSAggregateWindowHours(value int) int {
	if value <= 0 {
		return defaultRSSAggregateWindowHours
	}
	if value > maxRSSAggregateWindowHours {
		return maxRSSAggregateWindowHours
	}
	return value
}

func normalizeRSSAggregateGroupLimit(value int) int {
	if value <= 0 {
		return defaultRSSAggregateGroupLimit
	}
	if value > maxRSSAggregateGroupLimit {
		return maxRSSAggregateGroupLimit
	}
	return value
}

func normalizeRSSAggregateItemLimit(value int) int {
	if value <= 0 {
		return defaultRSSAggregateItemLimit
	}
	if value > maxRSSAggregateItemLimit {
		return maxRSSAggregateItemLimit
	}
	return value
}

func normalizeRSSAggregateItemsPerGroup(value int) int {
	if value <= 0 {
		return defaultRSSGroupItemsPerGroup
	}
	if value > maxRSSGroupItemsPerGroup {
		return maxRSSGroupItemsPerGroup
	}
	return value
}
