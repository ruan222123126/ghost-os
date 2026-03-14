package rss

import (
	"fmt"
	"strings"
)

const maxRSSReportInlineSources = 3

func finalizeRSSReportMarkdown(markdown string, briefing RSSBriefingResult, groups []RSSInboxTopicGroup) string {
	trimmed := strings.TrimSpace(markdown)
	if trimmed == "" {
		return ""
	}
	return injectRSSReportSourcesIntoWhatHappened(trimmed, briefing, groups)
}

func injectRSSReportSourcesIntoWhatHappened(markdown string, briefing RSSBriefingResult, groups []RSSInboxTopicGroup) string {
	if rssReportWhatHappenedHasInlineSources(markdown) {
		return markdown
	}
	sectionStart, sectionEnd, sectionText, ok := findRSSReportSection(markdown, "发生了什么", "What happened")
	if !ok {
		return markdown
	}
	sourceLines := buildRSSReportSourceLines(briefing, groups)
	if len(sourceLines) == 0 {
		return markdown
	}
	updated := injectRSSReportSourceLinesIntoSection(sectionText, briefing, sourceLines)
	if strings.TrimSpace(updated) == strings.TrimSpace(sectionText) {
		return markdown
	}
	return markdown[:sectionStart] + updated + markdown[sectionEnd:]
}

func buildRSSReportSourceLines(briefing RSSBriefingResult, groups []RSSInboxTopicGroup) []string {
	groupByID := make(map[string]RSSInboxTopicGroup, len(groups))
	for _, group := range groups {
		groupByID[group.ID] = group
	}
	sourceLines := make([]string, 0, len(briefing.Highlights))
	for _, highlight := range briefing.Highlights {
		group, ok := groupByID[strings.TrimSpace(highlight.GroupID)]
		if !ok {
			continue
		}
		if line := buildRSSReportSourceLine(group); line != "" {
			sourceLines = append(sourceLines, line)
		}
	}
	return sourceLines
}

func rssReportWhatHappenedHasInlineSources(markdown string) bool {
	_, _, section, ok := findRSSReportSection(markdown, "发生了什么", "What happened")
	return ok && strings.Contains(section, "出处：")
}

func findRSSReportSection(markdown string, names ...string) (int, int, string, bool) {
	lines := strings.SplitAfter(markdown, "\n")
	offset := 0
	start := -1
	end := len(markdown)
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if start < 0 {
			if rssReportSectionMatches(trimmed, names) {
				start = offset
			}
			offset += len(line)
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			return start, offset, markdown[start:offset], true
		}
		offset += len(line)
		if index == len(lines)-1 {
			end = len(markdown)
		}
	}
	if start < 0 {
		return 0, 0, "", false
	}
	return start, end, markdown[start:end], true
}

func rssReportSectionMatches(line string, names []string) bool {
	for _, name := range names {
		if line == "## "+name {
			return true
		}
	}
	return false
}

func injectRSSReportSourceLinesIntoSection(section string, briefing RSSBriefingResult, sourceLines []string) string {
	lines := strings.Split(section, "\n")
	if len(lines) == 0 {
		return section
	}
	inserts, nextSource := mapRSSReportSourceInsertions(lines, sourceLines)
	var builder strings.Builder
	for index, line := range lines {
		if index > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(line)
		if source, ok := inserts[index]; ok {
			if strings.TrimSpace(line) != "" {
				builder.WriteString("\n")
			}
			builder.WriteString(source)
		}
	}
	if nextSource < len(sourceLines) {
		builder.WriteString("\n\n")
		builder.WriteString(buildRSSReportFallbackEventBlocks(briefing, sourceLines[nextSource:]))
	}
	return strings.TrimSpace(builder.String())
}

func mapRSSReportSourceInsertions(lines []string, sourceLines []string) (map[int]string, int) {
	inserts := make(map[int]string, len(sourceLines))
	eventStarts := collectRSSReportEventBlockStarts(lines)
	nextSource := 0
	for index, start := range eventStarts {
		if nextSource >= len(sourceLines) {
			break
		}
		end := len(lines) - 1
		if index+1 < len(eventStarts) {
			end = eventStarts[index+1] - 1
		}
		inserts[rssReportEventInsertIndex(lines, start, end)] = sourceLines[nextSource]
		nextSource++
	}
	return inserts, nextSource
}

func collectRSSReportEventBlockStarts(lines []string) []int {
	starts := make([]int, 0, len(lines))
	for index := 1; index < len(lines); index++ {
		if isRSSReportEventBlockStart(lines[index]) {
			starts = append(starts, index)
		}
	}
	return starts
}

func rssReportEventInsertIndex(lines []string, start int, end int) int {
	for index := end; index >= start; index-- {
		if strings.TrimSpace(lines[index]) != "" {
			return index
		}
	}
	return start
}

func isRSSReportEventBlockStart(line string) bool {
	trimmed := strings.TrimSpace(line)
	switch {
	case trimmed == "":
		return false
	case strings.HasPrefix(trimmed, "**") && strings.HasSuffix(trimmed, "**"):
		return true
	default:
		return strings.HasPrefix(trimmed, "- **")
	}
}

func buildRSSReportFallbackEventBlocks(briefing RSSBriefingResult, sourceLines []string) string {
	if len(sourceLines) == 0 {
		return ""
	}
	var builder strings.Builder
	for index, sourceLine := range sourceLines {
		builder.WriteString("- **")
		builder.WriteString(rssReportFallbackEventTitle(briefing, index))
		builder.WriteString("**\n  ")
		builder.WriteString(sourceLine)
		if index+1 < len(sourceLines) {
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func rssReportFallbackEventTitle(briefing RSSBriefingResult, index int) string {
	title := fmt.Sprintf("事件 %d", index+1)
	if index >= len(briefing.Highlights) {
		return title
	}
	return firstNonEmptyString(briefing.Highlights[index].Headline, briefing.Highlights[index].TopicLabel, title)
}

func buildRSSReportSourceLine(group RSSInboxTopicGroup) string {
	items := dedupeRSSReportSourceItems(group.Items)
	if len(items) == 0 {
		return ""
	}
	if len(items) > maxRSSReportInlineSources {
		items = items[:maxRSSReportInlineSources]
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, formatRSSReportSourceItem(item))
	}
	return "出处：" + strings.Join(parts, "；")
}

func dedupeRSSReportSourceItems(items []RSSInboxItem) []RSSInboxItem {
	seen := make(map[string]struct{}, len(items))
	out := make([]RSSInboxItem, 0, len(items))
	for _, item := range items {
		key := rssReportSourceItemKey(item)
		if key == "" {
			key = fmt.Sprintf("untitled:%d", len(out))
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func rssReportSourceItemKey(item RSSInboxItem) string {
	if link := strings.TrimSpace(item.ItemLink); link != "" {
		return link
	}
	title := strings.TrimSpace(item.ItemTitle)
	source := rssReportItemSourceTitle(item)
	if title == "" && source == "" {
		return ""
	}
	return title + "::" + source
}

func formatRSSReportSourceItem(item RSSInboxItem) string {
	var builder strings.Builder
	if link := strings.TrimSpace(item.ItemLink); link != "" {
		builder.WriteString("[")
		builder.WriteString(firstNonEmptyString(item.ItemTitle, link))
		builder.WriteString("](")
		builder.WriteString(link)
		builder.WriteString(")")
	} else {
		builder.WriteString(firstNonEmptyString(item.ItemTitle, "Untitled source"))
	}
	builder.WriteString("（")
	builder.WriteString(rssReportItemSourceTitle(item))
	if !item.PublishedAt.IsZero() {
		builder.WriteString("，")
		builder.WriteString(item.PublishedAt.UTC().Format("2006-01-02"))
	}
	builder.WriteString("）")
	return builder.String()
}
