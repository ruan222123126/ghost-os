package rss

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestRSSBriefingStoreSaveAndLatest(t *testing.T) {
	store, err := NewRSSBriefingStore(filepath.Join(t.TempDir(), "briefings.json"))
	if err != nil {
		t.Fatalf("new briefing store: %v", err)
	}
	store.now = func() time.Time { return time.Date(2026, 3, 9, 12, 30, 0, 0, time.UTC) }

	saved, err := store.Save(RSSBriefingResult{
		Title:       "AI Brief",
		Summary:     "Top developments.",
		GeneratedAt: time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC),
		Highlights: []RSSBriefingHighlight{
			{Rank: 1, GroupID: "group-1", Headline: "Headline", Importance: "high", SourceItemCount: 2, SourceFeedCount: 1},
		},
	})
	if err != nil {
		t.Fatalf("save briefing: %v", err)
	}
	if saved.ID == "" {
		t.Fatal("expected saved briefing id")
	}
	if saved.SavedAt.IsZero() {
		t.Fatal("expected saved_at to be populated")
	}

	latest, err := store.Latest()
	if err != nil {
		t.Fatalf("latest briefing: %v", err)
	}
	if latest.ID != saved.ID {
		t.Fatalf("unexpected latest briefing: got %q want %q", latest.ID, saved.ID)
	}
	if latest.HighlightCount != 1 {
		t.Fatalf("unexpected highlight count: got %d want %d", latest.HighlightCount, 1)
	}
}

func TestRSSBriefingStoreLatestNotFound(t *testing.T) {
	store, err := NewRSSBriefingStore(filepath.Join(t.TempDir(), "briefings.json"))
	if err != nil {
		t.Fatalf("new briefing store: %v", err)
	}
	_, err = store.Latest()
	if !errors.Is(err, ErrRSSBriefingNotFound) {
		t.Fatalf("expected ErrRSSBriefingNotFound, got %v", err)
	}
}
