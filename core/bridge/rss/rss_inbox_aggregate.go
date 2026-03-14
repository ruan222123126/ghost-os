package rss

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
)

const (
	defaultRSSAggregateWindowHours = 24
	maxRSSAggregateWindowHours     = 24 * 7
	defaultRSSAggregateGroupLimit  = 12
	maxRSSAggregateGroupLimit      = 50
	defaultRSSAggregateItemLimit   = 200
	maxRSSAggregateItemLimit       = 500
	defaultRSSGroupItemsPerGroup   = 5
	maxRSSGroupItemsPerGroup       = 10
)

type RSSInboxGroupQuery struct {
	FeedID        string
	Tag           string
	Importance    string
	WindowHours   int
	Limit         int
	ItemLimit     int
	ItemsPerGroup int
}

type RSSInboxGroupResult struct {
	WindowHours  int                  `json:"window_hours"`
	GeneratedAt  time.Time            `json:"generated_at"`
	ScannedItems int                  `json:"scanned_items"`
	GroupCount   int                  `json:"group_count"`
	Groups       []RSSInboxTopicGroup `json:"groups"`
}

type RSSInboxTopicGroup struct {
	ID                string         `json:"id"`
	TopicLabel        string         `json:"topic_label"`
	Headline          string         `json:"headline"`
	Summary           string         `json:"summary,omitempty"`
	Importance        string         `json:"importance"`
	WindowStart       time.Time      `json:"window_start"`
	WindowEnd         time.Time      `json:"window_end"`
	LatestActivityAt  time.Time      `json:"latest_activity_at"`
	LatestPublishedAt time.Time      `json:"latest_published_at,omitempty"`
	LatestSavedAt     time.Time      `json:"latest_saved_at"`
	ItemCount         int            `json:"item_count"`
	FeedCount         int            `json:"feed_count"`
	Tags              []string       `json:"tags,omitempty"`
	FeedIDs           []string       `json:"feed_ids,omitempty"`
	Items             []RSSInboxItem `json:"items,omitempty"`
}

type rssAggregateGroupState struct {
	items           []RSSInboxItem
	windowStart     time.Time
	windowEnd       time.Time
	latestActivity  time.Time
	latestPublished time.Time
	latestSaved     time.Time
	feedIDs         map[string]struct{}
	tagCounts       map[string]int
	tokenCounts     map[string]int
	headline        string
	summary         string
	importance      string
}

var rssTopicStopwords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {}, "be": {}, "by": {}, "for": {}, "from": {},
	"how": {}, "in": {}, "into": {}, "is": {}, "it": {}, "its": {}, "new": {}, "of": {}, "on": {}, "or": {},
	"that": {}, "the": {}, "their": {}, "this": {}, "to": {}, "via": {}, "with": {}, "your": {},
}

func (s *RSSInboxService) Aggregate(query RSSInboxGroupQuery) (RSSInboxGroupResult, error) {
	if s == nil || s.inboxStore == nil {
		return RSSInboxGroupResult{}, fmt.Errorf("rss inbox service is not configured")
	}

	normalized := normalizeRSSInboxGroupQuery(query)
	items, err := s.inboxStore.List(RSSInboxListFilter{
		FeedID:     strings.TrimSpace(normalized.FeedID),
		Tag:        strings.TrimSpace(normalized.Tag),
		Importance: strings.TrimSpace(normalized.Importance),
		Limit:      normalized.ItemLimit,
	})
	if err != nil {
		return RSSInboxGroupResult{}, err
	}

	now := s.currentTime().UTC()
	cutoff := now.Add(-time.Duration(normalized.WindowHours) * time.Hour)
	filtered := make([]RSSInboxItem, 0, len(items))
	for _, item := range items {
		if rssInboxActivityTime(item).Before(cutoff) {
			continue
		}
		filtered = append(filtered, item)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		leftImportance := rssImportanceRank(filtered[i].Importance)
		rightImportance := rssImportanceRank(filtered[j].Importance)
		if leftImportance != rightImportance {
			return leftImportance > rightImportance
		}
		leftTime := rssInboxActivityTime(filtered[i])
		rightTime := rssInboxActivityTime(filtered[j])
		if !leftTime.Equal(rightTime) {
			return leftTime.After(rightTime)
		}
		return filtered[i].SavedAt.After(filtered[j].SavedAt)
	})

	bucketSize := rssAggregateBucketSize(normalized.WindowHours)
	states := make([]*rssAggregateGroupState, 0, len(filtered))
	for _, item := range filtered {
		bucketStart := rssBucketFloor(rssInboxActivityTime(item), bucketSize)
		bucketEnd := bucketStart.Add(bucketSize)
		itemTags := normalizeRSSInboxTags(item.Tags)
		itemTokens := extractRSSInboxTopicTokens(item)

		var best *rssAggregateGroupState
		bestScore := -1
		for _, state := range states {
			if !state.windowStart.Equal(bucketStart) {
				continue
			}
			score := rssAggregateMatchScore(state, itemTags, itemTokens)
			if score > bestScore {
				best = state
				bestScore = score
			}
		}
		if best == nil || bestScore < 2 {
			state := newRSSAggregateGroupState(item, bucketStart, bucketEnd, itemTags, itemTokens)
			states = append(states, state)
			continue
		}
		best.add(item, itemTags, itemTokens)
	}

	groups := make([]RSSInboxTopicGroup, 0, len(states))
	for _, state := range states {
		groups = append(groups, state.toGroup(normalized.ItemsPerGroup))
	}
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
	if len(groups) > normalized.Limit {
		groups = groups[:normalized.Limit]
	}

	return RSSInboxGroupResult{
		WindowHours:  normalized.WindowHours,
		GeneratedAt:  now,
		ScannedItems: len(filtered),
		GroupCount:   len(groups),
		Groups:       groups,
	}, nil
}

func normalizeRSSInboxGroupQuery(query RSSInboxGroupQuery) RSSInboxGroupQuery {
	query.FeedID = strings.TrimSpace(query.FeedID)
	query.Tag = strings.TrimSpace(query.Tag)
	query.Importance = strings.TrimSpace(query.Importance)
	if query.WindowHours <= 0 {
		query.WindowHours = defaultRSSAggregateWindowHours
	}
	if query.WindowHours > maxRSSAggregateWindowHours {
		query.WindowHours = maxRSSAggregateWindowHours
	}
	if query.Limit <= 0 {
		query.Limit = defaultRSSAggregateGroupLimit
	}
	if query.Limit > maxRSSAggregateGroupLimit {
		query.Limit = maxRSSAggregateGroupLimit
	}
	if query.ItemLimit <= 0 {
		query.ItemLimit = defaultRSSAggregateItemLimit
	}
	if query.ItemLimit > maxRSSAggregateItemLimit {
		query.ItemLimit = maxRSSAggregateItemLimit
	}
	if query.ItemsPerGroup <= 0 {
		query.ItemsPerGroup = defaultRSSGroupItemsPerGroup
	}
	if query.ItemsPerGroup > maxRSSGroupItemsPerGroup {
		query.ItemsPerGroup = maxRSSGroupItemsPerGroup
	}
	return query
}

func newRSSAggregateGroupState(item RSSInboxItem, bucketStart time.Time, bucketEnd time.Time, tags []string, tokens []string) *rssAggregateGroupState {
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
	if item.SavedAt.After(g.latestSaved) {
		g.latestSaved = item.SavedAt
	}
	if item.PublishedAt.After(g.latestPublished) {
		g.latestPublished = item.PublishedAt
	}
}

func (g *rssAggregateGroupState) toGroup(itemsPerGroup int) RSSInboxTopicGroup {
	items := make([]RSSInboxItem, len(g.items))
	for i := range g.items {
		items[i] = cloneRSSInboxItem(g.items[i])
	}
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
	if len(items) > itemsPerGroup {
		items = items[:itemsPerGroup]
	}

	feedIDs := make([]string, 0, len(g.feedIDs))
	for feedID := range g.feedIDs {
		feedIDs = append(feedIDs, feedID)
	}
	sort.Strings(feedIDs)
	tags := topRSSAggregateKeys(g.tagCounts, 4)
	if len(tags) == 0 {
		tags = topRSSAggregateKeys(g.tokenCounts, 4)
	}
	topicLabel := strings.Join(tags, " / ")
	if topicLabel == "" {
		topicLabel = strings.TrimSpace(g.headline)
	}
	if topicLabel == "" {
		topicLabel = "rss-topic"
	}

	groupID := newRSSAggregateGroupID(g.windowStart, topicLabel, feedIDs)
	return RSSInboxTopicGroup{
		ID:                groupID,
		TopicLabel:        topicLabel,
		Headline:          strings.TrimSpace(g.headline),
		Summary:           strings.TrimSpace(g.summary),
		Importance:        normalizeRSSInboxImportance(g.importance),
		WindowStart:       g.windowStart.UTC(),
		WindowEnd:         g.windowEnd.UTC(),
		LatestActivityAt:  g.latestActivity.UTC(),
		LatestPublishedAt: g.latestPublished.UTC(),
		LatestSavedAt:     g.latestSaved.UTC(),
		ItemCount:         len(g.items),
		FeedCount:         len(g.feedIDs),
		Tags:              tags,
		FeedIDs:           feedIDs,
		Items:             items,
	}
}

func rssAggregateMatchScore(group *rssAggregateGroupState, tags []string, tokens []string) int {
	if group == nil {
		return 0
	}
	tagOverlap := 0
	for _, tag := range tags {
		if group.tagCounts[tag] > 0 {
			tagOverlap++
		}
	}
	tokenOverlap := 0
	for _, token := range tokens {
		if group.tokenCounts[token] > 0 {
			tokenOverlap++
		}
	}
	switch {
	case tagOverlap >= 2:
		return 5 + tokenOverlap
	case tagOverlap >= 1 && tokenOverlap >= 1:
		return 4 + tokenOverlap
	case tokenOverlap >= 3:
		return 3 + tokenOverlap
	case tokenOverlap >= 2:
		return 2 + tokenOverlap
	default:
		return tokenOverlap
	}
}

func extractRSSInboxTopicTokens(item RSSInboxItem) []string {
	seen := make(map[string]struct{}, len(item.Tags)+8)
	out := make([]string, 0, len(item.Tags)+8)
	for _, tag := range normalizeRSSInboxTags(item.Tags) {
		if tag == "" {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	for _, token := range tokenizeRSSInboxText(item.ItemTitle) {
		if _, exists := seen[token]; exists {
			continue
		}
		seen[token] = struct{}{}
		out = append(out, token)
	}
	return out
}

func tokenizeRSSInboxText(input string) []string {
	fields := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(input)), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if len(field) < 3 {
			continue
		}
		if _, blocked := rssTopicStopwords[field]; blocked {
			continue
		}
		out = append(out, field)
	}
	return out
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

func shouldReplaceRSSGroupHeadline(item RSSInboxItem, currentHeadline string, currentSummary string, currentImportance string) bool {
	itemImportance := rssImportanceRank(item.Importance)
	groupImportance := rssImportanceRank(currentImportance)
	if itemImportance != groupImportance {
		return itemImportance > groupImportance
	}
	if strings.TrimSpace(currentSummary) == "" && strings.TrimSpace(rssInboxItemSummary(item)) != "" {
		return true
	}
	return strings.TrimSpace(item.ItemTitle) != "" && strings.TrimSpace(currentHeadline) == ""
}

func rssInboxItemSummary(item RSSInboxItem) string {
	if summary := strings.TrimSpace(item.AISummary); summary != "" {
		return summary
	}
	return strings.TrimSpace(item.RawSummary)
}

func rssImportanceRank(value string) int {
	switch normalizeRSSInboxImportance(value) {
	case "high":
		return 3
	case "normal":
		return 2
	default:
		return 1
	}
}

func rssInboxActivityTime(item RSSInboxItem) time.Time {
	if !item.PublishedAt.IsZero() {
		return item.PublishedAt.UTC()
	}
	return item.SavedAt.UTC()
}

func rssAggregateBucketSize(windowHours int) time.Duration {
	switch {
	case windowHours <= 24:
		return 6 * time.Hour
	case windowHours <= 72:
		return 12 * time.Hour
	default:
		return 24 * time.Hour
	}
}

func rssBucketFloor(ts time.Time, size time.Duration) time.Time {
	if ts.IsZero() {
		return time.Time{}
	}
	if size <= 0 {
		size = 6 * time.Hour
	}
	unix := ts.UTC().Unix()
	seconds := int64(size / time.Second)
	return time.Unix((unix/seconds)*seconds, 0).UTC()
}

func newRSSAggregateGroupID(windowStart time.Time, topicLabel string, feedIDs []string) string {
	payload := windowStart.UTC().Format(time.RFC3339) + "\n" + strings.TrimSpace(topicLabel) + "\n" + strings.Join(feedIDs, ",")
	hash := sha256.Sum256([]byte(payload))
	return "rssg_" + hex.EncodeToString(hash[:12])
}
