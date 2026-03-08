package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

const (
	defaultRSSClassifierTimeout = 45 * time.Second
	defaultRSSPollTaskID        = "system-rss-inbox-poll"
	maxRSSClassifierSummaryLen  = 280
	defaultRSSFetchMaxItems     = 10
	maxRSSFetchMaxItems         = 50
)

type RSSInboxPollOptions struct {
	MaxItemsPerFeed int
	AIBatchSize     int
	TraceID         string
	TaskID          string
}

type RSSInboxPollResult struct {
	FeedsScanned    int    `json:"feeds_scanned"`
	FeedsFailed     int    `json:"feeds_failed"`
	ItemsFetched    int    `json:"items_fetched"`
	ItemsCandidates int    `json:"items_candidates"`
	ItemsSaved      int    `json:"items_saved"`
	ItemsDiscarded  int    `json:"items_discarded"`
	TraceID         string `json:"trace_id,omitempty"`
	TaskID          string `json:"task_id,omitempty"`
}

type rssInboxFetcher interface {
	Fetch(context.Context, string, int) (tools.RSSResult, error)
}

type rssInboxClassifier interface {
	Classify(context.Context, tools.FeedSubscription, []rssInboxCandidate, string) ([]rssInboxClassification, error)
}

type RSSInboxService struct {
	feedStore       *tools.FeedStore
	inboxStore      *RSSInboxStore
	fetcher         rssInboxFetcher
	classifier      rssInboxClassifier
	now             func() time.Time
	maxItemsPerFeed int
	aiBatchSize     int
}

type rssInboxCandidate struct {
	FeedID       string
	FeedURL      string
	SourceTitle  string
	FeedTags     []string
	ItemTitle    string
	ItemLink     string
	RawSummary   string
	PublishedAt  time.Time
	DedupeKey    string
	ContentHash  string
	SourceFeedID string
}

type rssInboxClassification struct {
	Index      int      `json:"index"`
	Keep       bool     `json:"keep"`
	AISummary  string   `json:"ai_summary,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Importance string   `json:"importance,omitempty"`
	Reason     string   `json:"reason,omitempty"`
}

type defaultRSSInboxFetcher struct{}

type llmRSSInboxClassifier struct {
	store   *ConfigStore
	timeout time.Duration
	client  llm.Completer
	cfg     Config
}

func NewRSSInboxService(feedStore *tools.FeedStore, inboxStore *RSSInboxStore, classifier rssInboxClassifier, cfg Config) *RSSInboxService {
	if classifier == nil {
		classifier = &llmRSSInboxClassifier{store: nil, timeout: defaultRSSClassifierTimeout, cfg: cfg}
	}
	service := &RSSInboxService{
		feedStore:       feedStore,
		inboxStore:      inboxStore,
		fetcher:         defaultRSSInboxFetcher{},
		classifier:      classifier,
		now:             time.Now,
		maxItemsPerFeed: normalizeRSSPollMaxItems(cfg.RSSPollMaxItemsPerFeed),
		aiBatchSize:     normalizeRSSAIBatchSize(cfg.RSSAIBatchSize),
	}
	if llmClassifier, ok := classifier.(*llmRSSInboxClassifier); ok && llmClassifier.store == nil && llmClassifier.client == nil {
		llmClassifier.store = nil
		llmClassifier.cfg = cfg
	}
	return service
}

func newRSSInboxServiceFromConfig(store *ConfigStore) (*RSSInboxService, error) {
	var (
		cfg Config
		err error
	)
	if store != nil {
		cfg, err = loadConfigWithRuntime(store.RuntimeConfig())
	} else {
		cfg, err = LoadConfig()
	}
	if err != nil {
		return nil, err
	}
	feedStore, err := tools.NewFeedStore(cfg.RSSFeedsPath)
	if err != nil {
		return nil, err
	}
	inboxStore, err := NewRSSInboxStore(cfg.RSSInboxPath)
	if err != nil {
		return nil, err
	}
	classifier := &llmRSSInboxClassifier{store: store, timeout: defaultRSSClassifierTimeout, cfg: cfg}
	return NewRSSInboxService(feedStore, inboxStore, classifier, cfg), nil
}

func (s *RSSInboxService) Poll(ctx context.Context, opts RSSInboxPollOptions) (RSSInboxPollResult, error) {
	if s == nil || s.feedStore == nil || s.inboxStore == nil || s.fetcher == nil || s.classifier == nil {
		return RSSInboxPollResult{}, fmt.Errorf("rss inbox service is not configured")
	}
	enabled := true
	feeds, err := s.feedStore.List(tools.FeedListFilter{Enabled: &enabled})
	if err != nil {
		return RSSInboxPollResult{}, err
	}
	knownKeys, err := s.inboxStore.KnownDedupeKeys()
	if err != nil {
		return RSSInboxPollResult{}, err
	}
	result := RSSInboxPollResult{
		TraceID: strings.TrimSpace(opts.TraceID),
		TaskID:  strings.TrimSpace(opts.TaskID),
	}
	maxItemsPerFeed := normalizeRSSPollMaxItems(firstPositive(opts.MaxItemsPerFeed, s.maxItemsPerFeed))
	aiBatchSize := normalizeRSSAIBatchSize(firstPositive(opts.AIBatchSize, s.aiBatchSize))

	for _, feed := range feeds {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		result.FeedsScanned++
		fetched, err := s.fetcher.Fetch(ctx, feed.URL, maxItemsPerFeed)
		if err != nil {
			result.FeedsFailed++
			log.Printf("rss inbox poll fetch failed: feed_id=%s url=%s trace_id=%s error=%v", feed.ID, feed.URL, result.TraceID, err)
			continue
		}
		result.ItemsFetched += len(fetched.Items)
		candidates := make([]rssInboxCandidate, 0, len(fetched.Items))
		for _, item := range fetched.Items {
			candidate := buildRSSInboxCandidate(feed, item, fetched.Feed.Title)
			if candidate.DedupeKey == "" {
				continue
			}
			if _, exists := knownKeys[candidate.DedupeKey]; exists {
				continue
			}
			candidates = append(candidates, candidate)
		}
		result.ItemsCandidates += len(candidates)
		for start := 0; start < len(candidates); start += aiBatchSize {
			end := start + aiBatchSize
			if end > len(candidates) {
				end = len(candidates)
			}
			batch := candidates[start:end]
			batchKeys := make([]string, 0, len(batch))
			for _, candidate := range batch {
				batchKeys = append(batchKeys, candidate.DedupeKey)
			}
			decisions, err := s.classifier.Classify(ctx, feed, batch, result.TraceID)
			if err != nil {
				if markErr := s.inboxStore.MarkSeen(batchKeys); markErr != nil {
					return result, markErr
				}
				result.ItemsDiscarded += len(batch)
				log.Printf("rss inbox poll classify failed: feed_id=%s trace_id=%s error=%v", feed.ID, result.TraceID, err)
				continue
			}
			itemsToSave := buildRSSInboxItems(batch, decisions, result.TraceID, s.currentTime())
			saved, err := s.inboxStore.SaveItems(itemsToSave)
			if err != nil {
				return result, err
			}
			if err := s.inboxStore.MarkSeen(batchKeys); err != nil {
				return result, err
			}
			for _, item := range saved {
				knownKeys[item.DedupeKey] = struct{}{}
			}
			for _, key := range batchKeys {
				knownKeys[key] = struct{}{}
			}
			result.ItemsSaved += len(saved)
			result.ItemsDiscarded += len(batch) - len(saved)
		}
	}
	return result, nil
}

func (s *RSSInboxService) List(filter RSSInboxListFilter) ([]RSSInboxItem, error) {
	if s == nil || s.inboxStore == nil {
		return nil, fmt.Errorf("rss inbox service is not configured")
	}
	return s.inboxStore.List(filter)
}

func (s *RSSInboxService) Get(id string) (RSSInboxItem, error) {
	if s == nil || s.inboxStore == nil {
		return RSSInboxItem{}, fmt.Errorf("rss inbox service is not configured")
	}
	return s.inboxStore.Get(id)
}

func (defaultRSSInboxFetcher) Fetch(ctx context.Context, feedURL string, maxItems int) (tools.RSSResult, error) {
	return tools.FetchRSS(ctx, feedURL, tools.RSSFetchOptions{MaxItems: maxItems, IncludeSummary: true})
}

func (c *llmRSSInboxClassifier) Classify(ctx context.Context, feed tools.FeedSubscription, items []rssInboxCandidate, traceID string) ([]rssInboxClassification, error) {
	if len(items) == 0 {
		return nil, nil
	}
	client, cfg, err := c.workerClient()
	if err != nil {
		return nil, err
	}
	timeout := c.timeout
	if timeout <= 0 {
		timeout = defaultRSSClassifierTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	resp, err := client.Complete(runCtx, llm.CompletionRequest{Messages: []llm.Message{
		{Role: llm.RoleSystem, Text: "You are Ghost-OS RSS inbox worker. Review RSS items and keep only items that are worth a human's attention. Favor concrete announcements, product or model releases, security or policy changes, useful tutorials, sharp analysis, and notable ecosystem signals. Reject obvious spam, repetitive low-signal headlines, and thin updates. Return strict JSON array only with objects: index, keep, ai_summary, tags, importance, reason. importance must be one of low, normal, high. ai_summary and reason must stay concise."},
		{Role: llm.RoleUser, Text: renderRSSInboxClassificationPrompt(feed, items, cfg, traceID)},
	}})
	if err != nil {
		return nil, err
	}
	text := stripJSONCodeFence(strings.TrimSpace(resp.Message.Text))
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

func (c *llmRSSInboxClassifier) workerClient() (llm.Completer, Config, error) {
	if c != nil && c.client != nil {
		return c.client, c.cfg, nil
	}
	var (
		cfg Config
		err error
	)
	if c != nil && c.store != nil {
		cfg, err = loadConfigWithRuntime(c.store.RuntimeConfig())
	} else if c != nil && strings.TrimSpace(c.cfg.Model) != "" {
		cfg = c.cfg
	} else {
		cfg, err = LoadConfig()
	}
	if err != nil {
		return nil, Config{}, err
	}
	client := llm.NewClientWithOptions(llm.ClientOptions{
		Provider:           cfg.Provider,
		BaseURL:            cfg.BaseURL,
		APIKey:             cfg.APIKey,
		Model:              effectiveWorkerModel(cfg),
		ChatPath:           cfg.ChatPath,
		Headers:            cfg.ProviderHeaders,
		AnthropicVersion:   cfg.AnthropicVersion,
		AnthropicMaxTokens: cfg.AnthropicMaxTokens,
	})
	if c != nil {
		c.cfg = cfg
	}
	return client, cfg, nil
}

func renderRSSInboxClassificationPrompt(feed tools.FeedSubscription, items []rssInboxCandidate, cfg Config, traceID string) string {
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
		promptItem := promptItem{
			Index:    index,
			Title:    item.ItemTitle,
			Link:     item.ItemLink,
			Summary:  item.RawSummary,
			FeedTags: append([]string(nil), item.FeedTags...),
		}
		if !item.PublishedAt.IsZero() {
			promptItem.PublishedAt = item.PublishedAt.UTC().Format(time.RFC3339)
		}
		payload.Items = append(payload.Items, promptItem)
	}
	encoded, _ := json.MarshalIndent(payload, "", "  ")
	return string(encoded)
}

func buildRSSInboxCandidate(feed tools.FeedSubscription, item tools.RSSItem, fetchedSourceTitle string) rssInboxCandidate {
	publishedAt, _ := parseRSSInboxTime(item.PublishedAt)
	sourceTitle := strings.TrimSpace(item.SourceTitle)
	if sourceTitle == "" {
		sourceTitle = strings.TrimSpace(feed.Title)
	}
	if sourceTitle == "" {
		sourceTitle = strings.TrimSpace(fetchedSourceTitle)
	}
	dedupeKey := buildRSSInboxDedupeKey(feed.ID, item)
	return rssInboxCandidate{
		FeedID:       feed.ID,
		SourceFeedID: feed.ID,
		FeedURL:      feed.URL,
		SourceTitle:  sourceTitle,
		FeedTags:     append([]string(nil), feed.Tags...),
		ItemTitle:    strings.TrimSpace(item.Title),
		ItemLink:     strings.TrimSpace(item.Link),
		RawSummary:   strings.TrimSpace(item.Summary),
		PublishedAt:  publishedAt,
		DedupeKey:    dedupeKey,
		ContentHash:  buildRSSInboxContentHash(feed.ID, item),
	}
}

func buildRSSInboxItems(batch []rssInboxCandidate, decisions []rssInboxClassification, traceID string, savedAt time.Time) []RSSInboxItem {
	decisionByIndex := make(map[int]rssInboxClassification, len(decisions))
	for _, decision := range decisions {
		decisionByIndex[decision.Index] = decision
	}
	items := make([]RSSInboxItem, 0, len(batch))
	for index, candidate := range batch {
		decision, ok := decisionByIndex[index]
		if !ok || !decision.Keep {
			continue
		}
		tags := normalizeRSSInboxTags(append(append([]string(nil), candidate.FeedTags...), decision.Tags...))
		items = append(items, RSSInboxItem{
			FeedID:       candidate.FeedID,
			SourceFeedID: candidate.SourceFeedID,
			FeedURL:      candidate.FeedURL,
			SourceTitle:  candidate.SourceTitle,
			ItemTitle:    candidate.ItemTitle,
			ItemLink:     candidate.ItemLink,
			PublishedAt:  candidate.PublishedAt,
			RawSummary:   candidate.RawSummary,
			AISummary:    truncateRunes(strings.TrimSpace(decision.AISummary), maxRSSClassifierSummaryLen),
			Tags:         tags,
			Importance:   normalizeRSSInboxImportance(decision.Importance),
			Reason:       truncateRunes(strings.TrimSpace(decision.Reason), 180),
			ContentHash:  candidate.ContentHash,
			DedupeKey:    candidate.DedupeKey,
			TraceID:      strings.TrimSpace(traceID),
			SavedAt:      savedAt.UTC(),
		})
	}
	return items
}

func buildRSSInboxDedupeKey(feedID string, item tools.RSSItem) string {
	base := strings.TrimSpace(item.ID)
	if base != "" {
		return strings.TrimSpace(feedID) + "::id::" + base
	}
	if link := strings.TrimSpace(item.Link); link != "" {
		return strings.TrimSpace(feedID) + "::link::" + link
	}
	hash := sha256.Sum256([]byte(strings.TrimSpace(item.Title) + "\n" + strings.TrimSpace(item.PublishedAt)))
	return strings.TrimSpace(feedID) + "::hash::" + hex.EncodeToString(hash[:])
}

func buildRSSInboxContentHash(feedID string, item tools.RSSItem) string {
	payload := []string{
		strings.TrimSpace(feedID),
		strings.TrimSpace(item.ID),
		strings.TrimSpace(item.Title),
		strings.TrimSpace(item.Link),
		strings.TrimSpace(item.PublishedAt),
		strings.TrimSpace(item.Summary),
	}
	hash := sha256.Sum256([]byte(strings.Join(payload, "\n")))
	return hex.EncodeToString(hash[:])
}

func parseRSSInboxTime(raw string) (time.Time, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func normalizeRSSPollMaxItems(value int) int {
	if value <= 0 {
		return defaultRSSFetchMaxItems
	}
	if value > maxRSSFetchMaxItems {
		return maxRSSFetchMaxItems
	}
	return value
}

func normalizeRSSAIBatchSize(value int) int {
	if value <= 0 {
		return defaultRSSAIBatchSize
	}
	if value > 20 {
		return 20
	}
	return value
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func (s *RSSInboxService) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}
