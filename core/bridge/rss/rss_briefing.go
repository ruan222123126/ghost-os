package rss

import (
	"context"
	"time"

	"ghost-os/bridge/llm"
)

const (
	defaultRSSBriefingTimeout         = 45 * time.Second
	defaultRSSBriefingGroupLimit      = 10
	maxRSSBriefingGroupLimit          = 20
	defaultRSSBriefingHighlightsLimit = 5
	maxRSSBriefingHighlightsLimit     = 10
)

type RSSBriefingQuery struct {
	FeedID          string
	Tag             string
	Importance      string
	WindowHours     int
	GroupLimit      int
	ItemLimit       int
	ItemsPerGroup   int
	HighlightsLimit int
	TraceID         string
	TaskID          string
}

type RSSBriefingResult struct {
	ID             string                 `json:"id,omitempty"`
	Title          string                 `json:"title"`
	Summary        string                 `json:"summary,omitempty"`
	GeneratedAt    time.Time              `json:"generated_at"`
	SavedAt        time.Time              `json:"saved_at,omitempty"`
	WindowHours    int                    `json:"window_hours"`
	ScannedGroups  int                    `json:"scanned_groups"`
	HighlightCount int                    `json:"highlight_count"`
	TraceID        string                 `json:"trace_id,omitempty"`
	TaskID         string                 `json:"task_id,omitempty"`
	Highlights     []RSSBriefingHighlight `json:"highlights"`
	Report         *RSSReportResult       `json:"report,omitempty"`
	ReportError    string                 `json:"report_error,omitempty"`
}

type RSSBriefingHighlight struct {
	Rank            int      `json:"rank"`
	GroupID         string   `json:"group_id"`
	TopicLabel      string   `json:"topic_label,omitempty"`
	Headline        string   `json:"headline"`
	Summary         string   `json:"summary,omitempty"`
	WhyItMatters    string   `json:"why_it_matters,omitempty"`
	Importance      string   `json:"importance"`
	SourceItemCount int      `json:"source_item_count"`
	SourceFeedCount int      `json:"source_feed_count"`
	Tags            []string `json:"tags,omitempty"`
}

type rssBriefingBuilder interface {
	Build(context.Context, []RSSInboxTopicGroup, RSSBriefingQuery) (rssBriefingDraft, error)
}

type rssBriefingDraft struct {
	Title      string                      `json:"title,omitempty"`
	Summary    string                      `json:"summary,omitempty"`
	Highlights []rssBriefingDraftHighlight `json:"highlights"`
}

type rssBriefingDraftHighlight struct {
	GroupID      string `json:"group_id"`
	Headline     string `json:"headline,omitempty"`
	Summary      string `json:"summary,omitempty"`
	WhyItMatters string `json:"why_it_matters,omitempty"`
	Importance   string `json:"importance,omitempty"`
}

type llmRSSBriefingBuilder struct {
	store   *ConfigStore
	timeout time.Duration
	client  llm.Completer
	cfg     Config
}
