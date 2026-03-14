package rss

import (
	"sort"
	"time"
)

type rssInboxAggregator struct {
	spec rssInboxAggregateSpec
}

func newRSSInboxAggregator(spec rssInboxAggregateSpec) rssInboxAggregator {
	return rssInboxAggregator{spec: spec}
}

func (a rssInboxAggregator) Run(items []RSSInboxItem) RSSInboxGroupResult {
	filtered := a.selectItems(items)
	states := a.groupItems(filtered)
	groups := buildRSSInboxTopicGroups(states, a.spec.itemsPerGroup)
	sortRSSInboxTopicGroups(groups)
	if len(groups) > a.spec.limit {
		groups = groups[:a.spec.limit]
	}
	return buildRSSInboxGroupResult(a.spec, len(filtered), groups)
}

func (a rssInboxAggregator) selectItems(items []RSSInboxItem) []RSSInboxItem {
	filtered := make([]RSSInboxItem, 0, len(items))
	for _, item := range items {
		if !a.spec.includes(item) {
			continue
		}
		filtered = append(filtered, item)
	}
	sortRSSInboxAggregateItems(filtered)
	return filtered
}

func (a rssInboxAggregator) groupItems(items []RSSInboxItem) []*rssAggregateGroupState {
	states := make([]*rssAggregateGroupState, 0, len(items))
	for _, item := range items {
		itemTags := normalizeRSSInboxTags(item.Tags)
		itemTokens := extractRSSInboxTopicTokens(item)
		bucketStart := rssBucketFloor(rssInboxActivityTime(item), a.spec.bucketSize)
		bucketEnd := bucketStart.Add(a.spec.bucketSize)
		best, score := matchRSSAggregateGroup(states, bucketStart, itemTags, itemTokens)
		if best == nil || score < 2 {
			states = append(states, newRSSAggregateGroupState(item, bucketStart, bucketEnd, itemTags, itemTokens))
			continue
		}
		best.add(item, itemTags, itemTokens)
	}
	return states
}

func matchRSSAggregateGroup(
	states []*rssAggregateGroupState,
	bucketStart time.Time,
	tags []string,
	tokens []string,
) (*rssAggregateGroupState, int) {
	var best *rssAggregateGroupState
	bestScore := -1
	for _, state := range states {
		if !state.windowStart.Equal(bucketStart) {
			continue
		}
		score := rssAggregateMatchScore(state, tags, tokens)
		if score > bestScore {
			best = state
			bestScore = score
		}
	}
	return best, bestScore
}

func sortRSSInboxAggregateItems(items []RSSInboxItem) {
	sort.SliceStable(items, func(i, j int) bool {
		leftImportance := rssImportanceRank(items[i].Importance)
		rightImportance := rssImportanceRank(items[j].Importance)
		if leftImportance != rightImportance {
			return leftImportance > rightImportance
		}
		leftTime := rssInboxActivityTime(items[i])
		rightTime := rssInboxActivityTime(items[j])
		if !leftTime.Equal(rightTime) {
			return leftTime.After(rightTime)
		}
		return items[i].SavedAt.After(items[j].SavedAt)
	})
}

func sortRSSInboxTopicGroups(groups []RSSInboxTopicGroup) {
	sort.SliceStable(groups, func(i, j int) bool {
		leftImportance := rssImportanceRank(groups[i].Importance)
		rightImportance := rssImportanceRank(groups[j].Importance)
		if leftImportance != rightImportance {
			return leftImportance > rightImportance
		}
		if !groups[i].LatestActivityAt.Equal(groups[j].LatestActivityAt) {
			return groups[i].LatestActivityAt.After(groups[j].LatestActivityAt)
		}
		if groups[i].ItemCount != groups[j].ItemCount {
			return groups[i].ItemCount > groups[j].ItemCount
		}
		return groups[i].ID < groups[j].ID
	})
}
