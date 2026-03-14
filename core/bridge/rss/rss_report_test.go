package rss

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

type stubRSSBriefingBuilder struct {
	draft rssBriefingDraft
	err   error
}

func (b stubRSSBriefingBuilder) Build(context.Context, []RSSInboxTopicGroup, RSSBriefingQuery) (rssBriefingDraft, error) {
	return b.draft, b.err
}

type stubRSSReportBuilder struct {
	markdown string
	err      error
}

func (b stubRSSReportBuilder) Build(context.Context, RSSReportResult, RSSBriefingResult, []RSSInboxTopicGroup, RSSReportQuery) (string, error) {
	return b.markdown, b.err
}

type fakeReportCompleter struct {
	responses []*llm.CompletionResponse
	requests  []llm.CompletionRequest
}

func (f *fakeReportCompleter) Complete(_ context.Context, request llm.CompletionRequest) (*llm.CompletionResponse, error) {
	f.requests = append(f.requests, cloneReportCompletionRequest(request))
	if len(f.responses) == 0 {
		return nil, errors.New("unexpected complete call")
	}
	response := f.responses[0]
	f.responses = f.responses[1:]
	return response, nil
}

func cloneReportCompletionRequest(request llm.CompletionRequest) llm.CompletionRequest {
	return llm.CompletionRequest{
		Messages: llm.CloneMessages(request.Messages),
		Tools:    append([]llm.ToolDef(nil), request.Tools...),
	}
}

type fakeReportTool struct {
	name      string
	output    string
	callCount int
	lastArgs  json.RawMessage
}

func (f *fakeReportTool) Name() string { return f.name }

func (f *fakeReportTool) Description() string { return "fake report tool" }

func (f *fakeReportTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}

func (f *fakeReportTool) Execute(_ context.Context, argsJSON json.RawMessage, _ string) (string, error) {
	f.callCount++
	f.lastArgs = append([]byte(nil), argsJSON...)
	return f.output, nil
}

func reportToolCallsResponse(calls ...llm.ToolCall) *llm.CompletionResponse {
	return &llm.CompletionResponse{
		Message:      llm.Message{Role: llm.RoleAssistant, ToolCalls: calls},
		FinishReason: llm.FinishToolCalls,
	}
}

func reportStopResponse(text string) *llm.CompletionResponse {
	return &llm.CompletionResponse{
		Message:      llm.Message{Role: llm.RoleAssistant, Text: text},
		FinishReason: llm.FinishStop,
	}
}

func TestRSSInboxServiceBuildAndStoreBriefingWritesReportMarkdown(t *testing.T) {
	inboxStore, err := NewRSSInboxStore(filepath.Join(t.TempDir(), "inbox.json"))
	if err != nil {
		t.Fatalf("new inbox store: %v", err)
	}
	briefingStore, err := NewRSSBriefingStore(filepath.Join(t.TempDir(), "briefings.json"))
	if err != nil {
		t.Fatalf("new briefing store: %v", err)
	}
	reportStore, err := NewRSSReportStore(filepath.Join(t.TempDir(), "reports", "index.json"))
	if err != nil {
		t.Fatalf("new report store: %v", err)
	}
	now := time.Date(2026, 3, 9, 16, 0, 0, 0, time.UTC)
	_, err = inboxStore.SaveItems([]RSSInboxItem{
		{
			FeedID:      "feed-ai",
			SourceTitle: "Example Feed",
			FeedURL:     "https://example.com/ai.xml",
			ItemTitle:   "OpenAI launches GPT-6 coding agent",
			ItemLink:    "https://example.com/gpt-6",
			AISummary:   "A new coding-focused release landed.",
			Tags:        []string{"ai", "coding", "launch"},
			Importance:  "high",
			PublishedAt: now.Add(-2 * time.Hour),
			SavedAt:     now.Add(-90 * time.Minute),
			ContentHash: "hash-1",
			DedupeKey:   "feed-ai::1",
		},
	})
	if err != nil {
		t.Fatalf("save items: %v", err)
	}

	service := &RSSInboxService{
		inboxStore:      inboxStore,
		briefingStore:   briefingStore,
		reportStore:     reportStore,
		briefingBuilder: stubRSSBriefingBuilder{draft: rssBriefingDraft{Title: "AI Brief", Summary: "One AI launch stands out in this cycle."}},
		reportBuilder:   stubRSSReportBuilder{markdown: "# AI Brief Report\n\n## What happened\n\nA new launch stands out."},
		now:             func() time.Time { return now },
	}

	result, err := service.BuildAndStoreBriefing(context.Background(), RSSBriefingQuery{
		WindowHours:     24,
		GroupLimit:      5,
		ItemLimit:       10,
		ItemsPerGroup:   3,
		HighlightsLimit: 3,
		TraceID:         "trace-rss-briefing-report",
		TaskID:          "task-rss-briefing-report",
	})
	if err != nil {
		t.Fatalf("build and store briefing returned error: %v", err)
	}
	if result.Report == nil {
		t.Fatal("expected report metadata")
	}
	if result.Report.BriefingID != result.ID {
		t.Fatalf("unexpected report briefing id: got %q want %q", result.Report.BriefingID, result.ID)
	}
	if !strings.Contains(result.Report.Title, "2026-03-09 16:00 UTC") {
		t.Fatalf("expected report title to include generated time, got %q", result.Report.Title)
	}
	if !strings.HasSuffix(result.Report.MarkdownPath, ".md") {
		t.Fatalf("expected markdown path, got %q", result.Report.MarkdownPath)
	}
	body, err := os.ReadFile(result.Report.MarkdownPath)
	if err != nil {
		t.Fatalf("read report markdown: %v", err)
	}
	if !strings.Contains(string(body), "## What happened") {
		t.Fatalf("unexpected report body: %q", string(body))
	}
	if !strings.Contains(string(body), "出处：[OpenAI launches GPT-6 coding agent](https://example.com/gpt-6)（Example Feed，2026-03-09）") {
		t.Fatalf("expected report body to inline source inside what happened, got %q", string(body))
	}

	dossierPath := rssReportDossierPath(filepath.Dir(reportStore.path), result.Report.ID)
	dossier, err := os.ReadFile(dossierPath)
	if err != nil {
		t.Fatalf("read dossier: %v", err)
	}
	if !strings.Contains(string(dossier), "## Highlights") {
		t.Fatalf("unexpected dossier body: %q", string(dossier))
	}
	if !strings.Contains(string(dossier), "Source: Example Feed") {
		t.Fatalf("expected dossier to include source title, got %q", string(dossier))
	}
}

func TestAgentRSSReportBuilderUsesScopedToolsAndDossier(t *testing.T) {
	dossierDir := t.TempDir()
	report := RSSReportResult{ID: "rssr_test", BriefingID: "rssb_test", TraceID: "trace-rss-report", TaskID: "task-rss-report"}
	briefing := RSSBriefingResult{
		ID:             "rssb_test",
		Title:          "AI Brief",
		Summary:        "One AI launch stands out.",
		GeneratedAt:    time.Date(2026, 3, 9, 18, 0, 0, 0, time.UTC),
		HighlightCount: 1,
		Highlights: []RSSBriefingHighlight{
			{Rank: 1, GroupID: "group-1", Headline: "OpenAI launches GPT-6", Summary: "Launch summary", WhyItMatters: "Why this matters", Importance: "high"},
		},
	}
	groups := []RSSInboxTopicGroup{
		{
			ID:         "group-1",
			TopicLabel: "ai / launch",
			Headline:   "OpenAI launches GPT-6",
			Summary:    "Group summary",
			Importance: "high",
			ItemCount:  1,
			FeedCount:  1,
			Items: []RSSInboxItem{
				{SourceTitle: "Example Feed", ItemTitle: "OpenAI launches GPT-6", ItemLink: "https://example.com/gpt-6", AISummary: "Launch summary", Importance: "high"},
			},
		},
	}
	dossierPath := filepath.Join(dossierDir, "rssr_test.source.md")
	if err := writeRSSReportDossier(dossierPath, renderRSSReportDossier(report, briefing, groups)); err != nil {
		t.Fatalf("write dossier: %v", err)
	}

	scriptExecTool := &fakeReportTool{name: "script_exec", output: "dossier contents"}
	webSearchTool := &fakeReportTool{name: "web_search", output: "search results"}
	registry := tools.NewRegistry()
	registry.Register(scriptExecTool)
	registry.Register(webSearchTool)
	scriptArgs, err := json.Marshal(map[string]any{
		"script": fmt.Sprintf("print(tools.read_file(path=%q))", dossierPath),
	})
	if err != nil {
		t.Fatalf("marshal script args: %v", err)
	}
	completer := &fakeReportCompleter{
		responses: []*llm.CompletionResponse{
			reportToolCallsResponse(llm.ToolCall{ID: "call-read", Name: "script_exec", Arguments: json.RawMessage(scriptArgs)}),
			reportToolCallsResponse(llm.ToolCall{ID: "call-search", Name: "web_search", Arguments: json.RawMessage(`{"query":"OpenAI GPT-6 launch reaction"}`)}),
			reportStopResponse("# Investigated Report\n\n## What happened\n\nValidated report."),
		},
	}

	builder := &agentRSSReportBuilder{
		timeout: 5 * time.Second,
		buildRuntime: func(*ConfigStore) (agentRuntimeDependencies, error) {
			return agentRuntimeDependencies{
				cfg:      Config{PromptsPath: "", MaxTurns: 6},
				client:   completer,
				registry: registry,
				cleanup:  func() {},
			}, nil
		},
		allowedTools: []string{"script_exec", "web_search"},
	}

	markdown, err := builder.Build(context.Background(), report, briefing, groups, RSSReportQuery{
		TraceID:     "trace-rss-report",
		TaskID:      "task-rss-report",
		DossierPath: dossierPath,
	})
	if err != nil {
		t.Fatalf("builder returned error: %v", err)
	}
	if !strings.Contains(markdown, "## What happened") {
		t.Fatalf("unexpected markdown: %q", markdown)
	}
	if scriptExecTool.callCount != 1 {
		t.Fatalf("expected script_exec to be called once, got %d", scriptExecTool.callCount)
	}
	if webSearchTool.callCount != 1 {
		t.Fatalf("expected web_search to be called once, got %d", webSearchTool.callCount)
	}
	if len(completer.requests) != 3 {
		t.Fatalf("expected 3 completion requests, got %d", len(completer.requests))
	}
	if got := completer.requests[0].Messages[len(completer.requests[0].Messages)-1].Text; !strings.Contains(got, dossierPath) {
		t.Fatalf("expected prompt to include dossier path, got %q", got)
	}
	if got := completer.requests[0].Messages[len(completer.requests[0].Messages)-1].Text; !strings.Contains(got, "Simplified Chinese") {
		t.Fatalf("expected prompt to require simplified chinese, got %q", got)
	}
	if got := completer.requests[0].Messages[len(completer.requests[0].Messages)-1].Text; !strings.Contains(got, "Use this exact H1 title") {
		t.Fatalf("expected prompt to require exact title, got %q", got)
	}
	if got := completer.requests[0].Messages[len(completer.requests[0].Messages)-1].Text; !strings.Contains(got, "Do not move sources into a separate appendix section") {
		t.Fatalf("expected prompt to require inline event sources, got %q", got)
	}
	lastTools := completer.requests[0].Tools
	if len(lastTools) != 2 {
		t.Fatalf("expected scoped tools, got %d tool defs", len(lastTools))
	}
}

func TestRSSInboxServiceBuildAndStoreBriefingFallsBackWhenAgentReportFails(t *testing.T) {
	inboxStore, err := NewRSSInboxStore(filepath.Join(t.TempDir(), "inbox.json"))
	if err != nil {
		t.Fatalf("new inbox store: %v", err)
	}
	briefingStore, err := NewRSSBriefingStore(filepath.Join(t.TempDir(), "briefings.json"))
	if err != nil {
		t.Fatalf("new briefing store: %v", err)
	}
	reportStore, err := NewRSSReportStore(filepath.Join(t.TempDir(), "reports", "index.json"))
	if err != nil {
		t.Fatalf("new report store: %v", err)
	}
	now := time.Date(2026, 3, 9, 19, 0, 0, 0, time.UTC)
	_, err = inboxStore.SaveItems([]RSSInboxItem{{
		FeedID:      "feed-ai",
		SourceTitle: "Example Feed",
		FeedURL:     "https://example.com/ai.xml",
		ItemTitle:   "Launch",
		ItemLink:    "https://example.com/launch",
		AISummary:   "Launch summary",
		Importance:  "high",
		ContentHash: "hash-1",
		DedupeKey:   "feed-ai::1",
		SavedAt:     now.Add(-time.Minute),
	}})
	if err != nil {
		t.Fatalf("save items: %v", err)
	}

	service := &RSSInboxService{
		inboxStore:      inboxStore,
		briefingStore:   briefingStore,
		reportStore:     reportStore,
		briefingBuilder: stubRSSBriefingBuilder{draft: rssBriefingDraft{Title: "AI Brief", Summary: "One AI launch stands out in this cycle."}},
		reportBuilder:   stubRSSReportBuilder{err: errors.New("agent failed")},
		now:             func() time.Time { return now },
	}

	result, err := service.BuildAndStoreBriefing(context.Background(), RSSBriefingQuery{
		WindowHours:     24,
		GroupLimit:      5,
		ItemLimit:       10,
		ItemsPerGroup:   3,
		HighlightsLimit: 3,
		TraceID:         "trace-rss-briefing-report-fallback",
		TaskID:          "task-rss-briefing-report-fallback",
	})
	if err != nil {
		t.Fatalf("build and store briefing returned error: %v", err)
	}
	if result.Report == nil {
		t.Fatal("expected fallback report metadata")
	}
	if strings.TrimSpace(result.ReportError) != "" {
		t.Fatalf("expected fallback to suppress report error, got %q", result.ReportError)
	}
	body, err := os.ReadFile(result.Report.MarkdownPath)
	if err != nil {
		t.Fatalf("read fallback report markdown: %v", err)
	}
	if !strings.Contains(string(body), "# AI Brief Report - 2026-03-09 19:00 UTC") {
		t.Fatalf("expected fallback title to include generated time, got %q", string(body))
	}
	if !strings.Contains(string(body), "## 值得持续关注的具体信号") {
		t.Fatalf("unexpected fallback markdown: %q", string(body))
	}
	if !strings.Contains(string(body), "出处：[Launch](https://example.com/launch)（Example Feed）") {
		t.Fatalf("expected fallback markdown to inline source in what happened, got %q", string(body))
	}
}

func TestRenderAgentRSSReportPromptRequiresChinese(t *testing.T) {
	prompt := renderAgentRSSReportPrompt(
		RSSReportResult{ID: "rssr_test", Title: "RSS Report - 2026-03-09 12:00 UTC"},
		RSSBriefingResult{ID: "rssb_test"},
		nil,
		RSSReportQuery{DossierPath: "/tmp/rssr_test.source.md"},
	)
	if !strings.Contains(prompt, "Simplified Chinese") {
		t.Fatalf("expected prompt to require simplified chinese, got %q", prompt)
	}
	if !strings.Contains(prompt, "RSS Report - 2026-03-09 12:00 UTC") {
		t.Fatalf("expected prompt to include exact report title, got %q", prompt)
	}
	if !strings.Contains(prompt, "Do not move sources into a separate appendix section") {
		t.Fatalf("expected prompt to require inline sources, got %q", prompt)
	}
}

func TestFinalizeRSSReportMarkdownInjectsSourcesInsideWhatHappened(t *testing.T) {
	briefing := RSSBriefingResult{
		Highlights: []RSSBriefingHighlight{
			{Rank: 1, GroupID: "group-1", Headline: "身份与声誉基础设施"},
			{Rank: 2, GroupID: "group-2", Headline: "AI经济学悖论"},
		},
	}
	groups := []RSSInboxTopicGroup{
		{
			ID:       "group-1",
			Headline: "身份与声誉基础设施",
			Items: []RSSInboxItem{{
				SourceTitle: "Example Feed",
				ItemTitle:   "Portable reputation infra",
				ItemLink:    "https://example.com/reputation",
				PublishedAt: time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC),
			}},
		},
		{
			ID:       "group-2",
			Headline: "AI经济学悖论",
			Items: []RSSInboxItem{{
				SourceTitle: "ArXiv",
				ItemTitle:   "AI inequality paper",
				ItemLink:    "https://example.com/paper",
				PublishedAt: time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC),
			}},
		},
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
	got := finalizeRSSReportMarkdown(markdown, briefing, groups)
	if !strings.Contains(got, "平台推出跨平台参与历史系统。\n出处：[Portable reputation infra](https://example.com/reputation)（Example Feed，2026-03-09）") {
		t.Fatalf("expected first event to include inline source, got %q", got)
	}
	if !strings.Contains(got, "研究指出 AI 可能加剧不平等。\n出处：[AI inequality paper](https://example.com/paper)（ArXiv，2026-03-08）") {
		t.Fatalf("expected second event to include inline source, got %q", got)
	}
	if strings.Contains(got, "## 事件出处") {
		t.Fatalf("expected no appendix sources section, got %q", got)
	}
}
