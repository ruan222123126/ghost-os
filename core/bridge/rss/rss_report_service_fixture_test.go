package rss

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type rssReportServiceFixture struct {
	service     *RSSInboxService
	inboxStore  *RSSInboxStore
	reportStore *RSSReportStore
}

type rssReportServiceOptions struct {
	now           time.Time
	reportBuilder rssReportBuilder
}

type rssTestItemOptions struct {
	now         time.Time
	title       string
	link        string
	summary     string
	publishedAt time.Time
}

func newRSSReportServiceFixture(t *testing.T, opts rssReportServiceOptions) rssReportServiceFixture {
	t.Helper()
	if opts.reportBuilder == nil {
		t.Fatal("report builder is required")
	}
	rootDir := t.TempDir()
	inboxStore, err := NewRSSInboxStore(filepath.Join(rootDir, "inbox.json"))
	if err != nil {
		t.Fatalf("new inbox store: %v", err)
	}
	briefingStore, err := NewRSSBriefingStore(filepath.Join(rootDir, "briefings.json"))
	if err != nil {
		t.Fatalf("new briefing store: %v", err)
	}
	reportStore, err := NewRSSReportStore(filepath.Join(rootDir, "reports", "index.json"))
	if err != nil {
		t.Fatalf("new report store: %v", err)
	}
	return rssReportServiceFixture{
		service: &RSSInboxService{
			inboxStore:      inboxStore,
			briefingStore:   briefingStore,
			reportStore:     reportStore,
			briefingBuilder: stubRSSBriefingBuilder{draft: rssBriefingDraft{Title: "AI Brief", Summary: "One AI launch stands out in this cycle."}},
			reportBuilder:   opts.reportBuilder,
			now:             func() time.Time { return opts.now },
		},
		inboxStore:  inboxStore,
		reportStore: reportStore,
	}
}

func (f rssReportServiceFixture) saveItems(t *testing.T, items []RSSInboxItem) {
	t.Helper()
	if _, err := f.inboxStore.SaveItems(items); err != nil {
		t.Fatalf("save items: %v", err)
	}
}

func (f rssReportServiceFixture) buildBriefing(t *testing.T, traceID, taskID string) RSSBriefingResult {
	t.Helper()
	result, err := f.service.BuildAndStoreBriefing(context.Background(), testRSSBriefingQuery(traceID, taskID))
	if err != nil {
		t.Fatalf("build and store briefing returned error: %v", err)
	}
	return result
}

func testRSSBriefingQuery(traceID, taskID string) RSSBriefingQuery {
	return RSSBriefingQuery{
		WindowHours:     testRSSWindowHours,
		GroupLimit:      testRSSGroupLimit,
		ItemLimit:       testRSSItemLimit,
		ItemsPerGroup:   testRSSItemsPerGroup,
		HighlightsLimit: testRSSHighlightsLimit,
		TraceID:         traceID,
		TaskID:          taskID,
	}
}

func testRSSInboxItem(opts rssTestItemOptions) RSSInboxItem {
	key := strings.NewReplacer(" ", "-", "/", "-").Replace(strings.ToLower(strings.TrimSpace(opts.title)))
	return RSSInboxItem{
		FeedID:      "feed-ai",
		SourceTitle: "Example Feed",
		FeedURL:     "https://example.com/ai.xml",
		ItemTitle:   opts.title,
		ItemLink:    opts.link,
		AISummary:   opts.summary,
		Tags:        []string{"ai", "coding", "launch"},
		Importance:  "high",
		PublishedAt: opts.publishedAt,
		SavedAt:     opts.now.Add(-time.Minute),
		ContentHash: "hash-" + key,
		DedupeKey:   "feed-ai::" + key,
	}
}
