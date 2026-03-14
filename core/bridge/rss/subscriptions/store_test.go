package subscriptions

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestFeedStoreUpsertListUpdateDelete(t *testing.T) {
	store, err := NewFeedStore(filepath.Join(t.TempDir(), "feeds.json"))
	if err != nil {
		t.Fatalf("NewFeedStore: %v", err)
	}
	baseNow := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return baseNow }

	created, action, err := store.Upsert(FeedUpsertInput{
		URL:        "https://example.com/feed.xml#frag",
		ProbeTitle: "Ghost Feed",
		Tags:       []string{"ai", "news", "ai"},
		Priority:   "high",
	})
	if err != nil {
		t.Fatalf("Upsert create: %v", err)
	}
	if action != "created" {
		t.Fatalf("unexpected create action: %q", action)
	}
	if created.URL != "https://example.com/feed.xml" {
		t.Fatalf("expected normalized url, got %q", created.URL)
	}
	if created.Title != "Ghost Feed" || created.Priority != "high" || !created.Enabled {
		t.Fatalf("unexpected created feed: %+v", created)
	}
	if len(created.Tags) != 2 || created.Tags[0] != "ai" || created.Tags[1] != "news" {
		t.Fatalf("unexpected created tags: %v", created.Tags)
	}

	store.now = func() time.Time { return baseNow.Add(2 * time.Hour) }
	enabled := false
	existing, action, err := store.Upsert(FeedUpsertInput{
		URL:     "https://example.com/feed.xml",
		Title:   "Ghost Feed Custom",
		Tags:    []string{"ops"},
		Enabled: &enabled,
	})
	if err != nil {
		t.Fatalf("Upsert existing: %v", err)
	}
	if action != "updated_existing" {
		t.Fatalf("unexpected existing action: %q", action)
	}
	if existing.Title != "Ghost Feed Custom" || existing.Enabled {
		t.Fatalf("unexpected updated existing feed: %+v", existing)
	}
	if len(existing.Tags) != 3 {
		t.Fatalf("expected merged tags, got %v", existing.Tags)
	}

	list, err := store.List(FeedListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("unexpected list result: %+v", list)
	}

	store.now = func() time.Time { return baseNow.Add(3 * time.Hour) }
	newTitle := ""
	tags := []string{"digest"}
	priority := "low"
	updated, err := store.Update(created.ID, FeedUpdatePatch{Title: &newTitle, Tags: &tags, Priority: &priority})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Title != "" || updated.Priority != "low" || len(updated.Tags) != 1 || updated.Tags[0] != "digest" {
		t.Fatalf("unexpected updated feed: %+v", updated)
	}

	filtered, err := store.List(FeedListFilter{Priority: "low", Enabled: &enabled, Tag: "digest"})
	if err != nil {
		t.Fatalf("List filtered: %v", err)
	}
	if len(filtered) != 1 || filtered[0].ID != created.ID {
		t.Fatalf("unexpected filtered result: %+v", filtered)
	}

	deleted, err := store.Delete(created.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !deleted {
		t.Fatal("expected delete to return true")
	}
	deleted, err = store.Delete(created.ID)
	if err != nil {
		t.Fatalf("Delete missing: %v", err)
	}
	if deleted {
		t.Fatal("expected second delete to return false")
	}

	_, err = store.Update(created.ID, FeedUpdatePatch{Priority: &priority})
	if !errors.Is(err, ErrFeedNotFound) {
		t.Fatalf("expected ErrFeedNotFound, got %v", err)
	}
}

func TestFeedStoreListSortsEnabledAndPriority(t *testing.T) {
	store, err := NewFeedStore(filepath.Join(t.TempDir(), "feeds.json"))
	if err != nil {
		t.Fatalf("NewFeedStore: %v", err)
	}
	baseNow := time.Date(2026, 3, 8, 9, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return baseNow }
	_, _, err = store.Upsert(FeedUpsertInput{URL: "https://example.com/one.xml", ProbeTitle: "One", Priority: "low"})
	if err != nil {
		t.Fatalf("Upsert first: %v", err)
	}
	store.now = func() time.Time { return baseNow.Add(time.Minute) }
	_, _, err = store.Upsert(FeedUpsertInput{URL: "https://example.com/two.xml", ProbeTitle: "Two", Priority: "high"})
	if err != nil {
		t.Fatalf("Upsert second: %v", err)
	}
	store.now = func() time.Time { return baseNow.Add(2 * time.Minute) }
	disabled := false
	last, _, err := store.Upsert(FeedUpsertInput{
		URL:        "https://example.com/three.xml",
		ProbeTitle: "Three",
		Enabled:    &disabled,
		Priority:   "high",
	})
	if err != nil {
		t.Fatalf("Upsert third: %v", err)
	}
	feeds, err := store.List(FeedListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(feeds) != 3 {
		t.Fatalf("unexpected feed count: %d", len(feeds))
	}
	if feeds[0].Priority != "high" || !feeds[0].Enabled {
		t.Fatalf("expected enabled high-priority feed first, got %+v", feeds[0])
	}
	if feeds[2].ID != last.ID || feeds[2].Enabled {
		t.Fatalf("expected disabled feed last, got %+v", feeds[2])
	}
}
