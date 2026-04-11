package rss

import (
	"fmt"
	"strings"
)

// FormatInboxPollPreview formats a poll result for task response preview.
func FormatInboxPollPreview(result RSSInboxPollResult) string {
	return fmt.Sprintf(
		"rss poll feeds=%d failed=%d saved=%d discarded=%d",
		result.FeedsScanned,
		result.FeedsFailed,
		result.ItemsSaved,
		result.ItemsDiscarded,
	)
}

// FormatBriefingPreview formats a briefing result for task response preview.
func FormatBriefingPreview(result RSSBriefingResult) string {
	preview := fmt.Sprintf(
		"rss briefing highlights=%d groups=%d title=%s",
		result.HighlightCount,
		result.ScannedGroups,
		truncateRunes(strings.TrimSpace(result.Title), 80),
	)
	if result.Report != nil && strings.TrimSpace(result.Report.MarkdownPath) != "" {
		preview += " report=" + truncateRunes(strings.TrimSpace(result.Report.MarkdownPath), 120)
	}
	if strings.TrimSpace(result.ReportError) != "" {
		preview += " report_error=" + truncateRunes(strings.TrimSpace(result.ReportError), 80)
	}
	return preview
}
