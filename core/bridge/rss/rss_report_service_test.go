package rss

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRSSInboxServiceBuildAndStoreBriefingWritesReportMarkdown(t *testing.T) {
	now := time.Date(2026, 3, 9, 16, 0, 0, 0, time.UTC)
	fixture := newRSSReportServiceFixture(t, rssReportServiceOptions{
		now:           now,
		reportBuilder: stubRSSReportBuilder{markdown: "# AI Brief Report\n\n## What happened\n\nA new launch stands out."},
	})
	fixture.saveItems(t, []RSSInboxItem{testRSSInboxItem(rssTestItemOptions{
		now: now, title: "OpenAI launches GPT-6 coding agent",
		link: "https://example.com/gpt-6", summary: "A new coding-focused release landed.",
		publishedAt: now.Add(-2 * time.Hour),
	})})

	result := fixture.buildBriefing(t, "trace-rss-briefing-report", "task-rss-briefing-report")

	requireReportMetadata(t, result, now)
	requireFileContains(t, result.Report.MarkdownPath, "## What happened")
	requireFileContains(t, result.Report.MarkdownPath, "出处：[OpenAI launches GPT-6 coding agent](https://example.com/gpt-6)（Example Feed，2026-03-09）")
	dossierPath := rssReportDossierPath(filepath.Dir(fixture.reportStore.path), result.Report.ID)
	requireFileContains(t, dossierPath, "## Highlights")
	requireFileContains(t, dossierPath, "Source: Example Feed")
}

func TestRSSInboxServiceBuildAndStoreBriefingSurfacesReportFailure(t *testing.T) {
	now := time.Date(2026, 3, 9, 19, 0, 0, 0, time.UTC)
	fixture := newRSSReportServiceFixture(t, rssReportServiceOptions{
		now:           now,
		reportBuilder: stubRSSReportBuilder{err: errors.New("agent failed")},
	})
	fixture.saveItems(t, []RSSInboxItem{testRSSInboxItem(rssTestItemOptions{
		now: now, title: "Launch", link: "https://example.com/launch", summary: "Launch summary",
	})})

	result := fixture.buildBriefing(t, "trace-rss-briefing-report-fallback", "task-rss-briefing-report-fallback")

	if result.Report != nil {
		t.Fatalf("expected report metadata to be omitted on failure, got %+v", result.Report)
	}
	if strings.TrimSpace(result.ReportError) != "agent failed" {
		t.Fatalf("expected report error to be surfaced, got %q", result.ReportError)
	}
}

func TestRSSInboxServiceBuildAndStoreBriefingSurfacesEmptyReport(t *testing.T) {
	now := time.Date(2026, 3, 9, 20, 0, 0, 0, time.UTC)
	fixture := newRSSReportServiceFixture(t, rssReportServiceOptions{
		now:           now,
		reportBuilder: stubRSSReportBuilder{},
	})
	fixture.saveItems(t, []RSSInboxItem{testRSSInboxItem(rssTestItemOptions{
		now: now, title: "Launch", link: "https://example.com/launch", summary: "Launch summary",
	})})

	result := fixture.buildBriefing(t, "trace-rss-briefing-report-empty", "task-rss-briefing-report-empty")

	if result.Report != nil {
		t.Fatalf("expected report metadata to be omitted on empty report, got %+v", result.Report)
	}
	if strings.TrimSpace(result.ReportError) != "rss report builder returned empty content" {
		t.Fatalf("expected empty report error to be surfaced, got %q", result.ReportError)
	}
}
