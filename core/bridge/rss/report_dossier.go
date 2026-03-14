package rss

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	rssReportDossierDirPerm  = 0o700
	rssReportDossierFilePerm = 0o600
	timeRFC3339              = time.RFC3339
)

func renderRSSReportDossier(report RSSReportResult, briefing RSSBriefingResult, groups []RSSInboxTopicGroup) string {
	var builder strings.Builder
	builder.WriteString("# RSS Report Dossier\n\n")
	writeRSSReportContextSection(&builder, report, briefing, groups)
	writeRSSReportHighlightsSection(&builder, briefing)
	writeRSSReportGroupsSection(&builder, groups)
	return strings.TrimSpace(builder.String())
}

func writeRSSReportContextSection(
	builder *strings.Builder,
	report RSSReportResult,
	briefing RSSBriefingResult,
	groups []RSSInboxTopicGroup,
) {
	builder.WriteString("## Report Context\n\n")
	builder.WriteString("- Report ID: ")
	builder.WriteString(report.ID)
	builder.WriteString("\n- Briefing ID: ")
	builder.WriteString(briefing.ID)
	builder.WriteString("\n- Title: ")
	builder.WriteString(firstNonEmptyString(briefing.Title, report.Title))
	builder.WriteString("\n- Summary: ")
	builder.WriteString(firstNonEmptyString(briefing.Summary, report.Summary))
	builder.WriteString("\n")
	if !briefing.GeneratedAt.IsZero() {
		builder.WriteString("- Briefing generated at: ")
		builder.WriteString(briefing.GeneratedAt.UTC().Format(timeRFC3339))
		builder.WriteString("\n")
	}
	builder.WriteString("- Highlight count: ")
	builder.WriteString(fmt.Sprintf("%d", len(briefing.Highlights)))
	builder.WriteString("\n- Group count: ")
	builder.WriteString(fmt.Sprintf("%d", len(groups)))
	builder.WriteString("\n\n")
}

func writeRSSReportHighlightsSection(builder *strings.Builder, briefing RSSBriefingResult) {
	builder.WriteString("## Highlights\n\n")
	if len(briefing.Highlights) == 0 {
		builder.WriteString("- No highlights were generated.\n\n")
		return
	}
	for _, highlight := range briefing.Highlights {
		writeRSSReportHighlight(builder, highlight)
	}
}

func writeRSSReportHighlight(builder *strings.Builder, highlight RSSBriefingHighlight) {
	builder.WriteString("### ")
	builder.WriteString(firstNonEmptyString(highlight.Headline, highlight.TopicLabel, highlight.GroupID))
	builder.WriteString("\n- Group ID: ")
	builder.WriteString(highlight.GroupID)
	builder.WriteString("\n- Topic: ")
	builder.WriteString(firstNonEmptyString(highlight.TopicLabel, "n/a"))
	builder.WriteString("\n- Importance: ")
	builder.WriteString(firstNonEmptyString(highlight.Importance, "normal"))
	builder.WriteString("\n- Summary: ")
	builder.WriteString(firstNonEmptyString(highlight.Summary, "n/a"))
	builder.WriteString("\n- Why it matters: ")
	builder.WriteString(firstNonEmptyString(highlight.WhyItMatters, "n/a"))
	builder.WriteString("\n- Source item count: ")
	builder.WriteString(fmt.Sprintf("%d", highlight.SourceItemCount))
	builder.WriteString("\n- Source feed count: ")
	builder.WriteString(fmt.Sprintf("%d", highlight.SourceFeedCount))
	if len(highlight.Tags) > 0 {
		builder.WriteString("\n- Tags: ")
		builder.WriteString(strings.Join(highlight.Tags, ", "))
	}
	builder.WriteString("\n\n")
}

func writeRSSReportGroupsSection(builder *strings.Builder, groups []RSSInboxTopicGroup) {
	builder.WriteString("## Aggregated Groups\n\n")
	if len(groups) == 0 {
		builder.WriteString("- No groups were available.\n")
		return
	}
	for _, group := range groups {
		writeRSSReportGroup(builder, group)
	}
}

func writeRSSReportGroup(builder *strings.Builder, group RSSInboxTopicGroup) {
	builder.WriteString("### ")
	builder.WriteString(firstNonEmptyString(group.Headline, group.TopicLabel, group.ID))
	builder.WriteString("\n- Group ID: ")
	builder.WriteString(group.ID)
	builder.WriteString("\n- Topic label: ")
	builder.WriteString(firstNonEmptyString(group.TopicLabel, "n/a"))
	builder.WriteString("\n- Importance: ")
	builder.WriteString(firstNonEmptyString(group.Importance, "normal"))
	builder.WriteString("\n- Summary: ")
	builder.WriteString(firstNonEmptyString(group.Summary, "n/a"))
	builder.WriteString("\n- Item count: ")
	builder.WriteString(fmt.Sprintf("%d", group.ItemCount))
	builder.WriteString("\n- Feed count: ")
	builder.WriteString(fmt.Sprintf("%d", group.FeedCount))
	if len(group.Tags) > 0 {
		builder.WriteString("\n- Tags: ")
		builder.WriteString(strings.Join(group.Tags, ", "))
	}
	if !group.LatestActivityAt.IsZero() {
		builder.WriteString("\n- Latest activity at: ")
		builder.WriteString(group.LatestActivityAt.UTC().Format(timeRFC3339))
	}
	builder.WriteString("\n\n#### Source Items\n\n")
	writeRSSReportSourceItems(builder, group.Items)
	builder.WriteString("\n")
}

func writeRSSReportSourceItems(builder *strings.Builder, items []RSSInboxItem) {
	if len(items) == 0 {
		builder.WriteString("- No items.\n")
		return
	}
	for _, item := range items {
		builder.WriteString("- Source: ")
		builder.WriteString(rssReportItemSourceTitle(item))
		builder.WriteString("\n  Title: ")
		builder.WriteString(firstNonEmptyString(item.ItemTitle, "Untitled"))
		if link := strings.TrimSpace(item.ItemLink); link != "" {
			builder.WriteString("\n  Link: ")
			builder.WriteString(link)
		}
		if summary := rssInboxItemSummary(item); summary != "" {
			builder.WriteString("\n  Summary: ")
			builder.WriteString(summary)
		}
		if !item.PublishedAt.IsZero() {
			builder.WriteString("\n  Published at: ")
			builder.WriteString(item.PublishedAt.UTC().Format(timeRFC3339))
		}
		builder.WriteString("\n")
	}
}

func rssReportItemSourceTitle(item RSSInboxItem) string {
	return firstNonEmptyString(
		strings.TrimSpace(item.SourceTitle),
		rssReportItemSourceHost(item.ItemLink),
		strings.TrimSpace(item.SourceFeedID),
		strings.TrimSpace(item.FeedID),
		"unknown-source",
	)
}

func rssReportItemSourceHost(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(host), "www.")
}

func writeRSSReportDossier(path string, dossier string) error {
	resolved := strings.TrimSpace(path)
	if resolved == "" {
		return fmt.Errorf("rss report dossier path is required")
	}
	if err := os.MkdirAll(filepath.Dir(resolved), rssReportDossierDirPerm); err != nil {
		return fmt.Errorf("create rss report dossier directory %q: %w", filepath.Dir(resolved), err)
	}
	body := strings.TrimSpace(dossier) + "\n"
	tempPath := fmt.Sprintf("%s.tmp-%d", resolved, time.Now().UnixNano())
	if err := os.WriteFile(tempPath, []byte(body), rssReportDossierFilePerm); err != nil {
		return fmt.Errorf("write temp rss report dossier %q: %w", tempPath, err)
	}
	if err := os.Rename(tempPath, resolved); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("replace rss report dossier %q: %w", resolved, err)
	}
	return nil
}

func rssReportDossierPath(rootDir string, reportID string) string {
	return filepath.Join(rootDir, "dossiers", strings.TrimSpace(reportID)+".source.md")
}
