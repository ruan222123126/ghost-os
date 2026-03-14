package rss

import "time"

const (
	defaultRSSAggregateWindowHours = 24
	maxRSSAggregateWindowHours     = 24 * 7
	defaultRSSAggregateGroupLimit  = 12
	maxRSSAggregateGroupLimit      = 50
	defaultRSSAggregateItemLimit   = 200
	maxRSSAggregateItemLimit       = 500
	defaultRSSGroupItemsPerGroup   = 5
	maxRSSGroupItemsPerGroup       = 10
)

type RSSInboxGroupQuery struct {
	FeedID        string
	Tag           string
	Importance    string
	WindowHours   int
	Limit         int
	ItemLimit     int
	ItemsPerGroup int
}

type RSSInboxGroupResult struct {
	WindowHours  int                  `json:"window_hours"`
	GeneratedAt  time.Time            `json:"generated_at"`
	ScannedItems int                  `json:"scanned_items"`
	GroupCount   int                  `json:"group_count"`
	Groups       []RSSInboxTopicGroup `json:"groups"`
}

type RSSInboxTopicGroup struct {
	ID                string         `json:"id"`
	TopicLabel        string         `json:"topic_label"`
	Headline          string         `json:"headline"`
	Summary           string         `json:"summary,omitempty"`
	Importance        string         `json:"importance"`
	WindowStart       time.Time      `json:"window_start"`
	WindowEnd         time.Time      `json:"window_end"`
	LatestActivityAt  time.Time      `json:"latest_activity_at"`
	LatestPublishedAt time.Time      `json:"latest_published_at,omitempty"`
	LatestSavedAt     time.Time      `json:"latest_saved_at"`
	ItemCount         int            `json:"item_count"`
	FeedCount         int            `json:"feed_count"`
	Tags              []string       `json:"tags,omitempty"`
	FeedIDs           []string       `json:"feed_ids,omitempty"`
	Items             []RSSInboxItem `json:"items,omitempty"`
}

type rssAggregateGroupState struct {
	items           []RSSInboxItem
	windowStart     time.Time
	windowEnd       time.Time
	latestActivity  time.Time
	latestPublished time.Time
	latestSaved     time.Time
	feedIDs         map[string]struct{}
	tagCounts       map[string]int
	tokenCounts     map[string]int
	headline        string
	summary         string
	importance      string
}
