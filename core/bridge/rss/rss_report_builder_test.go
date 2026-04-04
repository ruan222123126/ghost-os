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
	if len(fixture.completer.requests[0].Tools) != testRSSScopedToolCount {
		t.Fatalf("expected %d scoped tools, got %d", testRSSScopedToolCount, len(fixture.completer.requests[0].Tools))
	}
	requireToolDefNames(t, fixture.completer.requests[0].Tools, []string{"script_exec", "web_search"})
}

func TestAgentRSSReportBuilderPromptIncludesDossierAndWritingContract(t *testing.T) {
	fixture := newRSSReportBuilderFixture(t)

	fixture.build(t)
	prompt := fixture.firstPrompt(t)

	requireStringContains(t, prompt, fixture.query.DossierPath)
	requireStringContains(t, prompt, "Simplified Chinese")
	requireStringContains(t, prompt, "Use this exact H1 title")
	requireStringContains(t, prompt, "Do not move sources into a separate appendix section")
	requireStringContains(t, prompt, "Do not add standalone sections for opportunities, risks, constraints, or predictions")
	requireStringContains(t, prompt, "avoid repeating the same point across sections")
	requireStringNotContains(t, prompt, "- Opportunities")
	requireStringNotContains(t, prompt, "- Risks / constraints")
	requireStringNotContains(t, prompt, "- What may happen next")
}

func TestAgentRSSReportBuilderPromptReflectsScopedToolVisibility(t *testing.T) {
	fixture := newRSSReportBuilderFixture(t)

	fixture.build(t)
	prompt := fixture.firstPrompt(t)

	requireStringContains(t, prompt, "Investigation tools for this run: `script_exec`, `web_search`.")
	requireStringNotContains(t, prompt, "`read_and_summarize`")
	requireStringNotContains(t, prompt, "`rss_fetch`")
}
