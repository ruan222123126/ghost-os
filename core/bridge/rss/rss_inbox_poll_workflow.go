package rss

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	rsssubscriptions "ghost-os/bridge/rss/subscriptions"
)

const maxRSSFetchMaxItems = 50

type rssInboxFeedLister interface {
	List(rsssubscriptions.FeedListFilter) ([]rsssubscriptions.FeedSubscription, error)
}

type rssInboxPollStore interface {
	KnownDedupeKeys() (map[string]struct{}, error)
	SaveItems([]RSSInboxItem) ([]RSSInboxItem, error)
	MarkSeen([]string) error
}

type rssInboxPollDependencies struct {
	feedStore       rssInboxFeedLister
	inboxStore      rssInboxPollStore
	fetcher         rssInboxFetcher
	classifier      rssInboxClassifier
	now             func() time.Time
	maxItemsPerFeed int
	aiBatchSize     int
}

type rssInboxPollRun struct {
	feeds           []rsssubscriptions.FeedSubscription
	knownKeys       map[string]struct{}
	result          RSSInboxPollResult
	maxItemsPerFeed int
	aiBatchSize     int
}

type rssInboxPollWorkflow struct {
	deps rssInboxPollDependencies
}

func (s *RSSInboxService) pollDependencies() rssInboxPollDependencies {
	if s == nil {
		return rssInboxPollDependencies{}
	}
	return rssInboxPollDependencies{
		feedStore:       s.feedStore,
		inboxStore:      s.inboxStore,
		fetcher:         s.fetcher,
		classifier:      s.classifier,
		now:             s.now,
		maxItemsPerFeed: s.maxItemsPerFeed,
		aiBatchSize:     s.aiBatchSize,
	}
}

func newRSSInboxPollWorkflow(deps rssInboxPollDependencies) rssInboxPollWorkflow {
	return rssInboxPollWorkflow{deps: deps}
}

func (w rssInboxPollWorkflow) Run(ctx context.Context, opts RSSInboxPollOptions) (RSSInboxPollResult, error) {
	if err := w.validate(); err != nil {
		return RSSInboxPollResult{}, err
	}
	run, err := w.prepareRun(opts)
	if err != nil {
		return RSSInboxPollResult{}, err
	}
	for _, feed := range run.feeds {
		if err := ctx.Err(); err != nil {
			return run.result, err
		}
		if err := w.pollFeed(ctx, feed, &run); err != nil {
			return run.result, err
		}
	}
	return run.result, nil
}

func (w rssInboxPollWorkflow) validate() error {
	if w.deps.feedStore == nil || w.deps.inboxStore == nil || w.deps.fetcher == nil || w.deps.classifier == nil {
		return fmt.Errorf("rss inbox service is not configured")
	}
	return nil
}

func (w rssInboxPollWorkflow) prepareRun(opts RSSInboxPollOptions) (rssInboxPollRun, error) {
	enabled := true
	feeds, err := w.deps.feedStore.List(rsssubscriptions.FeedListFilter{Enabled: &enabled})
	if err != nil {
		return rssInboxPollRun{}, err
	}
	knownKeys, err := w.deps.inboxStore.KnownDedupeKeys()
	if err != nil {
		return rssInboxPollRun{}, err
	}
	return rssInboxPollRun{
		feeds:     feeds,
		knownKeys: knownKeys,
		result: RSSInboxPollResult{
			TraceID: strings.TrimSpace(opts.TraceID),
			TaskID:  strings.TrimSpace(opts.TaskID),
		},
		maxItemsPerFeed: normalizeRSSPollMaxItems(firstPositive(opts.MaxItemsPerFeed, w.deps.maxItemsPerFeed)),
		aiBatchSize:     normalizeRSSAIBatchSize(firstPositive(opts.AIBatchSize, w.deps.aiBatchSize)),
	}, nil
}

func (w rssInboxPollWorkflow) pollFeed(
	ctx context.Context,
	feed rsssubscriptions.FeedSubscription,
	run *rssInboxPollRun,
) error {
	run.result.FeedsScanned++
	fetched, err := w.deps.fetcher.Fetch(ctx, feed.URL, run.maxItemsPerFeed)
	if err != nil {
		run.result.FeedsFailed++
		log.Printf("rss inbox poll fetch failed: feed_id=%s url=%s trace_id=%s error=%v", feed.ID, feed.URL, run.result.TraceID, err)
		return nil
	}
	run.result.ItemsFetched += len(fetched.Items)
	candidates := w.collectCandidates(feed, fetched, run.knownKeys)
	run.result.ItemsCandidates += len(candidates)
	return w.processBatches(ctx, feed, candidates, run)
}

func (w rssInboxPollWorkflow) collectCandidates(
	feed rsssubscriptions.FeedSubscription,
	fetched RSSResult,
	knownKeys map[string]struct{},
) []rssInboxCandidate {
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
	return candidates
}

func (w rssInboxPollWorkflow) processBatches(
	ctx context.Context,
	feed rsssubscriptions.FeedSubscription,
	candidates []rssInboxCandidate,
	run *rssInboxPollRun,
) error {
	for start := 0; start < len(candidates); start += run.aiBatchSize {
		end := start + run.aiBatchSize
		if end > len(candidates) {
			end = len(candidates)
		}
		if err := w.processBatch(ctx, feed, candidates[start:end], run); err != nil {
			return err
		}
	}
	return nil
}

func (w rssInboxPollWorkflow) processBatch(
	ctx context.Context,
	feed rsssubscriptions.FeedSubscription,
	batch []rssInboxCandidate,
	run *rssInboxPollRun,
) error {
	batchKeys := collectRSSInboxDedupeKeys(batch)
	decisions, err := w.deps.classifier.Classify(ctx, feed, batch, run.result.TraceID)
	if err != nil {
		if markErr := w.deps.inboxStore.MarkSeen(batchKeys); markErr != nil {
			return markErr
		}
		run.result.ItemsDiscarded += len(batch)
		log.Printf("rss inbox poll classify failed: feed_id=%s trace_id=%s error=%v", feed.ID, run.result.TraceID, err)
		return nil
	}
	itemsToSave := buildRSSInboxItems(batch, decisions, run.result.TraceID, w.currentTime())
	saved, err := w.deps.inboxStore.SaveItems(itemsToSave)
	if err != nil {
		return err
	}
	if err := w.deps.inboxStore.MarkSeen(batchKeys); err != nil {
		return err
	}
	run.result.ItemsSaved += len(saved)
	run.result.ItemsDiscarded += len(batch) - len(saved)
	markRSSInboxSeenKeys(run.knownKeys, batchKeys)
	markRSSInboxSavedItems(run.knownKeys, saved)
	return nil
}

func (w rssInboxPollWorkflow) currentTime() time.Time {
	if w.deps.now != nil {
		return w.deps.now()
	}
	return time.Now()
}

func collectRSSInboxDedupeKeys(batch []rssInboxCandidate) []string {
	keys := make([]string, 0, len(batch))
	for _, candidate := range batch {
		keys = append(keys, candidate.DedupeKey)
	}
	return keys
}

func markRSSInboxSeenKeys(knownKeys map[string]struct{}, keys []string) {
	for _, key := range keys {
		knownKeys[key] = struct{}{}
	}
}

func markRSSInboxSavedItems(knownKeys map[string]struct{}, items []RSSInboxItem) {
	for _, item := range items {
		knownKeys[item.DedupeKey] = struct{}{}
	}
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
