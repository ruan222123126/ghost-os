package rss

import "testing"

func TestRenderAgentRSSReportPromptRequiresChinese(t *testing.T) {
	prompt := renderAgentRSSReportPrompt(
		RSSReportResult{ID: "rssr_test", Title: "RSS Report - 2026-03-09 12:00 UTC"},
		RSSBriefingResult{ID: "rssb_test"},
		nil,
		RSSReportQuery{DossierPath: "/tmp/rssr_test.source.md"},
		"Use available tools when needed to validate important claims.",
	)

	requireStringContains(t, prompt, "Simplified Chinese")
	requireStringContains(t, prompt, "RSS Report - 2026-03-09 12:00 UTC")
	requireStringContains(t, prompt, "Do not move sources into a separate appendix section")
	requireStringContains(t, prompt, "Use available tools when needed to validate important claims.")
}

func TestRenderAgentRSSReportPromptFallsBackToDefaultToolGuidance(t *testing.T) {
	prompt := renderAgentRSSReportPrompt(
		RSSReportResult{ID: "rssr_test", Title: "RSS Report - 2026-03-09 12:00 UTC"},
		RSSBriefingResult{ID: "rssb_test"},
		nil,
		RSSReportQuery{DossierPath: "/tmp/rssr_test.source.md"},
		"",
	)

	requireStringContains(t, prompt, "Use the currently available tools when needed to validate important claims, inspect primary sources, and add missing context.")
}
