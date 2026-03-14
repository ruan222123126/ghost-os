package rss

import (
	"path/filepath"
	"testing"
	"time"
)

func TestRSSInboxServiceAggregateGroupsByTopicWindowAndImportance(t *testing.T) {
	inboxStore, err := NewRSSInboxStore(filepath.Join(t.TempDir(), "inbox.json"))
	if err != nil {
		t.Fatalf("new inbox store: %v", err)
	}
	now := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	_, err = inboxStore.SaveItems([]RSSInboxItem{
		{
			FeedID:      "feed-ai",
			FeedURL:     "https://example.com/ai.xml",
			ItemTitle:   "OpenAI launches GPT-6 coding agent",
			AISummary:   "New coding-focused model rollout.",
			Tags:        []string{"ai", "openai", "launch"},
			Importance:  "high",
			PublishedAt: time.Date(2026, 3, 9, 10, 15, 0, 0, time.UTC),
			SavedAt:     time.Date(2026, 3, 9, 10, 20, 0, 0, time.UTC),
			ContentHash: "hash-1",
			DedupeKey:   "feed-ai::1",
		},
		{
			FeedID:      "feed-ai",
			FeedURL:     "https://example.com/ai.xml",
			ItemTitle:   "GPT-6 agent launch expands coding workflow",
			AISummary:   "Companion launch details for the same release.",
			Tags:        []string{"ai", "launch", "coding"},
			Importance:  "normal",
			PublishedAt: time.Date(2026, 3, 9, 11, 0, 0, 0, time.UTC),
			SavedAt:     time.Date(2026, 3, 9, 11, 5, 0, 0, time.UTC),
			ContentHash: "hash-2",
			DedupeKey:   "feed-ai::2",
		},
		{
			FeedID:      "feed-sec",
			FeedURL:     "https://example.com/sec.xml",
			ItemTitle:   "Chrome ships emergency security update",
			AISummary:   "Patch fixes an actively exploited issue.",
			Tags:        []string{"security", "chrome"},
			Importance:  "high",
			PublishedAt: time.Date(2026, 3, 9, 18, 0, 0, 0, time.UTC),
			SavedAt:     time.Date(2026, 3, 9, 18, 2, 0, 0, time.UTC),
			ContentHash: "hash-3",
			DedupeKey:   "feed-sec::1",
		},
		{
			FeedID:      "feed-old",
			FeedURL:     "https://example.com/old.xml",
			ItemTitle:   "Old news outside the window",
			AISummary:   "Should be excluded.",
			Tags:        []string{"archive"},
			Importance:  "high",
			PublishedAt: time.Date(2026, 3, 7, 9, 0, 0, 0, time.UTC),
			SavedAt:     time.Date(2026, 3, 7, 9, 1, 0, 0, time.UTC),
			ContentHash: "hash-4",
			DedupeKey:   "feed-old::1",
		},
	})
	if err != nil {
		t.Fatalf("save items: %v", err)
	}

	service := &RSSInboxService{
		inboxStore: inboxStore,
		now:        func() time.Time { return now },
	}

	result, err := service.Aggregate(RSSInboxGroupQuery{
		WindowHours:   24,
		Limit:         10,
		ItemLimit:     20,
		ItemsPerGroup: 3,
	})
	if err != nil {
		t.Fatalf("aggregate returned error: %v", err)
	}
	if result.ScannedItems != 3 {
		t.Fatalf("unexpected scanned items: got %d want %d", result.ScannedItems, 3)
	}
	if len(result.Groups) != 2 {
		t.Fatalf("unexpected group count: got %d want %d", len(result.Groups), 2)
	}

	var launchGroup RSSInboxTopicGroup
	if result.Groups[0].ItemCount == 2 {
		launchGroup = result.Groups[0]
	} else {
		launchGroup = result.Groups[1]
	}
	if launchGroup.ItemCount != 2 {
		t.Fatalf("expected launch cluster to contain 2 items, got %d", launchGroup.ItemCount)
	}
	if launchGroup.Importance != "high" {
		t.Fatalf("expected launch cluster importance high, got %q", launchGroup.Importance)
	}
	if len(launchGroup.Items) != 2 {
		t.Fatalf("expected grouped items to be preserved, got %d", len(launchGroup.Items))
	}
	if launchGroup.TopicLabel == "" {
		t.Fatal("expected topic label")
	}
	if launchGroup.Items[0].ItemTitle == "" {
		t.Fatalf("expected representative items in cluster, got %+v", launchGroup.Items)
	}
}
