package orchestration

import (
	bridgeconfig "ghost-os/bridge/config"
	bridgerss "ghost-os/bridge/rss"
	rsssubscriptions "ghost-os/bridge/rss/subscriptions"
)

type RSSInboxPollOptions = bridgerss.RSSInboxPollOptions
type RSSInboxPollResult = bridgerss.RSSInboxPollResult
type RSSInboxService = bridgerss.RSSInboxService
type RSSInboxFetcher = bridgerss.RSSInboxFetcher
type RSSInboxClassifier = bridgerss.RSSInboxClassifier
type RSSInboxClassification = bridgerss.RSSInboxClassification
type RSSInboxCandidate = bridgerss.RSSInboxCandidate
type RSSBriefingQuery = bridgerss.RSSBriefingQuery
type RSSBriefingResult = bridgerss.RSSBriefingResult
type RSSBriefingHighlight = bridgerss.RSSBriefingHighlight
type RSSBriefingBuilder = bridgerss.RSSBriefingBuilder
type RSSBriefingDraft = bridgerss.RSSBriefingDraft
type RSSBriefingDraftHighlight = bridgerss.RSSBriefingDraftHighlight
type RSSInboxGroupQuery = bridgerss.RSSInboxGroupQuery
type RSSInboxGroupResult = bridgerss.RSSInboxGroupResult
type RSSInboxTopicGroup = bridgerss.RSSInboxTopicGroup
type RSSInboxListFilter = bridgerss.RSSInboxListFilter
type RSSInboxItem = bridgerss.RSSInboxItem
type RSSBriefingStore = bridgerss.RSSBriefingStore
type RSSReportStore = bridgerss.RSSReportStore
type RSSInboxStore = bridgerss.RSSInboxStore
type RSSReportQuery = bridgerss.RSSReportQuery
type RSSReportResult = bridgerss.RSSReportResult
type RSSFeedStore = rsssubscriptions.FeedStore
type RSSBriefingCompleter = bridgerss.RSSBriefingCompleter
type RSSActionHandler = bridgerss.ActionHandler
type RSSSystemTaskCoordinator = bridgerss.SystemTaskCoordinator
type RSSLogFunc = bridgerss.LogFunc

// Re-exported RSS param types from bridge/rss for backward compatibility.
type RSSInboxPollParams = bridgerss.InboxPollParams
type RSSInboxListParams = bridgerss.InboxListParams
type RSSInboxGetParams = bridgerss.InboxGetParams
type RSSInboxGroupsParams = bridgerss.InboxGroupsParams
type RSSBriefingParams = bridgerss.BriefingParams

var (
	ErrRSSInboxItemNotFound = bridgerss.ErrRSSInboxItemNotFound
	ErrRSSBriefingNotFound  = bridgerss.ErrRSSBriefingNotFound
)

const (
	defaultRSSPollTaskID              = bridgerss.DefaultRSSPollTaskID
	defaultRSSBriefingTaskID          = bridgerss.DefaultRSSBriefingTaskID
	defaultRSSAggregateWindowHours    = bridgerss.DefaultRSSAggregateWindowHours
	defaultRSSAggregateItemLimit      = bridgerss.DefaultRSSAggregateItemLimit
	defaultRSSBriefingGroupLimit      = bridgerss.DefaultRSSBriefingGroupLimit
	defaultRSSBriefingHighlightsLimit = bridgerss.DefaultRSSBriefingHighlightsLimit
)

// Re-exported RSS action constants from bridge/rss.
const (
	busActionRSSInboxPoll     = bridgerss.ActionInboxPoll
	busActionRSSInboxList     = bridgerss.ActionInboxList
	busActionRSSInboxGet      = bridgerss.ActionInboxGet
	busActionRSSInboxGroups   = bridgerss.ActionInboxGroups
	busActionRSSBriefingBuild = bridgerss.ActionBriefingBuild
	busActionRSSBriefingGet   = bridgerss.ActionBriefingGet
)

func newRSSInboxServiceFromConfig(store bridgeconfig.Store) (*RSSInboxService, error) {
	return bridgerss.NewRSSInboxServiceFromConfig(store)
}

func NewRSSInboxService(feedStore *RSSFeedStore, inboxStore *RSSInboxStore, briefingStore *RSSBriefingStore, reportStore *RSSReportStore, classifier RSSInboxClassifier, cfg bridgeconfig.Config) *RSSInboxService {
	return bridgerss.NewRSSInboxService(feedStore, inboxStore, briefingStore, reportStore, classifier, cfg)
}

func NewLLMRSSBriefingBuilder(client RSSBriefingCompleter, cfg bridgeconfig.Config) RSSBriefingBuilder {
	return bridgerss.NewLLMRSSBriefingBuilder(client, cfg)
}

func NewRSSActionHandler(inbox *RSSInboxService, initErr error, log RSSLogFunc) *RSSActionHandler {
	return bridgerss.NewActionHandler(inbox, initErr, log)
}

func NewRSSSystemTaskCoordinator(
	cfg bridgeconfig.Store,
	taskStore *TaskStore,
	scheduler *TaskScheduler,
	initErr error,
) *RSSSystemTaskCoordinator {
	return bridgerss.NewSystemTaskCoordinator(cfg, taskStore, scheduler, initErr)
}

func NewRSSInboxStore(path string) (*RSSInboxStore, error) {
	return bridgerss.NewRSSInboxStore(path)
}

func NewRSSBriefingStore(path string) (*RSSBriefingStore, error) {
	return bridgerss.NewRSSBriefingStore(path)
}

func NewRSSReportStore(path string) (*RSSReportStore, error) {
	return bridgerss.NewRSSReportStore(path)
}
