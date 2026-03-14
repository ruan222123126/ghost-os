package transport

import (
	"ghost-os/bridge/llm"
	bridgeorchestration "ghost-os/bridge/orchestration"
	"ghost-os/bridge/tools"
)

type RSSInboxService = bridgeorchestration.RSSInboxService
type RSSInboxFetcher = bridgeorchestration.RSSInboxFetcher
type RSSInboxClassifier = bridgeorchestration.RSSInboxClassifier
type RSSInboxClassification = bridgeorchestration.RSSInboxClassification
type RSSInboxCandidate = bridgeorchestration.RSSInboxCandidate
type RSSInboxStore = bridgeorchestration.RSSInboxStore
type RSSBriefingStore = bridgeorchestration.RSSBriefingStore
type RSSReportStore = bridgeorchestration.RSSReportStore
type RSSBriefingBuilder = bridgeorchestration.RSSBriefingBuilder

func NewRSSInboxService(
	feedStore *tools.FeedStore,
	inboxStore *RSSInboxStore,
	briefingStore *RSSBriefingStore,
	reportStore *RSSReportStore,
	classifier RSSInboxClassifier,
	cfg Config,
) *RSSInboxService {
	return bridgeorchestration.NewRSSInboxService(feedStore, inboxStore, briefingStore, reportStore, classifier, cfg)
}

func NewLLMRSSBriefingBuilder(client llm.Completer, cfg Config) RSSBriefingBuilder {
	return bridgeorchestration.NewLLMRSSBriefingBuilder(client, cfg)
}

func NewRSSInboxStore(path string) (*RSSInboxStore, error) {
	return bridgeorchestration.NewRSSInboxStore(path)
}

func NewRSSBriefingStore(path string) (*RSSBriefingStore, error) {
	return bridgeorchestration.NewRSSBriefingStore(path)
}

func NewRSSReportStore(path string) (*RSSReportStore, error) {
	return bridgeorchestration.NewRSSReportStore(path)
}
