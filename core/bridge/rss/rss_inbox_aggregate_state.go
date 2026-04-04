package rss

import (
	"strings"
	"time"
)

const (
	rssMatchScoreStrongTagOverlap = 2
	rssMatchScoreWeakTagOverlap   = 1
	rssMatchScoreHighTokenOverlap = 3
	rssMatchScoreMidTokenOverlap  = 2
	rssMatchScoreStrongTagBase    = 5
	rssMatchScoreTagTokenBase     = 4
	rssMatchScoreHighTokenBase    = 3
	rssMatchScoreMidTokenBase     = 2
)

func newRSSAggregateGroupState(
	item RSSInboxItem,
	bucketStart time.Time,
	bucketEnd time.Time,
	tags []string,
	tokens []string,
) *rssAggregateGroupState {
	state := &rssAggregateGroupState{
		items:       []RSSInboxItem{cloneRSSInboxItem(item)},
		windowStart: bucketStart,
		windowEnd:   bucketEnd,
		feedIDs:     make(map[string]struct{}, 1),
		tagCounts:   make(map[string]int, len(tags)),
		tokenCounts: make(map[string]int, len(tokens)),
		headline:    strings.TrimSpace(item.ItemTitle),
		summary:     rssInboxItemSummary(item),
		importance:  normalizeRSSInboxImportance(item.Importance),
	}
	state.feedIDs[item.FeedID] = struct{}{}
	for _, tag := range tags {
		state.tagCounts[tag]++
	}
	for _, token := range tokens {
		state.tokenCounts[token]++
	}
	state.updateTimes(item)
	return state
}

func (g *rssAggregateGroupState) add(item RSSInboxItem, tags []string, tokens []string) {
	g.items = append(g.items, cloneRSSInboxItem(item))
	if item.FeedID != "" {
		g.feedIDs[item.FeedID] = struct{}{}
	}
	for _, tag := range tags {
		g.tagCounts[tag]++
	}
	for _, token := range tokens {
		g.tokenCounts[token]++
	}
	if rssImportanceRank(item.Importance) > rssImportanceRank(g.importance) {
		g.importance = normalizeRSSInboxImportance(item.Importance)
	}
	if shouldReplaceRSSGroupHeadline(item, g.headline, g.summary, g.importance) {
		g.headline = strings.TrimSpace(item.ItemTitle)
		g.summary = rssInboxItemSummary(item)
	}
	g.updateTimes(item)
}

func (g *rssAggregateGroupState) updateTimes(item RSSInboxItem) {
	activityAt := rssInboxActivityTime(item)
	if activityAt.After(g.latestActivity) {
		g.latestActivity = activityAt
	}
	if item.PublishedAt.After(g.latestPublished) {
		g.latestPublished = item.PublishedAt
	}
	if item.SavedAt.After(g.latestSaved) {
		g.latestSaved = item.SavedAt
	}
}

func rssAggregateMatchScore(group *rssAggregateGroupState, tags []string, tokens []string) int {
	if group == nil {
		return 0
	}
	tagOverlap := rssAggregateOverlapCount(group.tagCounts, tags)
	tokenOverlap := rssAggregateOverlapCount(group.tokenCounts, tokens)
	return rssAggregateScore(tagOverlap, tokenOverlap)
}

func rssAggregateOverlapCount(counters map[string]int, values []string) int {
	overlap := 0
	for _, value := range values {
		if counters[value] > 0 {
			overlap++
		}
	}
	return overlap
}

func rssAggregateScore(tagOverlap int, tokenOverlap int) int {
	switch {
	case tagOverlap >= rssMatchScoreStrongTagOverlap:
		return rssMatchScoreStrongTagBase + tokenOverlap
	case tagOverlap >= rssMatchScoreWeakTagOverlap && tokenOverlap >= rssMatchScoreWeakTagOverlap:
		return rssMatchScoreTagTokenBase + tokenOverlap
	case tokenOverlap >= rssMatchScoreHighTokenOverlap:
		return rssMatchScoreHighTokenBase + tokenOverlap
	case tokenOverlap >= rssMatchScoreMidTokenOverlap:
		return rssMatchScoreMidTokenBase + tokenOverlap
	default:
		return tokenOverlap
	}
}
