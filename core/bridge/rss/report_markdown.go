package rss

import "strings"

func fallbackRSSReportMarkdown(report RSSReportResult, briefing RSSBriefingResult, groups []RSSInboxTopicGroup) string {
	var builder strings.Builder
	writeRSSReportTitleBlock(&builder, report, briefing)
	writeRSSReportWhatHappenedSection(&builder, briefing)
	writeRSSReportWhyItMattersSection(&builder, briefing)
	writeRSSReportOpportunitiesSection(&builder, briefing)
	writeRSSReportRisksSection(&builder, briefing)
	writeRSSReportNextSection(&builder, briefing)
	writeRSSReportSignalsSection(&builder, groups)
	return strings.TrimSpace(builder.String())
}

func writeRSSReportTitleBlock(builder *strings.Builder, report RSSReportResult, briefing RSSBriefingResult) {
	title := strings.TrimSpace(report.Title)
	if title == "" {
		title = rssReportTitleOrDefault(briefing.Title, report.GeneratedAt)
	}
	builder.WriteString("# ")
	builder.WriteString(title)
	builder.WriteString("\n\n")
	if briefing.GeneratedAt.IsZero() {
		return
	}
	builder.WriteString("_Generated: ")
	builder.WriteString(briefing.GeneratedAt.UTC().Format(timeRFC3339))
	builder.WriteString("_\n\n")
}

func writeRSSReportWhatHappenedSection(builder *strings.Builder, briefing RSSBriefingResult) {
	writeRSSReportSectionHeading(builder, "发生了什么")
	if summary := strings.TrimSpace(briefing.Summary); summary != "" {
		builder.WriteString(summary)
		builder.WriteString("\n\n")
	}
	for _, highlight := range briefing.Highlights {
		builder.WriteString("- **")
		builder.WriteString(firstNonEmptyString(highlight.Headline, highlight.TopicLabel, "Untitled signal"))
		builder.WriteString("**")
		if summary := firstNonEmptyString(highlight.Summary, highlight.WhyItMatters); summary != "" {
			builder.WriteString(": ")
			builder.WriteString(summary)
		}
		builder.WriteString("\n")
	}
	builder.WriteString("\n")
}

func writeRSSReportWhyItMattersSection(builder *strings.Builder, briefing RSSBriefingResult) {
	writeRSSReportSectionHeading(builder, "为什么重要")
	for _, highlight := range briefing.Highlights {
		builder.WriteString("- **")
		builder.WriteString(firstNonEmptyString(highlight.TopicLabel, highlight.Headline, "Signal"))
		builder.WriteString("**: ")
		builder.WriteString(firstNonEmptyString(
			highlight.WhyItMatters,
			highlight.Summary,
			"这个主题在当前 RSS 时间窗内出现了足够多的重复信号，值得重点关注。",
		))
		builder.WriteString("\n")
	}
	builder.WriteString("\n")
}

func writeRSSReportOpportunitiesSection(builder *strings.Builder, briefing RSSBriefingResult) {
	writeRSSReportSectionHeading(builder, "机会点")
	for _, highlight := range briefing.Highlights {
		builder.WriteString("- 跟踪 **")
		builder.WriteString(firstNonEmptyString(highlight.TopicLabel, highlight.Headline, "this topic"))
		builder.WriteString("** 是否会在近期带来产品、合作或分发层面的机会。\n")
	}
	builder.WriteString("\n")
}

func writeRSSReportRisksSection(builder *strings.Builder, briefing RSSBriefingResult) {
	writeRSSReportSectionHeading(builder, "风险与约束")
	builder.WriteString("- 本报告基于当前时间窗内的 RSS 覆盖生成，可能遗漏后续更正或站外上下文。\n")
	builder.WriteString("- 任何业务、运营或投资动作仍应回到一手来源做确认。\n")
	for _, highlight := range briefing.Highlights {
		builder.WriteString("- **")
		builder.WriteString(firstNonEmptyString(highlight.TopicLabel, highlight.Headline, "Signal"))
		builder.WriteString("** 仍可能处于早期信号阶段，需要继续等待更多来源确认。\n")
	}
	builder.WriteString("\n")
}

func writeRSSReportNextSection(builder *strings.Builder, briefing RSSBriefingResult) {
	writeRSSReportSectionHeading(builder, "接下来可能会怎样")
	for _, highlight := range briefing.Highlights {
		builder.WriteString("- 预计 **")
		builder.WriteString(firstNonEmptyString(highlight.TopicLabel, highlight.Headline, "this topic"))
		builder.WriteString("** 如果继续累积信号，下一轮里大概率会出现后续报道。\n")
	}
	builder.WriteString("\n")
}

func writeRSSReportSignalsSection(builder *strings.Builder, groups []RSSInboxTopicGroup) {
	writeRSSReportSectionHeading(builder, "值得持续关注的具体信号")
	for _, group := range groups {
		builder.WriteString("### ")
		builder.WriteString(firstNonEmptyString(group.Headline, group.TopicLabel, group.ID))
		builder.WriteString("\n")
		writeRSSReportGroupItems(builder, group.Items)
		builder.WriteString("\n")
	}
}

func writeRSSReportGroupItems(builder *strings.Builder, items []RSSInboxItem) {
	for _, item := range items {
		builder.WriteString("- ")
		if link := strings.TrimSpace(item.ItemLink); link != "" {
			builder.WriteString("[")
			builder.WriteString(firstNonEmptyString(item.ItemTitle, link))
			builder.WriteString("](")
			builder.WriteString(link)
			builder.WriteString(")")
		} else {
			builder.WriteString(firstNonEmptyString(item.ItemTitle, "Untitled source"))
		}
		if summary := rssInboxItemSummary(item); summary != "" {
			builder.WriteString(": ")
			builder.WriteString(summary)
		}
		builder.WriteString("\n")
	}
}

func writeRSSReportSectionHeading(builder *strings.Builder, title string) {
	builder.WriteString("## ")
	builder.WriteString(title)
	builder.WriteString("\n\n")
}
