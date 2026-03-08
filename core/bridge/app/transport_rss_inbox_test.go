package app

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"ghost-os/bridge/tools"
)

func TestHandleRSSInboxListGetAndPoll(t *testing.T) {
	handler, service, _ := newTestHandlerWithService(t, nil, nil)
	feedStore, err := tools.NewFeedStore(filepath.Join(t.TempDir(), "feeds.json"))
	if err != nil {
		t.Fatalf("new feed store: %v", err)
	}
	feed, _, err := feedStore.Upsert(tools.FeedUpsertInput{URL: "https://example.com/feed.xml", Title: "Example Feed"})
	if err != nil {
		t.Fatalf("upsert feed: %v", err)
	}
	inboxStore, err := NewRSSInboxStore(filepath.Join(t.TempDir(), "inbox.json"))
	if err != nil {
		t.Fatalf("new inbox store: %v", err)
	}
	service.rssInbox = &RSSInboxService{
		feedStore:  feedStore,
		inboxStore: inboxStore,
		fetcher: testRSSInboxFetcher{byURL: map[string]tools.RSSResult{
			feed.URL: {Feed: tools.RSSFeedInfo{Title: feed.Title}, Items: []tools.RSSItem{{ID: "post-1", Title: "Launch", Summary: "Launch summary", Link: "https://example.com/launch"}}},
		}},
		classifier: testRSSInboxClassifier{decisions: []rssInboxClassification{{Index: 0, Keep: true, AISummary: "Launch summary", Importance: "high", Reason: "Useful"}}},
		now:        func() time.Time { return time.Date(2026, 3, 8, 14, 0, 0, 0, time.UTC) },
	}
	service.rssInitErr = nil

	pollResp := serveRequest(handler, http.MethodPost, "/api/rss/inbox", `{}`, map[string]string{"Content-Type": "application/json"})
	if pollResp.Code != http.StatusOK {
		t.Fatalf("unexpected poll status: got %d want %d body=%s", pollResp.Code, http.StatusOK, pollResp.Body.String())
	}

	listResp := serveRequest(handler, http.MethodGet, "/api/rss/inbox?importance=high", "", nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("unexpected list status: got %d want %d body=%s", listResp.Code, http.StatusOK, listResp.Body.String())
	}
	items, ok := decodeResponseBody(t, listResp).Payload.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("unexpected list payload: %#v", decodeResponseBody(t, listResp).Payload)
	}
	item, _ := items[0].(map[string]any)
	id, _ := item["id"].(string)
	if id == "" {
		t.Fatalf("expected inbox item id, payload=%#v", item)
	}

	getResp := serveRequest(handler, http.MethodGet, "/api/rss/inbox/"+id, "", nil)
	if getResp.Code != http.StatusOK {
		t.Fatalf("unexpected get status: got %d want %d body=%s", getResp.Code, http.StatusOK, getResp.Body.String())
	}
	getItem, _ := decodeResponseBody(t, getResp).Payload.(map[string]any)
	if getItem["feed_id"] != feed.ID {
		t.Fatalf("unexpected inbox item payload: %#v", getItem)
	}
}

func TestBusRSSInboxActions(t *testing.T) {
	handler, service, _ := newTestHandlerWithService(t, nil, nil)
	service.rssInbox = nil
	service.rssInitErr = errors.New("rss not configured")

	resp := serveRequest(handler, http.MethodPost, "/api/bus", `{"action":"RSS_INBOX_LIST","params":{},"trace_id":"trace-rss-bus"}`, nil)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status: got %d want %d", resp.Code, http.StatusInternalServerError)
	}
}

func TestExecuteRSSInboxPollUsecaseUsesTaskID(t *testing.T) {
	feedStore, err := tools.NewFeedStore(filepath.Join(t.TempDir(), "feeds.json"))
	if err != nil {
		t.Fatalf("new feed store: %v", err)
	}
	feed, _, err := feedStore.Upsert(tools.FeedUpsertInput{URL: "https://example.com/feed.xml", Title: "Example Feed"})
	if err != nil {
		t.Fatalf("upsert feed: %v", err)
	}
	inboxStore, err := NewRSSInboxStore(filepath.Join(t.TempDir(), "inbox.json"))
	if err != nil {
		t.Fatalf("new inbox store: %v", err)
	}
	service := &bridgeService{rssInbox: &RSSInboxService{
		feedStore:  feedStore,
		inboxStore: inboxStore,
		fetcher: testRSSInboxFetcher{byURL: map[string]tools.RSSResult{
			feed.URL: {Feed: tools.RSSFeedInfo{Title: feed.Title}, Items: []tools.RSSItem{{ID: "post-1", Title: "Launch", Summary: "Launch summary"}}},
		}},
		classifier: testRSSInboxClassifier{decisions: []rssInboxClassification{{Index: 0, Keep: true, AISummary: "Launch summary", Importance: "normal", Reason: "Useful"}}},
		now:        func() time.Time { return time.Date(2026, 3, 8, 15, 0, 0, 0, time.UTC) },
	}}
	result, _, err := service.executeRSSInboxPollUsecase(context.Background(), rssInboxPollParams{}, "task-1", "trace-rss-task")
	if err != nil {
		t.Fatalf("poll usecase returned error: %v", err)
	}
	if result.TaskID != "task-1" {
		t.Fatalf("unexpected task id: got %q want %q", result.TaskID, "task-1")
	}
}
