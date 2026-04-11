package transport

import (
	bridgeconfig "ghost-os/bridge/config"
	bridgeorchestration "ghost-os/bridge/orchestration"
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
type rssFeedStore = bridgeorchestration.RSSFeedStore
type rssBriefingCompleter = bridgeorchestration.RSSBriefingCompleter

func NewRSSInboxService(
	feedStore *rssFeedStore,
	inboxStore *RSSInboxStore,
	briefingStore *RSSBriefingStore,
	reportStore *RSSReportStore,
	classifier RSSInboxClassifier,
	cfg bridgeconfig.Config,
) *RSSInboxService {
	return bridgeorchestration.NewRSSInboxService(feedStore, inboxStore, briefingStore, reportStore, classifier, cfg)
}

func NewLLMRSSBriefingBuilder(client rssBriefingCompleter, cfg bridgeconfig.Config) RSSBriefingBuilder {
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
