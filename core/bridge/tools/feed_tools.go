package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type FeedSubscribeTool struct {
	store     *FeedStore
	rssClient *RSSFetchTool
}

type FeedListTool struct{ store *FeedStore }
type FeedUpdateTool struct{ store *FeedStore }
type FeedUnsubscribeTool struct{ store *FeedStore }

type feedSubscribeArgs struct {
	URL      string   `json:"url"`
	Title    string   `json:"title,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Priority string   `json:"priority,omitempty"`
	Enabled  *bool    `json:"enabled,omitempty"`
}

type feedListArgs struct {
	Enabled  *bool  `json:"enabled,omitempty"`
	Tag      string `json:"tag,omitempty"`
	Priority string `json:"priority,omitempty"`
}

type feedUpdateArgs struct {
	FeedID   string    `json:"feed_id"`
	Title    *string   `json:"title,omitempty"`
	Tags     *[]string `json:"tags,omitempty"`
	Priority *string   `json:"priority,omitempty"`
	Enabled  *bool     `json:"enabled,omitempty"`
}

type feedUnsubscribeArgs struct {
	FeedID string `json:"feed_id"`
}

type feedSubscribeResult struct {
	Action string           `json:"action"`
	Feed   FeedSubscription `json:"feed"`
}

type feedListResult struct {
	Feeds []FeedSubscription `json:"feeds"`
	Count int                `json:"count"`
}

type feedDeleteResult struct {
	Deleted bool   `json:"deleted"`
	FeedID  string `json:"feed_id"`
}

func NewFeedSubscribeTool(store *FeedStore) Tool {
	return &FeedSubscribeTool{store: store, rssClient: newDefaultRSSFetchTool()}
}

func NewFeedListTool(store *FeedStore) Tool {
	return &FeedListTool{store: store}
}

func NewFeedUpdateTool(store *FeedStore) Tool {
	return &FeedUpdateTool{store: store}
}

func NewFeedUnsubscribeTool(store *FeedStore) Tool {
	return &FeedUnsubscribeTool{store: store}
}

func (FeedSubscribeTool) Name() string { return "feed_subscribe" }

func (FeedSubscribeTool) Description() string {
	return "Add or refresh a shared RSS/Atom feed source after HTTPS and feed validation."
}

func (FeedSubscribeTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"url":{"type":"string","description":"HTTPS RSS or Atom feed URL."},
			"title":{"type":"string","description":"Optional custom display title."},
			"tags":{"type":"array","items":{"type":"string"},"description":"Optional feed tags."},
			"priority":{"type":"string","enum":["low","normal","high"],"description":"Optional feed priority (default: normal)."},
			"enabled":{"type":"boolean","description":"Whether the feed should be enabled (default: true)."}
		},
		"required":["url"],
		"additionalProperties":false
	}`)
}

func (t *FeedSubscribeTool) Execute(ctx context.Context, argsJSON json.RawMessage, _ string) (string, error) {
	if t == nil || t.store == nil {
		return "", fmt.Errorf("feed store is not configured")
	}
	var args feedSubscribeArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}
	feedURL, err := parseFeedSubscribeURL(args.URL)
	if err != nil {
		return "", err
	}
	if t.rssClient == nil {
		t.rssClient = newDefaultRSSFetchTool()
	}
	if err := t.rssClient.validateRequestURL(ctx, feedURL); err != nil {
		return "", err
	}
	result, err := t.rssClient.fetch(ctx, feedURL)
	if err != nil {
		return "", err
	}
	feed, action, err := t.store.Upsert(FeedUpsertInput{
		URL:        feedURL.String(),
		Title:      args.Title,
		Tags:       args.Tags,
		Priority:   args.Priority,
		Enabled:    args.Enabled,
		ProbeTitle: result.Feed.Title,
	})
	if err != nil {
		return "", err
	}
	return marshalFeedToolResult(feedSubscribeResult{Action: action, Feed: feed})
}

func (FeedListTool) Name() string { return "feed_list" }

func (FeedListTool) Description() string {
	return "List shared RSS/Atom feed sources with optional enabled, tag, and priority filters."
}

func (FeedListTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"enabled":{"type":"boolean","description":"Filter by enabled state."},
			"tag":{"type":"string","description":"Filter by a single tag."},
			"priority":{"type":"string","enum":["low","normal","high"],"description":"Filter by priority."}
		},
		"additionalProperties":false
	}`)
}

func (t *FeedListTool) Execute(_ context.Context, argsJSON json.RawMessage, _ string) (string, error) {
	if t == nil || t.store == nil {
		return "", fmt.Errorf("feed store is not configured")
	}
	var args feedListArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("decode args: %w", err)
		}
	}
	feeds, err := t.store.List(FeedListFilter{Enabled: args.Enabled, Tag: args.Tag, Priority: args.Priority})
	if err != nil {
		return "", err
	}
	return marshalFeedToolResult(feedListResult{Feeds: feeds, Count: len(feeds)})
}

func (FeedUpdateTool) Name() string { return "feed_update" }

func (FeedUpdateTool) Description() string {
	return "Update metadata, priority, tags, or enabled state for a shared RSS/Atom feed source."
}

func (FeedUpdateTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"feed_id":{"type":"string","description":"Feed identifier returned by feed_list or feed_subscribe."},
			"title":{"type":"string","description":"Optional replacement display title; empty string clears it."},
			"tags":{"type":"array","items":{"type":"string"},"description":"Optional full replacement tag list."},
			"priority":{"type":"string","enum":["low","normal","high"],"description":"Optional replacement priority."},
			"enabled":{"type":"boolean","description":"Optional enabled flag."}
		},
		"required":["feed_id"],
		"additionalProperties":false
	}`)
}

func (t *FeedUpdateTool) Execute(_ context.Context, argsJSON json.RawMessage, _ string) (string, error) {
	if t == nil || t.store == nil {
		return "", fmt.Errorf("feed store is not configured")
	}
	var args feedUpdateArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}
	feed, err := t.store.Update(args.FeedID, FeedUpdatePatch{
		Title:    args.Title,
		Tags:     args.Tags,
		Priority: args.Priority,
		Enabled:  args.Enabled,
	})
	if err != nil {
		return "", err
	}
	return marshalFeedToolResult(feedSubscribeResult{Action: "updated", Feed: feed})
}

func (FeedUnsubscribeTool) Name() string { return "feed_unsubscribe" }

func (FeedUnsubscribeTool) Description() string {
	return "Remove a shared RSS/Atom feed source by feed_id."
}

func (FeedUnsubscribeTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"feed_id":{"type":"string","description":"Feed identifier returned by feed_list or feed_subscribe."}
		},
		"required":["feed_id"],
		"additionalProperties":false
	}`)
}

func (t *FeedUnsubscribeTool) Execute(_ context.Context, argsJSON json.RawMessage, _ string) (string, error) {
	if t == nil || t.store == nil {
		return "", fmt.Errorf("feed store is not configured")
	}
	var args feedUnsubscribeArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}
	deleted, err := t.store.Delete(args.FeedID)
	if err != nil {
		return "", err
	}
	return marshalFeedToolResult(feedDeleteResult{Deleted: deleted, FeedID: strings.TrimSpace(args.FeedID)})
}

func parseFeedSubscribeURL(raw string) (*url.URL, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("url is required")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	parsed.Fragment = ""
	return parsed, nil
}

func marshalFeedToolResult(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode feed tool result: %w", err)
	}
	return string(encoded), nil
}

func newDefaultRSSFetchTool() *RSSFetchTool {
	tool := &RSSFetchTool{
		validateURL: validateRSSURL,
		now:         time.Now,
		bodyLimit:   defaultRSSBodyLimitBytes,
	}
	tool.httpClient = &http.Client{
		Timeout: defaultRSSFetchTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRSSRedirects {
				return fmt.Errorf("too many redirects")
			}
			return tool.validateRequestURL(req.Context(), req.URL)
		},
	}
	return tool
}
