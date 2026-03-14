package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	rsssubscriptions "ghost-os/bridge/rss/subscriptions"
)

const (
	feedManageOperationSubscribe   = "subscribe"
	feedManageOperationList        = "list"
	feedManageOperationUpdate      = "update"
	feedManageOperationUnsubscribe = "unsubscribe"
)

type FeedManageTool struct {
	store     *rsssubscriptions.FeedStore
	rssClient *RSSFetchTool
}

type feedManageArgs struct {
	Operation string    `json:"operation"`
	URL       string    `json:"url,omitempty"`
	FeedID    string    `json:"feed_id,omitempty"`
	Title     *string   `json:"title,omitempty"`
	Tags      *[]string `json:"tags,omitempty"`
	Priority  *string   `json:"priority,omitempty"`
	Enabled   *bool     `json:"enabled,omitempty"`
	Tag       *string   `json:"tag,omitempty"`
}

type feedManageResult struct {
	Action string                            `json:"action"`
	Feed   rsssubscriptions.FeedSubscription `json:"feed"`
}

type feedListResult struct {
	Feeds []rsssubscriptions.FeedSubscription `json:"feeds"`
	Count int                                 `json:"count"`
}

type feedDeleteResult struct {
	Deleted bool   `json:"deleted"`
	FeedID  string `json:"feed_id"`
}

func NewFeedManageTool(store *rsssubscriptions.FeedStore) Tool {
	return &FeedManageTool{store: store, rssClient: newDefaultRSSFetchTool()}
}

func (FeedManageTool) Name() string {
	return "feed_manage"
}

func (FeedManageTool) Description() string {
	return "Manage shared RSS/Atom feed sources: subscribe, list, update metadata, or unsubscribe."
}

func (FeedManageTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"operation":{"type":"string","enum":["subscribe","list","update","unsubscribe"]},
			"url":{"type":"string","description":"HTTPS RSS or Atom feed URL for subscribe."},
			"feed_id":{"type":"string","description":"Feed identifier for update or unsubscribe."},
			"title":{"type":"string","description":"Display title for subscribe, or replacement title for update; empty string clears it on update."},
			"tags":{"type":"array","items":{"type":"string"},"description":"Tags for subscribe, or full replacement tag list for update."},
			"priority":{"type":"string","enum":["low","normal","high"],"description":"Priority for subscribe, update, or list filter."},
			"enabled":{"type":"boolean","description":"Enabled state for subscribe, update, or list filter."},
			"tag":{"type":"string","description":"Single tag filter for list."}
		},
		"required":["operation"],
		"additionalProperties":false
	}`)
}

func (t *FeedManageTool) Execute(ctx context.Context, argsJSON json.RawMessage, _ string) (string, error) {
	if t == nil || t.store == nil {
		return "", fmt.Errorf("feed store is not configured")
	}

	var args feedManageArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}

	switch strings.ToLower(strings.TrimSpace(args.Operation)) {
	case feedManageOperationSubscribe:
		return t.executeSubscribe(ctx, args)
	case feedManageOperationList:
		return t.executeList(args)
	case feedManageOperationUpdate:
		return t.executeUpdate(args)
	case feedManageOperationUnsubscribe:
		return t.executeUnsubscribe(args)
	default:
		return "", fmt.Errorf("unsupported operation %q", strings.TrimSpace(args.Operation))
	}
}

func (t *FeedManageTool) executeSubscribe(ctx context.Context, args feedManageArgs) (string, error) {
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
	feed, action, err := t.store.Upsert(rsssubscriptions.FeedUpsertInput{
		URL:        feedURL.String(),
		Title:      feedOptionalStringValue(args.Title),
		Tags:       feedOptionalStringsValue(args.Tags),
		Priority:   feedOptionalStringValue(args.Priority),
		Enabled:    args.Enabled,
		ProbeTitle: result.Feed.Title,
	})
	if err != nil {
		return "", err
	}
	return marshalFeedToolResult(feedManageResult{Action: action, Feed: feed})
}

func (t *FeedManageTool) executeList(args feedManageArgs) (string, error) {
	feeds, err := t.store.List(rsssubscriptions.FeedListFilter{
		Enabled:  args.Enabled,
		Tag:      feedOptionalStringValue(args.Tag),
		Priority: feedOptionalStringValue(args.Priority),
	})
	if err != nil {
		return "", err
	}
	return marshalFeedToolResult(feedListResult{Feeds: feeds, Count: len(feeds)})
}

func (t *FeedManageTool) executeUpdate(args feedManageArgs) (string, error) {
	feedID := strings.TrimSpace(args.FeedID)
	if feedID == "" {
		return "", fmt.Errorf("feed_id is required")
	}

	feed, err := t.store.Update(feedID, rsssubscriptions.FeedUpdatePatch{
		Title:    args.Title,
		Tags:     args.Tags,
		Priority: args.Priority,
		Enabled:  args.Enabled,
	})
	if err != nil {
		return "", err
	}
	return marshalFeedToolResult(feedManageResult{Action: "updated", Feed: feed})
}

func (t *FeedManageTool) executeUnsubscribe(args feedManageArgs) (string, error) {
	feedID := strings.TrimSpace(args.FeedID)
	if feedID == "" {
		return "", fmt.Errorf("feed_id is required")
	}

	deleted, err := t.store.Delete(feedID)
	if err != nil {
		return "", err
	}
	return marshalFeedToolResult(feedDeleteResult{Deleted: deleted, FeedID: feedID})
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

func feedOptionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func feedOptionalStringsValue(values *[]string) []string {
	if values == nil {
		return nil
	}
	return *values
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
