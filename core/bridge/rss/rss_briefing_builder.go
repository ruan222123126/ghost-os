package rss

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

func (b *llmRSSBriefingBuilder) Build(
	ctx context.Context,
	groups []RSSInboxTopicGroup,
	query RSSBriefingQuery,
) (rssBriefingDraft, error) {
	if len(groups) == 0 {
		return rssBriefingDraft{}, nil
	}
	client, cfg, err := b.workerClient()
	if err != nil {
		return rssBriefingDraft{}, err
	}
	runCtx, cancel := context.WithTimeout(ctx, b.builderTimeout())
	defer cancel()
	resp, err := client.Complete(runCtx, llm.CompletionRequest{Messages: []llm.Message{
		{Role: llm.RoleSystem, Text: "You are Ghost-OS RSS briefing worker. Build a concise, high-signal news briefing from grouped RSS topics. Return strict JSON only with object fields: title, summary, highlights. highlights must be an array of objects with: group_id, headline, summary, why_it_matters, importance. Use only the provided group_id values. importance must be one of low, normal, high. Do not include markdown, commentary, or code fences."},
		{Role: llm.RoleUser, Text: renderRSSBriefingPrompt(groups, query, cfg)},
	}})
	if err != nil {
		return rssBriefingDraft{}, err
	}
	return decodeRSSBriefingDraft(resp.Message.Text)
}

func (b *llmRSSBriefingBuilder) workerClient() (llm.Completer, Config, error) {
	if b != nil && b.client != nil {
		return b.client, b.cfg, nil
	}
	cfg, err := loadRSSBriefingConfig(b)
	if err != nil {
		return nil, Config{}, err
	}
	client := llm.NewClientWithOptions(providerClientOptions(cfg, effectiveWorkerModel(cfg)))
	if b != nil {
		b.cfg = cfg
	}
	return client, cfg, nil
}

func (b *llmRSSBriefingBuilder) builderTimeout() time.Duration {
	if b != nil && b.timeout > 0 {
		return b.timeout
	}
	return defaultRSSBriefingTimeout
}

func loadRSSBriefingConfig(b *llmRSSBriefingBuilder) (Config, error) {
	switch {
	case b != nil && b.store != nil:
		return loadConfigWithRuntime(b.store.RuntimeConfig())
	case b != nil && strings.TrimSpace(b.cfg.Provider.Model) != "":
		return b.cfg, nil
	default:
		return LoadConfig()
	}
}

func decodeRSSBriefingDraft(text string) (rssBriefingDraft, error) {
	text = stripJSONCodeFence(strings.TrimSpace(text))
	if text == "" {
		return rssBriefingDraft{}, fmt.Errorf("rss briefing builder returned empty content")
	}
	var draft rssBriefingDraft
	if err := json.Unmarshal([]byte(text), &draft); err != nil {
		return rssBriefingDraft{}, fmt.Errorf("decode rss briefing draft: %w", err)
	}
	draft.Title = truncateRunes(strings.TrimSpace(draft.Title), 140)
	draft.Summary = truncateRunes(strings.TrimSpace(draft.Summary), 400)
	for i := range draft.Highlights {
		draft.Highlights[i].GroupID = strings.TrimSpace(draft.Highlights[i].GroupID)
		draft.Highlights[i].Headline = truncateRunes(strings.TrimSpace(draft.Highlights[i].Headline), 180)
		draft.Highlights[i].Summary = truncateRunes(strings.TrimSpace(draft.Highlights[i].Summary), 280)
		draft.Highlights[i].WhyItMatters = truncateRunes(strings.TrimSpace(draft.Highlights[i].WhyItMatters), 220)
		draft.Highlights[i].Importance = normalizeRSSInboxImportance(draft.Highlights[i].Importance)
	}
	return draft, nil
}

func renderRSSBriefingPrompt(groups []RSSInboxTopicGroup, query RSSBriefingQuery, cfg Config) string {
	type promptItem struct {
		Title       string `json:"title,omitempty"`
		Summary     string `json:"summary,omitempty"`
		Importance  string `json:"importance,omitempty"`
		PublishedAt string `json:"published_at,omitempty"`
		Link        string `json:"link,omitempty"`
	}
	type promptGroup struct {
		GroupID          string       `json:"group_id"`
		TopicLabel       string       `json:"topic_label,omitempty"`
		Headline         string       `json:"headline,omitempty"`
		Summary          string       `json:"summary,omitempty"`
		Importance       string       `json:"importance,omitempty"`
		ItemCount        int          `json:"item_count"`
		FeedCount        int          `json:"feed_count"`
		LatestActivityAt string       `json:"latest_activity_at,omitempty"`
		Tags             []string     `json:"tags,omitempty"`
		Items            []promptItem `json:"items,omitempty"`
	}
	payload := struct {
		WorkerModel     string        `json:"worker_model"`
		TraceID         string        `json:"trace_id,omitempty"`
		WindowHours     int           `json:"window_hours"`
		HighlightsLimit int           `json:"highlights_limit"`
		Groups          []promptGroup `json:"groups"`
	}{
		WorkerModel:     effectiveWorkerModel(cfg),
		TraceID:         strings.TrimSpace(query.TraceID),
		WindowHours:     query.WindowHours,
		HighlightsLimit: query.HighlightsLimit,
		Groups:          make([]promptGroup, 0, len(groups)),
	}
	for _, group := range groups {
		entry := promptGroup{
			GroupID:    group.ID,
			TopicLabel: group.TopicLabel,
			Headline:   group.Headline,
			Summary:    group.Summary,
			Importance: group.Importance,
			ItemCount:  group.ItemCount,
			FeedCount:  group.FeedCount,
			Tags:       append([]string(nil), group.Tags...),
			Items:      make([]promptItem, 0, len(group.Items)),
		}
		if !group.LatestActivityAt.IsZero() {
			entry.LatestActivityAt = group.LatestActivityAt.UTC().Format(time.RFC3339)
		}
		for _, item := range group.Items {
			promptItem := promptItem{
				Title:      item.ItemTitle,
				Summary:    rssInboxItemSummary(item),
				Importance: item.Importance,
				Link:       item.ItemLink,
			}
			if !item.PublishedAt.IsZero() {
				promptItem.PublishedAt = item.PublishedAt.UTC().Format(time.RFC3339)
			}
			entry.Items = append(entry.Items, promptItem)
		}
		payload.Groups = append(payload.Groups, entry)
	}
	encoded, _ := json.MarshalIndent(payload, "", "  ")
	return string(encoded)
}
