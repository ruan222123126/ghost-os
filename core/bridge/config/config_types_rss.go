package config

import "time"

type RSSConfig struct {
	FeedsPath           string
	InboxPath           string
	BriefingsPath       string
	ReportsPath         string
	PollEnabled         bool
	PollInterval        time.Duration
	PollMaxItemsPerFeed int
	AIBatchSize         int
	BriefingEnabled     bool
	BriefingInterval    time.Duration
}
