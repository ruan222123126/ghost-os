package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	rsssubscriptions "ghost-os/bridge/rss/subscriptions"
)

func TestFeedSubscribeToolExecuteCreatesAndListsFeeds(t *testing.T) {
	store, err := rsssubscriptions.NewFeedStore(filepath.Join(t.TempDir(), "feeds.json"))
	if err != nil {
		t.Fatalf("NewFeedStore: %v", err)
	}

	tool := &FeedManageTool{
		store: store,
		rssClient: &RSSFetchTool{
			httpClient:  &http.Client{Transport: staticRSSRoundTripper(`<rss version="2.0"><channel><title>Ghost Feed</title><item><guid>a</guid><title>A</title><pubDate>Sun, 08 Mar 2026 10:00:00 GMT</pubDate></item></channel></rss>`)},
			validateURL: func(context.Context, *url.URL) error { return nil },
			now:         func() time.Time { return time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC) },
			bodyLimit:   defaultRSSBodyLimitBytes,
		},
	}
	output, err := tool.Execute(context.Background(), json.RawMessage(`{"operation":"subscribe","url":"https://example.com/feed.xml","tags":["ai"],"priority":"high"}`), "trace-feed-1")
	if err != nil {
		t.Fatalf("Execute subscribe: %v", err)
	}
	var subscribeResult feedManageResult
	if err := json.Unmarshal([]byte(output), &subscribeResult); err != nil {
		t.Fatalf("decode subscribe result: %v", err)
	}
	if subscribeResult.Action != "created" {
		t.Fatalf("unexpected action: %q", subscribeResult.Action)
	}
	if subscribeResult.Feed.Title != "Ghost Feed" || subscribeResult.Feed.Priority != "high" {
		t.Fatalf("unexpected feed result: %+v", subscribeResult.Feed)
	}

	listOutput, err := tool.Execute(context.Background(), json.RawMessage(`{"operation":"list","enabled":true}`), "trace-feed-2")
	if err != nil {
		t.Fatalf("Execute list: %v", err)
	}
	var listResult feedListResult
	if err := json.Unmarshal([]byte(listOutput), &listResult); err != nil {
		t.Fatalf("decode list result: %v", err)
	}
	if listResult.Count != 1 || listResult.Feeds[0].ID != subscribeResult.Feed.ID {
		t.Fatalf("unexpected list result: %+v", listResult)
	}
}

func TestFeedUpdateAndUnsubscribeToolsExecute(t *testing.T) {
	store, err := rsssubscriptions.NewFeedStore(filepath.Join(t.TempDir(), "feeds.json"))
	if err != nil {
		t.Fatalf("NewFeedStore: %v", err)
	}
	feed, _, err := store.Upsert(rsssubscriptions.FeedUpsertInput{URL: "https://example.com/feed.xml", ProbeTitle: "Ghost Feed"})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	tool := NewFeedManageTool(store)
	updateOutput, err := tool.Execute(context.Background(), json.RawMessage(`{"operation":"update","feed_id":"`+feed.ID+`","title":"Ops Feed","tags":["ops"],"enabled":false}`), "trace-feed-3")
	if err != nil {
		t.Fatalf("Execute update: %v", err)
	}
	var updateResult feedManageResult
	if err := json.Unmarshal([]byte(updateOutput), &updateResult); err != nil {
		t.Fatalf("decode update result: %v", err)
	}
	if updateResult.Feed.Title != "Ops Feed" || updateResult.Feed.Enabled {
		t.Fatalf("unexpected updated feed: %+v", updateResult.Feed)
	}

	deleteOutput, err := tool.Execute(context.Background(), json.RawMessage(`{"operation":"unsubscribe","feed_id":"`+feed.ID+`"}`), "trace-feed-4")
	if err != nil {
		t.Fatalf("Execute unsubscribe: %v", err)
	}
	var deleteResult feedDeleteResult
	if err := json.Unmarshal([]byte(deleteOutput), &deleteResult); err != nil {
		t.Fatalf("decode delete result: %v", err)
	}
	if !deleteResult.Deleted || deleteResult.FeedID != feed.ID {
		t.Fatalf("unexpected delete result: %+v", deleteResult)
	}
	deleteOutput, err = tool.Execute(context.Background(), json.RawMessage(`{"operation":"unsubscribe","feed_id":"`+feed.ID+`"}`), "trace-feed-5")
	if err != nil {
		t.Fatalf("Execute missing unsubscribe: %v", err)
	}
	if err := json.Unmarshal([]byte(deleteOutput), &deleteResult); err != nil {
		t.Fatalf("decode second delete result: %v", err)
	}
	if deleteResult.Deleted {
		t.Fatalf("expected delete=false on missing feed, got %+v", deleteResult)
	}
}

func TestFeedSubscribeToolRejectsInvalidSource(t *testing.T) {
	store, err := rsssubscriptions.NewFeedStore(filepath.Join(t.TempDir(), "feeds.json"))
	if err != nil {
		t.Fatalf("NewFeedStore: %v", err)
	}
	tool := NewFeedManageTool(store)
	_, err = tool.Execute(context.Background(), json.RawMessage(`{"operation":"subscribe","url":"http://example.com/feed.xml"}`), "trace-feed-6")
	if err == nil {
		t.Fatal("expected error for non-https source")
	}
	if !strings.Contains(err.Error(), "https") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type staticRSSRoundTripper string

func (s staticRSSRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/rss+xml"}},
		Body:       io.NopCloser(strings.NewReader(string(s))),
	}, nil
}
