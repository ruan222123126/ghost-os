package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"ghost-os/bridge/tools"
)

type testRSSInboxFetcher struct {
	byURL map[string]tools.RSSResult
	err   map[string]error
}

func (f testRSSInboxFetcher) Fetch(_ context.Context, feedURL string, _ int) (tools.RSSResult, error) {
	if err := f.err[feedURL]; err != nil {
		return tools.RSSResult{}, err
	}
	result, ok := f.byURL[feedURL]
	if !ok {
		return tools.RSSResult{}, nil
	}
	return result, nil
}

type testRSSInboxClassifier struct {
	decisions []rssInboxClassification
	err       error
	perFeed   map[string][]rssInboxClassification
	errByFeed map[string]error
}

func (c testRSSInboxClassifier) Classify(_ context.Context, feed tools.FeedSubscription, items []rssInboxCandidate, _ string) ([]rssInboxClassification, error) {
	if err := c.errByFeed[feed.ID]; err != nil {
		return nil, err
	}
	if err := c.errByFeed[feed.URL]; err != nil {
		return nil, err
	}
	if decisions, ok := c.perFeed[feed.ID]; ok {
		return decisions[:min(len(decisions), len(items))], nil
	}
	if decisions, ok := c.perFeed[feed.URL]; ok {
		return decisions[:min(len(decisions), len(items))], nil
	}
	if c.err != nil {
		return nil, c.err
	}
	return c.decisions[:min(len(c.decisions), len(items))], nil
}

func TestRSSInboxServicePollProcessesNewItemsAndSkipsDuplicates(t *testing.T) {
	feedStore, err := tools.NewFeedStore(filepath.Join(t.TempDir(), "feeds.json"))
	if err != nil {
		t.Fatalf("new feed store: %v", err)
	}
	_, _, err = feedStore.Upsert(tools.FeedUpsertInput{URL: "https://example.com/feed.xml", Title: "Example Feed", Tags: []string{"AI"}})
	if err != nil {
		t.Fatalf("upsert feed: %v", err)
	}
	inboxStore, err := NewRSSInboxStore(filepath.Join(t.TempDir(), "inbox.json"))
	if err != nil {
		t.Fatalf("new inbox store: %v", err)
	}
	service := &RSSInboxService{
		feedStore:  feedStore,
		inboxStore: inboxStore,
		fetcher: testRSSInboxFetcher{byURL: map[string]tools.RSSResult{
			"https://example.com/feed.xml": {
				Feed: tools.RSSFeedInfo{Title: "Example Feed"},
				Items: []tools.RSSItem{
					{ID: "post-1", Title: "Launch", Link: "https://example.com/launch", Summary: "Launch summary", PublishedAt: "2026-03-08T08:00:00Z"},
					{ID: "post-2", Title: "Skip", Link: "https://example.com/skip", Summary: "Skip summary", PublishedAt: "2026-03-08T07:00:00Z"},
				},
			},
		}},
		classifier:  testRSSInboxClassifier{decisions: []rssInboxClassification{{Index: 0, Keep: true, AISummary: "Launch summary", Tags: []string{"Launch"}, Importance: "high", Reason: "Useful"}, {Index: 1, Keep: false, Reason: "Low signal"}}},
		now:         func() time.Time { return time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC) },
		aiBatchSize: 4,
	}

	result, err := service.Poll(context.Background(), RSSInboxPollOptions{TraceID: "trace-rss-1"})
	if err != nil {
		t.Fatalf("poll returned error: %v", err)
	}
	if result.ItemsSaved != 1 || result.ItemsDiscarded != 1 {
		t.Fatalf("unexpected poll result: %+v", result)
	}

	items, err := inboxStore.List(RSSInboxListFilter{})
	if err != nil {
		t.Fatalf("list inbox: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected inbox size: got %d want %d", len(items), 1)
	}
	if items[0].Importance != "high" || items[0].TraceID != "trace-rss-1" {
		t.Fatalf("unexpected saved item: %+v", items[0])
	}
	if len(items[0].Tags) != 2 {
		t.Fatalf("expected merged feed/ai tags, got %+v", items[0].Tags)
	}

	result, err = service.Poll(context.Background(), RSSInboxPollOptions{TraceID: "trace-rss-2"})
	if err != nil {
		t.Fatalf("second poll returned error: %v", err)
	}
	if result.ItemsSaved != 0 {
		t.Fatalf("expected duplicate poll to save nothing, got %+v", result)
	}
}

func TestRSSInboxServicePollContinuesOnFeedAndClassifierFailures(t *testing.T) {
	feedStore, err := tools.NewFeedStore(filepath.Join(t.TempDir(), "feeds.json"))
	if err != nil {
		t.Fatalf("new feed store: %v", err)
	}
	_, _, _ = feedStore.Upsert(tools.FeedUpsertInput{URL: "https://example.com/good.xml", Title: "Good"})
	_, _, _ = feedStore.Upsert(tools.FeedUpsertInput{URL: "https://example.com/bad.xml", Title: "Bad"})
	_, _, _ = feedStore.Upsert(tools.FeedUpsertInput{URL: "https://example.com/classifier.xml", Title: "Classifier"})
	inboxStore, err := NewRSSInboxStore(filepath.Join(t.TempDir(), "inbox.json"))
	if err != nil {
		t.Fatalf("new inbox store: %v", err)
	}
	service := &RSSInboxService{
		feedStore:  feedStore,
		inboxStore: inboxStore,
		fetcher: testRSSInboxFetcher{
			byURL: map[string]tools.RSSResult{
				"https://example.com/good.xml":       {Feed: tools.RSSFeedInfo{Title: "Good"}, Items: []tools.RSSItem{{ID: "good-1", Title: "Good Item", Summary: "Keep me"}}},
				"https://example.com/classifier.xml": {Feed: tools.RSSFeedInfo{Title: "Classifier"}, Items: []tools.RSSItem{{ID: "cls-1", Title: "Broken JSON", Summary: "Will fail classify"}}},
			},
			err: map[string]error{"https://example.com/bad.xml": errors.New("fetch failed")},
		},
		classifier: testRSSInboxClassifier{
			perFeed:   map[string][]rssInboxClassification{"https://example.com/good.xml": {{Index: 0, Keep: true, AISummary: "Good", Importance: "normal", Reason: "Useful"}}},
			errByFeed: map[string]error{"https://example.com/classifier.xml": errors.New("bad json")},
		},
		now:         func() time.Time { return time.Date(2026, 3, 8, 13, 0, 0, 0, time.UTC) },
		aiBatchSize: 4,
	}

	result, err := service.Poll(context.Background(), RSSInboxPollOptions{TraceID: "trace-rss-3"})
	if err != nil {
		t.Fatalf("poll returned error: %v", err)
	}
	if result.FeedsScanned != 3 || result.FeedsFailed != 1 {
		t.Fatalf("unexpected feed counters: %+v", result)
	}
	if result.ItemsSaved != 1 || result.ItemsDiscarded != 1 {
		t.Fatalf("unexpected item counters: %+v", result)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
