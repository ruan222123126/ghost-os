package rss

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/llm"
	rsssubscriptions "ghost-os/bridge/rss/subscriptions"
	"ghost-os/bridge/tools"
)

const defaultRSSClassifierTimeout = 45 * time.Second

type RSSResult = tools.RSSResult
type RSSItem = tools.RSSItem

type defaultRSSInboxFetcher struct{}

type llmRSSInboxClassifier struct {
	store   *ConfigStore
	timeout time.Duration
	client  llm.Completer
	cfg     Config
}

func (defaultRSSInboxFetcher) Fetch(ctx context.Context, feedURL string, maxItems int) (RSSResult, error) {
	return tools.FetchRSS(ctx, feedURL, tools.RSSFetchOptions{MaxItems: maxItems, IncludeSummary: true})
}

func (c *llmRSSInboxClassifier) Classify(
	ctx context.Context,
	feed rsssubscriptions.FeedSubscription,
	items []rssInboxCandidate,
	traceID string,
) ([]rssInboxClassification, error) {
	if len(items) == 0 {
		return nil, nil
	}
	client, cfg, err := c.workerClient()
	if err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithTimeout(ctx, c.classifierTimeout())
	defer cancel()
	resp, err := client.Complete(runCtx, llm.CompletionRequest{Messages: []llm.Message{
		{Role: llm.RoleSystem, Text: "You are Ghost-OS RSS inbox worker. Review RSS items and keep only items that are worth a human's attention. Favor concrete announcements, product or model releases, security or policy changes, useful tutorials, sharp analysis, and notable ecosystem signals. Reject obvious spam, repetitive low-signal headlines, and thin updates. Return strict JSON array only with objects: index, keep, ai_summary, tags, importance, reason. importance must be one of low, normal, high. ai_summary and reason must stay concise."},
		{Role: llm.RoleUser, Text: renderRSSInboxClassificationPrompt(feed, items, cfg, traceID)},
	}})
	if err != nil {
		return nil, err
	}
	return decodeRSSInboxClassification(resp.Message.Text)
}

func (c *llmRSSInboxClassifier) workerClient() (llm.Completer, Config, error) {
	if c != nil && c.client != nil {
		return c.client, c.cfg, nil
	}
	cfg, err := loadRSSClassifierConfig(c)
	if err != nil {
		return nil, Config{}, err
	}
	client := llm.NewClientWithOptions(providerClientOptions(cfg, effectiveWorkerModel(cfg)))
	if c != nil {
		c.cfg = cfg
	}
	return client, cfg, nil
}

func (c *llmRSSInboxClassifier) classifierTimeout() time.Duration {
	if c != nil && c.timeout > 0 {
		return c.timeout
	}
	return defaultRSSClassifierTimeout
}

func loadRSSClassifierConfig(c *llmRSSInboxClassifier) (Config, error) {
	switch {
	case c != nil && c.store != nil:
		return loadConfigWithRuntime(c.store.RuntimeConfig())
	case c != nil && strings.TrimSpace(c.cfg.Provider.Model) != "":
		return c.cfg, nil
	default:
		return LoadConfig()
	}
}

func decodeRSSInboxClassification(text string) ([]rssInboxClassification, error) {
	text = stripJSONCodeFence(strings.TrimSpace(text))
	if text == "" {
		return nil, fmt.Errorf("rss inbox classifier returned empty content")
	}
	var decisions []rssInboxClassification
	if err := json.Unmarshal([]byte(text), &decisions); err != nil {
		return nil, fmt.Errorf("decode rss inbox classification: %w", err)
	}
	for i := range decisions {
		decisions[i].AISummary = truncateRunes(strings.TrimSpace(decisions[i].AISummary), maxRSSClassifierSummaryLen)
		decisions[i].Importance = normalizeRSSInboxImportance(decisions[i].Importance)
		decisions[i].Reason = truncateRunes(strings.TrimSpace(decisions[i].Reason), 180)
		decisions[i].Tags = normalizeRSSInboxTags(decisions[i].Tags)
	}
	return decisions, nil
}

func renderRSSInboxClassificationPrompt(
	feed rsssubscriptions.FeedSubscription,
	items []rssInboxCandidate,
	cfg Config,
	traceID string,
) string {
	type promptItem struct {
		Index       int      `json:"index"`
		Title       string   `json:"title,omitempty"`
		Link        string   `json:"link,omitempty"`
		PublishedAt string   `json:"published_at,omitempty"`
		Summary     string   `json:"summary,omitempty"`
		FeedTags    []string `json:"feed_tags,omitempty"`
	}
	payload := struct {
		WorkerModel string         `json:"worker_model"`
		TraceID     string         `json:"trace_id,omitempty"`
		Feed        map[string]any `json:"feed"`
		Items       []promptItem   `json:"items"`
	}{
		WorkerModel: effectiveWorkerModel(cfg),
		TraceID:     strings.TrimSpace(traceID),
		Feed: map[string]any{
			"feed_id":  feed.ID,
			"feed_url": feed.URL,
			"title":    feed.Title,
			"priority": feed.Priority,
			"tags":     append([]string(nil), feed.Tags...),
		},
		Items: make([]promptItem, 0, len(items)),
	}
	for index, item := range items {
		entry := promptItem{
			Index:    index,
			Title:    item.ItemTitle,
			Link:     item.ItemLink,
			Summary:  item.RawSummary,
			FeedTags: append([]string(nil), item.FeedTags...),
		}
		if !item.PublishedAt.IsZero() {
			entry.PublishedAt = item.PublishedAt.UTC().Format(time.RFC3339)
		}
		payload.Items = append(payload.Items, entry)
	}
	encoded, _ := json.MarshalIndent(payload, "", "  ")
	return string(encoded)
}
