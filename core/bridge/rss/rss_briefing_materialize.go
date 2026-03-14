package rss

import (
	"fmt"
	"strings"
	"time"
)

func normalizeRSSBriefingQuery(query RSSBriefingQuery) RSSBriefingQuery {
	query.FeedID = strings.TrimSpace(query.FeedID)
	query.Tag = strings.TrimSpace(query.Tag)
	query.Importance = strings.TrimSpace(query.Importance)
	query.TraceID = strings.TrimSpace(query.TraceID)
	query.TaskID = strings.TrimSpace(query.TaskID)
	if query.GroupLimit <= 0 {
		query.GroupLimit = defaultRSSBriefingGroupLimit
	}
	if query.GroupLimit > maxRSSBriefingGroupLimit {
		query.GroupLimit = maxRSSBriefingGroupLimit
	}
	if query.HighlightsLimit <= 0 {
		query.HighlightsLimit = defaultRSSBriefingHighlightsLimit
	}
	if query.HighlightsLimit > maxRSSBriefingHighlightsLimit {
		query.HighlightsLimit = maxRSSBriefingHighlightsLimit
	}
	if query.ItemLimit <= 0 {
		query.ItemLimit = defaultRSSAggregateItemLimit
	}
	if query.ItemsPerGroup <= 0 {
		query.ItemsPerGroup = 3
	}
	return query
}

func newRSSBriefingResult(query RSSBriefingQuery, windowHours int, now time.Time) RSSBriefingResult {
	return RSSBriefingResult{
		Title:         "RSS Briefing",
		GeneratedAt:   now.UTC(),
		WindowHours:   windowHours,
		TraceID:       strings.TrimSpace(query.TraceID),
		TaskID:        strings.TrimSpace(query.TaskID),
		Highlights:    []RSSBriefingHighlight{},
		ScannedGroups: 0,
	}
}

func finalizeRSSBriefingResult(
	result RSSBriefingResult,
	draft rssBriefingDraft,
	groups []RSSInboxTopicGroup,
	limit int,
) RSSBriefingResult {
	result.Title = rssBriefingTitleOrDefault(strings.TrimSpace(draft.Title), result.WindowHours)
	result.Summary = truncateRunes(strings.TrimSpace(draft.Summary), 400)
	result.ScannedGroups = len(groups)
	result.Highlights = materializeRSSBriefingHighlights(draft.Highlights, groups, limit)
	result.HighlightCount = len(result.Highlights)
	if result.Summary == "" {
		result.Summary = fallbackRSSBriefingSummary(result.Highlights)
	}
	return result
}

func materializeRSSBriefingHighlights(
	draft []rssBriefingDraftHighlight,
	groups []RSSInboxTopicGroup,
	limit int,
) []RSSBriefingHighlight {
	if limit <= 0 {
		limit = defaultRSSBriefingHighlightsLimit
	}
	groupByID := make(map[string]RSSInboxTopicGroup, len(groups))
	for _, group := range groups {
		groupByID[group.ID] = group
	}

	out := make([]RSSBriefingHighlight, 0, limit)
	seen := make(map[string]struct{}, limit)
	appendHighlight := func(group RSSInboxTopicGroup, item rssBriefingDraftHighlight) bool {
		if _, exists := seen[group.ID]; exists {
			return false
		}
		seen[group.ID] = struct{}{}
		out = append(out, RSSBriefingHighlight{
			Rank:            len(out) + 1,
			GroupID:         group.ID,
			TopicLabel:      group.TopicLabel,
			Headline:        firstNonEmptyString(item.Headline, group.Headline, group.TopicLabel),
			Summary:         firstNonEmptyString(item.Summary, group.Summary),
			WhyItMatters:    truncateRunes(strings.TrimSpace(item.WhyItMatters), 220),
			Importance:      normalizeRSSInboxImportance(firstNonEmptyString(item.Importance, group.Importance)),
			SourceItemCount: group.ItemCount,
			SourceFeedCount: group.FeedCount,
			Tags:            append([]string(nil), group.Tags...),
		})
		return len(out) >= limit
	}
	for _, item := range draft {
		group, ok := groupByID[item.GroupID]
		if ok && appendHighlight(group, item) {
			return out
		}
	}
	for _, group := range groups {
		if appendHighlight(group, rssBriefingDraftHighlight{}) {
			return out
		}
	}
	return out
}

func fallbackRSSBriefingSummary(highlights []RSSBriefingHighlight) string {
	if len(highlights) == 0 {
		return ""
	}
	if len(highlights) == 1 {
		return firstNonEmptyString(highlights[0].Summary, highlights[0].Headline)
	}
	return fmt.Sprintf("%d notable RSS developments selected for this briefing.", len(highlights))
}

func rssBriefingTitleOrDefault(input string, windowHours int) string {
	if strings.TrimSpace(input) != "" {
		return input
	}
	return fmt.Sprintf("RSS Briefing (%dh)", windowHours)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
