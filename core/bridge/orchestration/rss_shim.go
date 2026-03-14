package orchestration

import (
	"strings"

	"ghost-os/bridge/llm"
	bridgerss "ghost-os/bridge/rss"
	"ghost-os/bridge/tools"
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

var (
	ErrRSSInboxItemNotFound = bridgerss.ErrRSSInboxItemNotFound
	ErrRSSBriefingNotFound  = bridgerss.ErrRSSBriefingNotFound
)

const (
	defaultRSSPollTaskID             = bridgerss.DefaultRSSPollTaskID
	defaultRSSBriefingTaskID         = bridgerss.DefaultRSSBriefingTaskID
	defaultRSSAggregateWindowHours   = bridgerss.DefaultRSSAggregateWindowHours
	defaultRSSAggregateItemLimit     = bridgerss.DefaultRSSAggregateItemLimit
	defaultRSSBriefingGroupLimit     = bridgerss.DefaultRSSBriefingGroupLimit
	defaultRSSBriefingHighlightsLimit = bridgerss.DefaultRSSBriefingHighlightsLimit
)

func newRSSInboxServiceFromConfig(store *ConfigStore) (*RSSInboxService, error) {
	if store == nil {
		return bridgerss.NewRSSInboxServiceFromConfig(nil)
	}
	return bridgerss.NewRSSInboxServiceFromConfig(store.Inner())
}

func NewRSSInboxService(feedStore *tools.FeedStore, inboxStore *RSSInboxStore, briefingStore *RSSBriefingStore, reportStore *RSSReportStore, classifier RSSInboxClassifier, cfg Config) *RSSInboxService {
	return bridgerss.NewRSSInboxService(feedStore, inboxStore, briefingStore, reportStore, classifier, cfg)
}

func NewLLMRSSBriefingBuilder(client llm.Completer, cfg Config) RSSBriefingBuilder {
	return bridgerss.NewLLMRSSBriefingBuilder(client, cfg)
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

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
