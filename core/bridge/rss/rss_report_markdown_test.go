package rss

import (
	"strings"
	"testing"
	"time"
)

func TestFinalizeRSSReportMarkdownInjectsSourcesInsideWhatHappened(t *testing.T) {
	briefing, groups, markdown := testFinalizeRSSReportMarkdownFixture()

	got := finalizeRSSReportMarkdown(markdown, briefing, groups)

	requireStringContains(t, got, "平台推出跨平台参与历史系统。\n出处：[Portable reputation infra](https://example.com/reputation)（Example Feed，2026-03-09）")
	requireStringContains(t, got, "研究指出 AI 可能加剧不平等。\n出处：[AI inequality paper](https://example.com/paper)（ArXiv，2026-03-08）")
	requireStringNotContains(t, got, "## 事件出处")
}

func testFinalizeRSSReportMarkdownFixture() (RSSBriefingResult, []RSSInboxTopicGroup, string) {
	briefing := RSSBriefingResult{Highlights: []RSSBriefingHighlight{
		{Rank: 1, GroupID: "group-1", Headline: "身份与声誉基础设施"},
		{Rank: 2, GroupID: "group-2", Headline: "AI经济学悖论"},
	}}
	groups := []RSSInboxTopicGroup{
		{ID: "group-1", Headline: "身份与声誉基础设施", Items: []RSSInboxItem{{
			SourceTitle: "Example Feed", ItemTitle: "Portable reputation infra",
			ItemLink: "https://example.com/reputation", PublishedAt: time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC),
		}}},
		{ID: "group-2", Headline: "AI经济学悖论", Items: []RSSInboxItem{{
			SourceTitle: "ArXiv", ItemTitle: "AI inequality paper",
			ItemLink: "https://example.com/paper", PublishedAt: time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC),
		}}},
	}
	markdown := strings.TrimSpace(`
# Report

## 发生了什么

**身份与声誉基础设施**
平台推出跨平台参与历史系统。

**AI经济学悖论**
研究指出 AI 可能加剧不平等。

## 为什么重要

值得跟踪。
`)
	return briefing, groups, markdown
}
