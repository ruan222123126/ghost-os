package subscriptions

import (
	"errors"
	"time"
)

var ErrFeedNotFound = errors.New("feed not found")

const defaultFeedPriority = "normal"

type FeedSubscription struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Title     string    `json:"title,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	Priority  string    `json:"priority"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type FeedListFilter struct {
	Enabled  *bool
	Tag      string
	Priority string
}

type FeedUpsertInput struct {
	URL        string
	Title      string
	Tags       []string
	Priority   string
	Enabled    *bool
	ProbeTitle string
}

type FeedUpdatePatch struct {
	Title    *string
	Tags     *[]string
	Priority *string
	Enabled  *bool
}
