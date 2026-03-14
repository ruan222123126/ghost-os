package rss

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRSSReportStoreSaveAndLatest(t *testing.T) {
	store, err := NewRSSReportStore(filepath.Join(t.TempDir(), "reports", "index.json"))
	if err != nil {
		t.Fatalf("new report store: %v", err)
	}
	store.now = func() time.Time { return time.Date(2026, 3, 9, 13, 0, 0, 0, time.UTC) }

	saved, err := store.Save(RSSReportResult{
		BriefingID:     "rssb_123",
		Title:          "AI Brief Report",
		Summary:        "A compact report.",
		GeneratedAt:    time.Date(2026, 3, 9, 12, 45, 0, 0, time.UTC),
		TraceID:        "trace-rss-report",
		HighlightCount: 2,
		GroupCount:     1,
		SourceGroupIDs: []string{"group-1"},
	}, "# AI Brief Report\n\nReport body.")
	if err != nil {
		t.Fatalf("save report: %v", err)
	}
	if saved.ID == "" {
		t.Fatal("expected report id")
	}
	if saved.MarkdownPath == "" {
		t.Fatal("expected markdown path")
	}

	body, err := os.ReadFile(saved.MarkdownPath)
	if err != nil {
		t.Fatalf("read markdown: %v", err)
	}
	if string(body) != "# AI Brief Report\n\nReport body.\n" {
		t.Fatalf("unexpected markdown body: %q", string(body))
	}

	latest, err := store.Latest()
	if err != nil {
		t.Fatalf("latest report: %v", err)
	}
	if latest.ID != saved.ID {
		t.Fatalf("unexpected latest report id: got %q want %q", latest.ID, saved.ID)
	}
	if latest.MarkdownPath != saved.MarkdownPath {
		t.Fatalf("unexpected markdown path: got %q want %q", latest.MarkdownPath, saved.MarkdownPath)
	}
}

func TestRSSReportStoreLatestNotFound(t *testing.T) {
	store, err := NewRSSReportStore(filepath.Join(t.TempDir(), "reports", "index.json"))
	if err != nil {
		t.Fatalf("new report store: %v", err)
	}
	_, err = store.Latest()
	if !errors.Is(err, ErrRSSReportNotFound) {
		t.Fatalf("expected ErrRSSReportNotFound, got %v", err)
	}
}
