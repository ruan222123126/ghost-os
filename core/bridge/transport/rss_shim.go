package transport

import (
	bridgeconfig "ghost-os/bridge/config"
	bridgeorchestration "ghost-os/bridge/orchestration"
	bridgerss "ghost-os/bridge/rss"
	rsssubscriptions "ghost-os/bridge/rss/subscriptions"
)

type RSSInboxService = bridgerss.RSSInboxService
type RSSInboxFetcher = bridgerss.RSSInboxFetcher
type RSSInboxClassifier = bridgerss.RSSInboxClassifier
type RSSInboxClassification = bridgerss.RSSInboxClassification
type RSSInboxCandidate = bridgerss.RSSInboxCandidate
type RSSInboxStore = bridgerss.RSSInboxStore
type RSSBriefingStore = bridgerss.RSSBriefingStore
type RSSReportStore = bridgerss.RSSReportStore
type RSSBriefingBuilder = bridgerss.RSSBriefingBuilder
type rssFeedStore = rsssubscriptions.FeedStore
type rssBriefingCompleter = bridgerss.RSSBriefingCompleter

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
