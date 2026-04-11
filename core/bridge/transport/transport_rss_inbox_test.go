package transport

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	bridgeorchestration "ghost-os/bridge/orchestration"
	rsssubscriptions "ghost-os/bridge/rss/subscriptions"
	"ghost-os/bridge/tools"
)

func TestHandleRSSInboxListGetAndPoll(t *testing.T) {
	handler, service, _ := newTestHandlerWithService(t, nil, nil)
	feedStore, err := rsssubscriptions.NewFeedStore(filepath.Join(t.TempDir(), "feeds.json"))
	if err != nil {
		t.Fatalf("new feed store: %v", err)
	}
	feed, _, err := feedStore.Upsert(rsssubscriptions.FeedUpsertInput{URL: "https://example.com/feed.xml", Title: "Example Feed"})
	if err != nil {
		t.Fatalf("upsert feed: %v", err)
	}
	inboxStore, err := NewRSSInboxStore(filepath.Join(t.TempDir(), "inbox.json"))
	if err != nil {
		t.Fatalf("new inbox store: %v", err)
	}
	briefingStore, err := NewRSSBriefingStore(filepath.Join(t.TempDir(), "briefings.json"))
	if err != nil {
		t.Fatalf("new briefing store: %v", err)
	}
	rssConfig := bridgeconfig.Config{
		Provider: bridgeconfig.ProviderConfig{Model: "gpt-4o"},
		Worker:   bridgeconfig.WorkerConfig{Model: "gpt-4o-mini"},
	}
	rssService := NewRSSInboxService(feedStore, inboxStore, briefingStore, nil, nil, rssConfig)
	rssService.SetFetcher(testRSSInboxFetcher{byURL: map[string]tools.RSSResult{
		feed.URL: {Feed: tools.RSSFeedInfo{Title: feed.Title}, Items: []tools.RSSItem{{ID: "post-1", Title: "Launch", Summary: "Launch summary", Link: "https://example.com/launch"}}},
	}})
	rssService.SetClassifier(testRSSInboxClassifier{decisions: []RSSInboxClassification{{Index: 0, Keep: true, AISummary: "Launch summary", Importance: "high", Reason: "Useful"}}})
	rssService.SetBriefingBuilder(NewLLMRSSBriefingBuilder(&fakeSelectorCompleter{response: &llm.CompletionResponse{
		Message: llm.Message{Role: llm.RoleAssistant, Text: `{
  "title":"Launch Brief",
  "summary":"One launch matters today.",
  "highlights":[{"group_id":"invalid","headline":"ignored"}]
}`},
	}}, rssConfig))
	rssService.SetNow(func() time.Time { return time.Date(2026, 3, 8, 14, 0, 0, 0, time.UTC) })
	service.SetRSSInboxService(rssService, nil)

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

	groupResp := serveRequest(handler, http.MethodGet, "/api/rss/inbox/groups?window_hours=24&limit=5", "", nil)
	if groupResp.Code != http.StatusOK {
		t.Fatalf("unexpected groups status: got %d want %d body=%s", groupResp.Code, http.StatusOK, groupResp.Body.String())
	}
	groupPayload, _ := decodeResponseBody(t, groupResp).Payload.(map[string]any)
	groups, _ := groupPayload["groups"].([]any)
	if len(groups) != 1 {
		t.Fatalf("unexpected grouped payload: %#v", groupPayload)
	}

	briefingBuildResp := serveRequest(
		handler,
		http.MethodPost,
		"/api/rss/briefing",
		`{"window_hours":24,"highlights_limit":3}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if briefingBuildResp.Code != http.StatusOK {
		t.Fatalf("unexpected briefing build status: got %d want %d body=%s", briefingBuildResp.Code, http.StatusOK, briefingBuildResp.Body.String())
	}

	briefingResp := serveRequest(handler, http.MethodGet, "/api/rss/briefing", "", nil)
	if briefingResp.Code != http.StatusOK {
		t.Fatalf("unexpected briefing status: got %d want %d body=%s", briefingResp.Code, http.StatusOK, briefingResp.Body.String())
	}
	briefingPayload, _ := decodeResponseBody(t, briefingResp).Payload.(map[string]any)
	highlights, _ := briefingPayload["highlights"].([]any)
	if len(highlights) != 1 {
		t.Fatalf("unexpected briefing payload: %#v", briefingPayload)
	}
}

func TestBusRSSInboxActions(t *testing.T) {
	handler, service, _ := newTestHandlerWithService(t, nil, nil)
	service.SetRSSInboxService(nil, errors.New("rss not configured"))

	resp := serveRequest(handler, http.MethodPost, "/api/bus", `{"action":"RSS_INBOX_LIST","params":{},"trace_id":"trace-rss-bus"}`, nil)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status: got %d want %d", resp.Code, http.StatusInternalServerError)
	}

	resp = serveRequest(handler, http.MethodPost, "/api/bus", `{"action":"RSS_INBOX_GROUPS","params":{},"trace_id":"trace-rss-groups"}`, nil)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status: got %d want %d", resp.Code, http.StatusInternalServerError)
	}

	resp = serveRequest(handler, http.MethodPost, "/api/bus", `{"action":"RSS_BRIEFING_BUILD","params":{},"trace_id":"trace-rss-briefing"}`, nil)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status: got %d want %d", resp.Code, http.StatusInternalServerError)
	}

	resp = serveRequest(handler, http.MethodPost, "/api/bus", `{"action":"RSS_BRIEFING_GET","params":{},"trace_id":"trace-rss-briefing-get"}`, nil)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status: got %d want %d", resp.Code, http.StatusInternalServerError)
	}
}

func TestExecuteRSSInboxPollUsecaseUsesTaskID(t *testing.T) {
	feedStore, err := rsssubscriptions.NewFeedStore(filepath.Join(t.TempDir(), "feeds.json"))
	if err != nil {
		t.Fatalf("new feed store: %v", err)
	}
	feed, _, err := feedStore.Upsert(rsssubscriptions.FeedUpsertInput{URL: "https://example.com/feed.xml", Title: "Example Feed"})
	if err != nil {
		t.Fatalf("upsert feed: %v", err)
	}
	inboxStore, err := NewRSSInboxStore(filepath.Join(t.TempDir(), "inbox.json"))
	if err != nil {
		t.Fatalf("new inbox store: %v", err)
	}
	rssService := NewRSSInboxService(feedStore, inboxStore, nil, nil, nil, bridgeconfig.Config{})
	rssService.SetFetcher(testRSSInboxFetcher{byURL: map[string]tools.RSSResult{
		feed.URL: {Feed: tools.RSSFeedInfo{Title: feed.Title}, Items: []tools.RSSItem{{ID: "post-1", Title: "Launch", Summary: "Launch summary"}}},
	}})
	rssService.SetClassifier(testRSSInboxClassifier{decisions: []RSSInboxClassification{{Index: 0, Keep: true, AISummary: "Launch summary", Importance: "normal", Reason: "Useful"}}})
	rssService.SetNow(func() time.Time { return time.Date(2026, 3, 8, 15, 0, 0, 0, time.UTC) })
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	service.SetRSSInboxService(rssService, nil)
	result, _, err := service.ExecuteRSSInboxPollUsecase(context.Background(), bridgeorchestration.RSSInboxPollParams{}, "task-1", "trace-rss-task")
	if err != nil {
		t.Fatalf("poll usecase returned error: %v", err)
	}
	if result.TaskID != "task-1" {
		t.Fatalf("unexpected task id: got %q want %q", result.TaskID, "task-1")
	}
}
