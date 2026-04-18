package config

import "time"

const (
	defaultRSSFeedsPath           = "~/.ghost-os/rss/feeds.json"
	defaultRSSInboxPath           = "~/.ghost-os/rss/inbox.json"
	defaultRSSBriefingsPath       = "~/.ghost-os/rss/briefings.json"
	defaultRSSReportsPath         = "~/.ghost-os/rss/reports/index.json"
	defaultRSSPollInterval        = 15 * time.Minute
	defaultRSSPollMaxItemsPerFeed = 10
	defaultRSSAIBatchSize         = 5
	defaultRSSBriefingInterval    = 30 * time.Minute
)
