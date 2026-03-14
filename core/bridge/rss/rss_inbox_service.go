package rss

import (
	"context"
	"fmt"
	"time"

	rsssubscriptions "ghost-os/bridge/rss/subscriptions"
)

const defaultRSSPollTaskID = "system-rss-inbox-poll"

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
	Fetch(context.Context, string, int) (RSSResult, error)
}

type rssInboxClassifier interface {
	Classify(context.Context, rsssubscriptions.FeedSubscription, []rssInboxCandidate, string) ([]rssInboxClassification, error)
}

type RSSInboxService struct {
	feedStore       *rsssubscriptions.FeedStore
	inboxStore      *RSSInboxStore
	briefingStore   *RSSBriefingStore
	reportStore     *RSSReportStore
	fetcher         rssInboxFetcher
	classifier      rssInboxClassifier
	briefingBuilder rssBriefingBuilder
	reportBuilder   rssReportBuilder
	now             func() time.Time
	maxItemsPerFeed int
	aiBatchSize     int
}

func NewRSSInboxService(
	feedStore *rsssubscriptions.FeedStore,
	inboxStore *RSSInboxStore,
	briefingStore *RSSBriefingStore,
	reportStore *RSSReportStore,
	classifier rssInboxClassifier,
	cfg Config,
) *RSSInboxService {
	if classifier == nil {
		classifier = &llmRSSInboxClassifier{timeout: defaultRSSClassifierTimeout, cfg: cfg}
	}
	service := &RSSInboxService{
		feedStore:       feedStore,
		inboxStore:      inboxStore,
		briefingStore:   briefingStore,
		reportStore:     reportStore,
		fetcher:         defaultRSSInboxFetcher{},
		classifier:      classifier,
		briefingBuilder: &llmRSSBriefingBuilder{timeout: defaultRSSBriefingTimeout, cfg: cfg},
		reportBuilder:   &agentRSSReportBuilder{timeout: defaultRSSReportTimeout},
		now:             time.Now,
		maxItemsPerFeed: normalizeRSSPollMaxItems(cfg.RSS.PollMaxItemsPerFeed),
		aiBatchSize:     normalizeRSSAIBatchSize(cfg.RSS.AIBatchSize),
	}
	if llmClassifier, ok := classifier.(*llmRSSInboxClassifier); ok && llmClassifier.store == nil && llmClassifier.client == nil {
		llmClassifier.cfg = cfg
	}
	return service
}

func (s *RSSInboxService) Poll(ctx context.Context, opts RSSInboxPollOptions) (RSSInboxPollResult, error) {
	return newRSSInboxPollWorkflow(s.pollDependencies()).Run(ctx, opts)
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

func (s *RSSInboxService) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}
