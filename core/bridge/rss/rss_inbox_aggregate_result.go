package rss

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"time"
)

func buildRSSInboxGroupResult(
	spec rssInboxAggregateSpec,
	scannedItems int,
	groups []RSSInboxTopicGroup,
) RSSInboxGroupResult {
	return RSSInboxGroupResult{
		WindowHours:  spec.windowHours,
		GeneratedAt:  spec.generatedAt,
		ScannedItems: scannedItems,
		GroupCount:   len(groups),
		Groups:       groups,
	}
}

func buildRSSInboxTopicGroups(states []*rssAggregateGroupState, itemsPerGroup int) []RSSInboxTopicGroup {
	groups := make([]RSSInboxTopicGroup, 0, len(states))
	for _, state := range states {
		groups = append(groups, buildRSSInboxTopicGroup(state, itemsPerGroup))
	}
	return groups
}

func buildRSSInboxTopicGroup(state *rssAggregateGroupState, itemsPerGroup int) RSSInboxTopicGroup {
	items := cloneRSSInboxGroupItems(state.items)
	sortRSSInboxAggregateItems(items)
	if len(items) > itemsPerGroup {
		items = items[:itemsPerGroup]
	}

	feedIDs := sortedRSSInboxFeedIDs(state.feedIDs)
	tags := topRSSAggregateKeys(state.tagCounts, 4)
	if len(tags) == 0 {
		tags = topRSSAggregateKeys(state.tokenCounts, 4)
	}
	topicLabel := buildRSSAggregateTopicLabel(tags, state.headline)
	return RSSInboxTopicGroup{
		ID:                newRSSAggregateGroupID(state.windowStart, topicLabel, feedIDs),
		TopicLabel:        topicLabel,
		Headline:          strings.TrimSpace(state.headline),
		Summary:           strings.TrimSpace(state.summary),
		Importance:        normalizeRSSInboxImportance(state.importance),
		WindowStart:       state.windowStart.UTC(),
		WindowEnd:         state.windowEnd.UTC(),
		LatestActivityAt:  state.latestActivity.UTC(),
		LatestPublishedAt: state.latestPublished.UTC(),
		LatestSavedAt:     state.latestSaved.UTC(),
		ItemCount:         len(state.items),
		FeedCount:         len(state.feedIDs),
		Tags:              tags,
		FeedIDs:           feedIDs,
		Items:             items,
	}
}

func cloneRSSInboxGroupItems(items []RSSInboxItem) []RSSInboxItem {
	cloned := make([]RSSInboxItem, len(items))
	for i := range items {
		cloned[i] = cloneRSSInboxItem(items[i])
	}
	return cloned
}

func sortedRSSInboxFeedIDs(feedIDs map[string]struct{}) []string {
	out := make([]string, 0, len(feedIDs))
	for feedID := range feedIDs {
		out = append(out, feedID)
	}
	sort.Strings(out)
	return out
}

func buildRSSAggregateTopicLabel(tags []string, headline string) string {
	if label := strings.Join(tags, " / "); label != "" {
		return label
	}
	if label := strings.TrimSpace(headline); label != "" {
		return label
	}
	return "rss-topic"
}

func topRSSAggregateKeys(counts map[string]int, limit int) []string {
	if len(counts) == 0 || limit <= 0 {
		return nil
	}

	type kv struct {
		key   string
		count int
	}

	values := make([]kv, 0, len(counts))
	for key, count := range counts {
		if strings.TrimSpace(key) == "" || count <= 0 {
			continue
		}
		values = append(values, kv{key: key, count: count})
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].count != values[j].count {
			return values[i].count > values[j].count
		}
		return values[i].key < values[j].key
	})
	if len(values) > limit {
		values = values[:limit]
	}

	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.key)
	}
	return out
}

func newRSSAggregateGroupID(windowStart time.Time, topicLabel string, feedIDs []string) string {
	payload := windowStart.UTC().Format(time.RFC3339) + "\n" + strings.TrimSpace(topicLabel)
	payload += "\n" + strings.Join(feedIDs, ",")
	hash := sha256.Sum256([]byte(payload))
	return "rssg_" + hex.EncodeToString(hash[:12])
}
