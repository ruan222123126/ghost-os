package rss

import rsssubscriptions "ghost-os/bridge/rss/subscriptions"

type rssInboxStores struct {
	feedStore     *rsssubscriptions.FeedStore
	inboxStore    *RSSInboxStore
	briefingStore *RSSBriefingStore
	reportStore   *RSSReportStore
}

func newRSSInboxServiceFromConfig(store *ConfigStore) (*RSSInboxService, error) {
	cfg, err := loadRSSInboxConfig(store)
	if err != nil {
		return nil, err
	}
	stores, err := newRSSInboxStores(cfg)
	if err != nil {
		return nil, err
	}
	classifier := &llmRSSInboxClassifier{store: store, timeout: defaultRSSClassifierTimeout, cfg: cfg}
	service := NewRSSInboxService(
		stores.feedStore,
		stores.inboxStore,
		stores.briefingStore,
		stores.reportStore,
		classifier,
		cfg,
	)
	service.briefingBuilder = &llmRSSBriefingBuilder{store: store, timeout: defaultRSSBriefingTimeout, cfg: cfg}
	return service, nil
}

func loadRSSInboxConfig(store *ConfigStore) (Config, error) {
	if store != nil {
		return store.Config()
	}
	return LoadConfig()
}

func newRSSInboxStores(cfg Config) (rssInboxStores, error) {
	feedStore, err := rsssubscriptions.NewFeedStore(cfg.RSS.FeedsPath)
	if err != nil {
		return rssInboxStores{}, err
	}
	inboxStore, err := NewRSSInboxStore(cfg.RSS.InboxPath)
	if err != nil {
		return rssInboxStores{}, err
	}
	briefingStore, err := NewRSSBriefingStore(cfg.RSS.BriefingsPath)
	if err != nil {
		return rssInboxStores{}, err
	}
	reportStore, err := NewRSSReportStore(cfg.RSS.ReportsPath)
	if err != nil {
		return rssInboxStores{}, err
	}
	return rssInboxStores{
		feedStore:     feedStore,
		inboxStore:    inboxStore,
		briefingStore: briefingStore,
		reportStore:   reportStore,
	}, nil
}
