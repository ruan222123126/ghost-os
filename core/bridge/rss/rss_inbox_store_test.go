package rss

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestRSSInboxStoreSaveListGetAndFilter(t *testing.T) {
	store, err := NewRSSInboxStore(filepath.Join(t.TempDir(), "inbox.json"))
	if err != nil {
		t.Fatalf("operation failed: %v", err)
	}
	store.now = func() time.Time { return time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC) }

	items, err := store.SaveItems([]RSSInboxItem{
		{
			FeedID:       "feed-a",
			SourceFeedID: "feed-a",
			FeedURL:      "https://example.com/feed.xml",
			SourceTitle:  "Example Feed",
			ItemTitle:    "Alpha",
			ItemLink:     "https://example.com/a",
			PublishedAt:  time.Date(2026, 3, 8, 8, 0, 0, 0, time.UTC),
			AISummary:    "Alpha summary",
			Tags:         []string{"AI", "Launch"},
			Importance:   "high",
			Reason:       "Important update",
			ContentHash:  "hash-a",
			DedupeKey:    "feed-a::id::alpha",
		},
		{
			FeedID:       "feed-b",
			SourceFeedID: "feed-b",
			FeedURL:      "https://example.com/other.xml",
			SourceTitle:  "Other Feed",
			ItemTitle:    "Beta",
			ItemLink:     "https://example.com/b",
			PublishedAt:  time.Date(2026, 3, 7, 8, 0, 0, 0, time.UTC),
			AISummary:    "Beta summary",
			Tags:         []string{"ops"},
			Importance:   "low",
			Reason:       "Low priority",
			ContentHash:  "hash-b",
			DedupeKey:    "feed-b::id::beta",
			SavedAt:      time.Date(2026, 3, 8, 11, 0, 0, 0, time.UTC),
		},
	})
	if err != nil {
		t.Fatalf("operation failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("unexpected saved count: got %d want %d", len(items), 2)
	}

	dup, err := store.SaveItems([]RSSInboxItem{{
		FeedID:       "feed-a",
		SourceFeedID: "feed-a",
		FeedURL:      "https://example.com/feed.xml",
		ContentHash:  "hash-a2",
		DedupeKey:    "feed-a::id::alpha",
	}})
	if err != nil {
		t.Fatalf("operation failed: %v", err)
	}
	if len(dup) != 0 {
		t.Fatalf("expected duplicate save to be skipped, got %d", len(dup))
	}

	list, err := store.List(RSSInboxListFilter{})
	if err != nil {
		t.Fatalf("operation failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("unexpected list count: got %d want %d", len(list), 2)
	}
	if list[0].ItemTitle != "Alpha" {
		t.Fatalf("expected newest saved item first, got %q", list[0].ItemTitle)
	}
	if list[0].SavedAt.IsZero() {
		t.Fatal("expected saved_at to be populated")
	}

	filtered, err := store.List(RSSInboxListFilter{FeedID: "feed-a", Tag: "ai", Importance: "high"})
	if err != nil {
		t.Fatalf("operation failed: %v", err)
	}
	if len(filtered) != 1 || filtered[0].FeedID != "feed-a" {
		t.Fatalf("unexpected filtered items: %#v", filtered)
	}

	got, err := store.Get(items[0].ID)
	if err != nil {
		t.Fatalf("operation failed: %v", err)
	}
	if got.ID != items[0].ID {
		t.Fatalf("unexpected get id: got %q want %q", got.ID, items[0].ID)
	}

	_, err = store.Get("missing")
	if !errors.Is(err, ErrRSSInboxItemNotFound) {
		t.Fatalf("expected ErrRSSInboxItemNotFound, got %v", err)
	}
}
