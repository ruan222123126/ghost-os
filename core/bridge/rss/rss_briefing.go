package rss

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

const (
	defaultRSSBriefingTimeout         = 45 * time.Second
	defaultRSSBriefingGroupLimit      = 10
	maxRSSBriefingGroupLimit          = 20
	defaultRSSBriefingHighlightsLimit = 5
	maxRSSBriefingHighlightsLimit     = 10
	defaultRSSBriefingTaskID          = "system-rss-briefing-build"
)

type RSSBriefingQuery struct {
	FeedID          string
	Tag             string
	Importance      string
	WindowHours     int
	GroupLimit      int
	ItemLimit       int
	ItemsPerGroup   int
	HighlightsLimit int
	TraceID         string
	TaskID          string
}

type RSSBriefingResult struct {
	ID             string                 `json:"id,omitempty"`
	Title          string                 `json:"title"`
	Summary        string                 `json:"summary,omitempty"`
	GeneratedAt    time.Time              `json:"generated_at"`
	SavedAt        time.Time              `json:"saved_at,omitempty"`
	WindowHours    int                    `json:"window_hours"`
	ScannedGroups  int                    `json:"scanned_groups"`
	HighlightCount int                    `json:"highlight_count"`
	TraceID        string                 `json:"trace_id,omitempty"`
	TaskID         string                 `json:"task_id,omitempty"`
	Highlights     []RSSBriefingHighlight `json:"highlights"`
	Report         *RSSReportResult       `json:"report,omitempty"`
	ReportError    string                 `json:"report_error,omitempty"`
}

type RSSBriefingHighlight struct {
	Rank            int      `json:"rank"`
	GroupID         string   `json:"group_id"`
	TopicLabel      string   `json:"topic_label,omitempty"`
	Headline        string   `json:"headline"`
	Summary         string   `json:"summary,omitempty"`
	WhyItMatters    string   `json:"why_it_matters,omitempty"`
	Importance      string   `json:"importance"`
	SourceItemCount int      `json:"source_item_count"`
	SourceFeedCount int      `json:"source_feed_count"`
	Tags            []string `json:"tags,omitempty"`
}

type rssBriefingBuilder interface {
	Build(context.Context, []RSSInboxTopicGroup, RSSBriefingQuery) (rssBriefingDraft, error)
}

type rssBriefingDraft struct {
	Title      string                      `json:"title,omitempty"`
	Summary    string                      `json:"summary,omitempty"`
	Highlights []rssBriefingDraftHighlight `json:"highlights"`
}

type rssBriefingDraftHighlight struct {
	GroupID      string `json:"group_id"`
	Headline     string `json:"headline,omitempty"`
	Summary      string `json:"summary,omitempty"`
	WhyItMatters string `json:"why_it_matters,omitempty"`
	Importance   string `json:"importance,omitempty"`
}

type llmRSSBriefingBuilder struct {
	store   *ConfigStore
	timeout time.Duration
	client  llm.Completer
	cfg     Config
}

func (s *RSSInboxService) BuildBriefing(ctx context.Context, query RSSBriefingQuery) (RSSBriefingResult, error) {
	result, _, err := s.buildBriefingWithAggregate(ctx, query)
	return result, err
}

func (s *RSSInboxService) BuildAndStoreBriefing(ctx context.Context, query RSSBriefingQuery) (RSSBriefingResult, error) {
	if s == nil || s.briefingStore == nil {
		return RSSBriefingResult{}, fmt.Errorf("rss briefing store is not configured")
	}
	result, groups, err := s.buildBriefingWithAggregate(ctx, query)
	if err != nil {
		return RSSBriefingResult{}, err
	}
	result.SavedAt = s.currentTime().UTC()
	saved, err := s.briefingStore.Save(result)
	if err != nil {
		return RSSBriefingResult{}, err
	}
	if s.reportStore == nil || s.reportBuilder == nil {
		return saved, nil
	}
	report, err := s.BuildAndStoreReport(ctx, saved, groups, RSSReportQuery{
		TraceID: saved.TraceID,
		TaskID:  saved.TaskID,
	})
	if err != nil {
		saved.ReportError = err.Error()
		log.Printf("rss report build skipped: briefing_id=%s trace_id=%s error=%v", saved.ID, saved.TraceID, err)
		return saved, nil
	}
	saved.Report = &report
	return saved, nil
}

func (s *RSSInboxService) LatestBriefing() (RSSBriefingResult, error) {
	if s == nil || s.briefingStore == nil {
		return RSSBriefingResult{}, fmt.Errorf("rss briefing store is not configured")
	}
	return s.briefingStore.Latest()
}

func (b *llmRSSBriefingBuilder) Build(ctx context.Context, groups []RSSInboxTopicGroup, query RSSBriefingQuery) (rssBriefingDraft, error) {
	if len(groups) == 0 {
		return rssBriefingDraft{}, nil
	}
	client, cfg, err := b.workerClient()
	if err != nil {
		return rssBriefingDraft{}, err
	}
	timeout := b.timeout
	if timeout <= 0 {
		timeout = defaultRSSBriefingTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resp, err := client.Complete(runCtx, llm.CompletionRequest{Messages: []llm.Message{
		{Role: llm.RoleSystem, Text: "You are Ghost-OS RSS briefing worker. Build a concise, high-signal news briefing from grouped RSS topics. Return strict JSON only with object fields: title, summary, highlights. highlights must be an array of objects with: group_id, headline, summary, why_it_matters, importance. Use only the provided group_id values. importance must be one of low, normal, high. Do not include markdown, commentary, or code fences."},
		{Role: llm.RoleUser, Text: renderRSSBriefingPrompt(groups, query, cfg)},
	}})
	if err != nil {
		return rssBriefingDraft{}, err
	}

	text := stripJSONCodeFence(strings.TrimSpace(resp.Message.Text))
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

func (b *llmRSSBriefingBuilder) workerClient() (llm.Completer, Config, error) {
	if b != nil && b.client != nil {
		return b.client, b.cfg, nil
	}
	var (
		cfg Config
		err error
	)
	if b != nil && b.store != nil {
		cfg, err = loadConfigWithRuntime(b.store.RuntimeConfig())
	} else if b != nil && strings.TrimSpace(b.cfg.Provider.Model) != "" {
		cfg = b.cfg
	} else {
		cfg, err = LoadConfig()
	}
	if err != nil {
		return nil, Config{}, err
	}
	client := llm.NewClientWithOptions(providerClientOptions(cfg, effectiveWorkerModel(cfg)))
	if b != nil {
		b.cfg = cfg
	}
	return client, cfg, nil
}

func normalizeRSSBriefingQuery(query RSSBriefingQuery) RSSBriefingQuery {
	query.FeedID = strings.TrimSpace(query.FeedID)
	query.Tag = strings.TrimSpace(query.Tag)
	query.Importance = strings.TrimSpace(query.Importance)
	query.TraceID = strings.TrimSpace(query.TraceID)
	if query.GroupLimit <= 0 {
		query.GroupLimit = defaultRSSBriefingGroupLimit
	}
	if query.GroupLimit > maxRSSBriefingGroupLimit {
		query.GroupLimit = maxRSSBriefingGroupLimit
	}
	if query.HighlightsLimit <= 0 {
		query.HighlightsLimit = defaultRSSBriefingHighlightsLimit
	}
	if query.HighlightsLimit > maxRSSBriefingHighlightsLimit {
		query.HighlightsLimit = maxRSSBriefingHighlightsLimit
	}
	if query.ItemLimit <= 0 {
		query.ItemLimit = defaultRSSAggregateItemLimit
	}
	if query.ItemsPerGroup <= 0 {
		query.ItemsPerGroup = 3
	}
	return query
}

func (s *RSSInboxService) buildBriefingWithAggregate(ctx context.Context, query RSSBriefingQuery) (RSSBriefingResult, []RSSInboxTopicGroup, error) {
	if s == nil || s.inboxStore == nil {
		return RSSBriefingResult{}, nil, fmt.Errorf("rss inbox service is not configured")
	}
	if s.briefingBuilder == nil {
		return RSSBriefingResult{}, nil, fmt.Errorf("rss briefing builder is not configured")
	}

	query = normalizeRSSBriefingQuery(query)
	aggregate, err := s.Aggregate(RSSInboxGroupQuery{
		FeedID:        query.FeedID,
		Tag:           query.Tag,
		Importance:    query.Importance,
		WindowHours:   query.WindowHours,
		Limit:         query.GroupLimit,
		ItemLimit:     query.ItemLimit,
		ItemsPerGroup: query.ItemsPerGroup,
	})
	if err != nil {
		return RSSBriefingResult{}, nil, err
	}

	result := RSSBriefingResult{
		Title:         "RSS Briefing",
		GeneratedAt:   s.currentTime().UTC(),
		WindowHours:   aggregate.WindowHours,
		ScannedGroups: len(aggregate.Groups),
		TraceID:       strings.TrimSpace(query.TraceID),
		TaskID:        strings.TrimSpace(query.TaskID),
		Highlights:    []RSSBriefingHighlight{},
	}
	if len(aggregate.Groups) == 0 {
		result.Summary = "No notable items matched the current RSS briefing window."
		return result, aggregate.Groups, nil
	}

	draft, err := s.briefingBuilder.Build(ctx, aggregate.Groups, query)
	if err != nil {
		return RSSBriefingResult{}, nil, err
	}
	result.Title = rssBriefingTitleOrDefault(strings.TrimSpace(draft.Title), aggregate.WindowHours)
	result.Summary = truncateRunes(strings.TrimSpace(draft.Summary), 400)
	result.Highlights = materializeRSSBriefingHighlights(draft.Highlights, aggregate.Groups, query.HighlightsLimit)
	result.HighlightCount = len(result.Highlights)
	if result.Summary == "" {
		result.Summary = fallbackRSSBriefingSummary(result.Highlights)
	}
	return result, aggregate.Groups, nil
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

func materializeRSSBriefingHighlights(draft []rssBriefingDraftHighlight, groups []RSSInboxTopicGroup, limit int) []RSSBriefingHighlight {
	if limit <= 0 {
		limit = defaultRSSBriefingHighlightsLimit
	}
	groupByID := make(map[string]RSSInboxTopicGroup, len(groups))
	for _, group := range groups {
		groupByID[group.ID] = group
	}

	out := make([]RSSBriefingHighlight, 0, limit)
	seen := make(map[string]struct{}, limit)
	for _, item := range draft {
		group, ok := groupByID[item.GroupID]
		if !ok {
			continue
		}
		if _, exists := seen[group.ID]; exists {
			continue
		}
		seen[group.ID] = struct{}{}
		out = append(out, RSSBriefingHighlight{
			Rank:            len(out) + 1,
			GroupID:         group.ID,
			TopicLabel:      group.TopicLabel,
			Headline:        firstNonEmptyString(item.Headline, group.Headline, group.TopicLabel),
			Summary:         firstNonEmptyString(item.Summary, group.Summary),
			WhyItMatters:    truncateRunes(strings.TrimSpace(item.WhyItMatters), 220),
			Importance:      normalizeRSSInboxImportance(firstNonEmptyString(item.Importance, group.Importance)),
			SourceItemCount: group.ItemCount,
			SourceFeedCount: group.FeedCount,
			Tags:            append([]string(nil), group.Tags...),
		})
		if len(out) >= limit {
			break
		}
	}

	for _, group := range groups {
		if _, exists := seen[group.ID]; exists {
			continue
		}
		out = append(out, RSSBriefingHighlight{
			Rank:            len(out) + 1,
			GroupID:         group.ID,
			TopicLabel:      group.TopicLabel,
			Headline:        firstNonEmptyString(group.Headline, group.TopicLabel),
			Summary:         group.Summary,
			Importance:      normalizeRSSInboxImportance(group.Importance),
			SourceItemCount: group.ItemCount,
			SourceFeedCount: group.FeedCount,
			Tags:            append([]string(nil), group.Tags...),
		})
		if len(out) >= limit {
			break
		}
	}
	return out
}

func fallbackRSSBriefingSummary(highlights []RSSBriefingHighlight) string {
	if len(highlights) == 0 {
		return ""
	}
	if len(highlights) == 1 {
		return firstNonEmptyString(highlights[0].Summary, highlights[0].Headline)
	}
	return fmt.Sprintf("%d notable RSS developments selected for this briefing.", len(highlights))
}

func rssBriefingTitleOrDefault(input string, windowHours int) string {
	if strings.TrimSpace(input) != "" {
		return input
	}
	return fmt.Sprintf("RSS Briefing (%dh)", windowHours)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
