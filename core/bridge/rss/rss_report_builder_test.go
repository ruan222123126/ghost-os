package rss

import "testing"

func TestAgentRSSReportBuilderUsesOnlyScopedTools(t *testing.T) {
	fixture := newRSSReportBuilderFixture(t)

	markdown := fixture.build(t)

	requireStringContains(t, markdown, "## What happened")
	requireReportToolCallCount(t, fixture.scriptExecTool, testRSSSingleToolCallCount)
	requireReportToolCallCount(t, fixture.webSearchTool, testRSSSingleToolCallCount)
	if len(fixture.completer.requests) != testRSSBuildRequestCount {
		t.Fatalf("expected %d completion requests, got %d", testRSSBuildRequestCount, len(fixture.completer.requests))
	}
	requireToolDefNames(t, fixture.completer.requests[0].Tools, []string{"script_exec", "web_search"})
}

func TestAgentRSSReportBuilderPromptIncludesDossierAndScopedGuidance(t *testing.T) {
	fixture := newRSSReportBuilderFixture(t)

	fixture.build(t)
	prompt := fixture.firstPrompt(t)

	requireStringContains(t, prompt, fixture.query.DossierPath)
	requireStringContains(t, prompt, "Simplified Chinese")
	requireStringContains(t, prompt, "Use this exact H1 title")
	requireStringContains(t, prompt, "Do not move sources into a separate appendix section")
	requireStringContains(t, prompt, "Investigation tools for this run: `script_exec`, `web_search`.")
	requireStringNotContains(t, prompt, "`read_and_summarize`")
	requireStringNotContains(t, prompt, "`rss_fetch`")
}
