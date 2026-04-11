package orchestration

import (
	bridgeconfig "ghost-os/bridge/config"
	bridgerss "ghost-os/bridge/rss"
	rsssubscriptions "ghost-os/bridge/rss/subscriptions"
)

func newRSSInboxServiceFromConfig(store bridgeconfig.Store) (*bridgerss.RSSInboxService, error) {
	service, err := bridgerss.NewRSSInboxServiceFromConfig(store)
	if err != nil {
		return nil, err
	}
	service.SetReportBuilder(newRuntimeRSSReportBuilder(store))
	return service, nil
}

func NewRSSInboxService(
	feedStore *rsssubscriptions.FeedStore,
	inboxStore *bridgerss.RSSInboxStore,
	briefingStore *bridgerss.RSSBriefingStore,
	reportStore *bridgerss.RSSReportStore,
	classifier bridgerss.RSSInboxClassifier,
	cfg bridgeconfig.Config,
) *bridgerss.RSSInboxService {
	service := bridgerss.NewRSSInboxService(feedStore, inboxStore, briefingStore, reportStore, classifier, cfg)
	service.SetReportBuilder(newRuntimeRSSReportBuilder(nil))
	return service
}
