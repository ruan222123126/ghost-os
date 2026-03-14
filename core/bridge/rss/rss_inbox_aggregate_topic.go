package rss

import (
	"strings"
	"time"
	"unicode"
)

var rssTopicStopwords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {}, "be": {}, "by": {}, "for": {}, "from": {},
	"how": {}, "in": {}, "into": {}, "is": {}, "it": {}, "its": {}, "new": {}, "of": {}, "on": {}, "or": {},
	"that": {}, "the": {}, "their": {}, "this": {}, "to": {}, "via": {}, "with": {}, "your": {},
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

func shouldReplaceRSSGroupHeadline(
	item RSSInboxItem,
	currentHeadline string,
	currentSummary string,
	currentImportance string,
) bool {
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
