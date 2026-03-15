package rss

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

type rssReportBuilderFixture struct {
	builder        *agentRSSReportBuilder
	completer      *fakeReportCompleter
	report         RSSReportResult
	briefing       RSSBriefingResult
	groups         []RSSInboxTopicGroup
	query          RSSReportQuery
	scriptExecTool *fakeReportTool
	webSearchTool  *fakeReportTool
}

type rssReportToolState struct {
	completer      *fakeReportCompleter
	registry       *tools.Registry
	scriptExecTool *fakeReportTool
	webSearchTool  *fakeReportTool
}

func newRSSReportBuilderFixture(t *testing.T) rssReportBuilderFixture {
	t.Helper()
	report, briefing, groups := testRSSReportContext()
	dossierPath := writeTestRSSReportDossier(t, report, briefing, groups)
	toolState := newRSSReportToolState(t, dossierPath)
	query := RSSReportQuery{TraceID: "trace-rss-report", TaskID: "task-rss-report", DossierPath: dossierPath}
	builder := &agentRSSReportBuilder{
		timeout: testRSSReportTimeout,
		buildRuntime: func(*ConfigStore) (agentRuntimeDependencies, error) {
			return agentRuntimeDependencies{
				cfg:      Config{PromptsPath: "", MaxTurns: testRSSBuilderMaxTurns},
				client:   toolState.completer,
				registry: toolState.registry,
				cleanup:  func() {},
			}, nil
		},
		allowedTools: []string{"script_exec", "web_search"},
	}
	return rssReportBuilderFixture{
		builder:        builder,
		completer:      toolState.completer,
		report:         report,
		briefing:       briefing,
		groups:         groups,
		query:          query,
		scriptExecTool: toolState.scriptExecTool,
		webSearchTool:  toolState.webSearchTool,
	}
}

func testRSSReportContext() (RSSReportResult, RSSBriefingResult, []RSSInboxTopicGroup) {
	report := RSSReportResult{
		ID:         "rssr_test",
		BriefingID: "rssb_test",
		Title:      "AI Brief Report - 2026-03-09 18:00 UTC",
		TraceID:    "trace-rss-report",
		TaskID:     "task-rss-report",
	}
	briefing := RSSBriefingResult{
		ID:             "rssb_test",
		Title:          "AI Brief",
		Summary:        "One AI launch stands out.",
		GeneratedAt:    time.Date(2026, 3, 9, 18, 0, 0, 0, time.UTC),
		HighlightCount: testRSSSingleToolCallCount,
		Highlights: []RSSBriefingHighlight{{
			Rank: 1, GroupID: "group-1", Headline: "OpenAI launches GPT-6",
			Summary: "Launch summary", WhyItMatters: "Why this matters", Importance: "high",
		}},
	}
	groups := []RSSInboxTopicGroup{{
		ID: "group-1", TopicLabel: "ai / launch", Headline: "OpenAI launches GPT-6",
		Summary: "Group summary", Importance: "high", ItemCount: testRSSSingleToolCallCount, FeedCount: testRSSSingleToolCallCount,
		Items: []RSSInboxItem{{
			SourceTitle: "Example Feed", ItemTitle: "OpenAI launches GPT-6",
			ItemLink: "https://example.com/gpt-6", AISummary: "Launch summary", Importance: "high",
		}},
	}}
	return report, briefing, groups
}

func writeTestRSSReportDossier(
	t *testing.T,
	report RSSReportResult,
	briefing RSSBriefingResult,
	groups []RSSInboxTopicGroup,
) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), report.ID+".source.md")
	if err := writeRSSReportDossier(path, renderRSSReportDossier(report, briefing, groups)); err != nil {
		t.Fatalf("write dossier: %v", err)
	}
	return path
}

func newRSSReportToolState(t *testing.T, dossierPath string) rssReportToolState {
	t.Helper()
	scriptExecTool := &fakeReportTool{name: "script_exec", output: "dossier contents"}
	webSearchTool := &fakeReportTool{name: "web_search", output: "search results"}
	registry := tools.NewRegistry()
	registry.Register(scriptExecTool)
	registry.Register(webSearchTool)
	scriptArgs := mustMarshalJSON(t, map[string]any{
		"script": fmt.Sprintf("print(tools.read_file(path=%q))", dossierPath),
	})
	return rssReportToolState{
		registry:       registry,
		scriptExecTool: scriptExecTool,
		webSearchTool:  webSearchTool,
		completer: &fakeReportCompleter{responses: []*llm.CompletionResponse{
			reportToolCallsResponse(llm.ToolCall{ID: "call-read", Name: "script_exec", Arguments: scriptArgs}),
			reportToolCallsResponse(llm.ToolCall{ID: "call-search", Name: "web_search", Arguments: mustMarshalJSON(t, map[string]string{"query": "OpenAI GPT-6 launch reaction"})}),
			reportStopResponse("# Investigated Report\n\n## What happened\n\nValidated report."),
		}},
	}
}

func (f rssReportBuilderFixture) build(t *testing.T) string {
	t.Helper()
	markdown, err := f.builder.Build(context.Background(), f.report, f.briefing, f.groups, f.query)
	if err != nil {
		t.Fatalf("builder returned error: %v", err)
	}
	return markdown
}

func (f rssReportBuilderFixture) firstPrompt(t *testing.T) string {
	t.Helper()
	if len(f.completer.requests) == 0 {
		t.Fatal("expected at least one completion request")
	}
	messages := f.completer.requests[0].Messages
	if len(messages) == 0 {
		t.Fatal("expected completion request messages")
	}
	return messages[len(messages)-1].Text
}
