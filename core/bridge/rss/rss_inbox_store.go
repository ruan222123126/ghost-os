package rss

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var ErrRSSInboxItemNotFound = errors.New("rss inbox item not found")

const defaultRSSInboxLimit = 100

type RSSInboxItem struct {
	ID           string    `json:"id"`
	FeedID       string    `json:"feed_id"`
	SourceFeedID string    `json:"source_feed_id"`
	FeedURL      string    `json:"feed_url"`
	SourceTitle  string    `json:"source_title,omitempty"`
	ItemTitle    string    `json:"item_title,omitempty"`
	ItemLink     string    `json:"item_link,omitempty"`
	PublishedAt  time.Time `json:"published_at,omitempty"`
	RawSummary   string    `json:"raw_summary,omitempty"`
	AISummary    string    `json:"ai_summary,omitempty"`
	Tags         []string  `json:"tags,omitempty"`
	Importance   string    `json:"importance"`
	Reason       string    `json:"reason,omitempty"`
	ContentHash  string    `json:"content_hash"`
	DedupeKey    string    `json:"dedupe_key"`
	TraceID      string    `json:"trace_id,omitempty"`
	SavedAt      time.Time `json:"saved_at"`
}

type RSSInboxListFilter struct {
	FeedID          string
	Tag             string
	Importance      string
	SavedAfter      time.Time
	SavedBefore     time.Time
	PublishedAfter  time.Time
	PublishedBefore time.Time
	Limit           int
}

type RSSInboxStore struct {
	path string
	now  func() time.Time
	mu   sync.Mutex
}

type rssInboxSnapshot struct {
	Items    []RSSInboxItem `json:"items,omitempty"`
	SeenKeys []string       `json:"seen_keys,omitempty"`
}

func NewRSSInboxStore(path string) (*RSSInboxStore, error) {
	resolved, err := resolveRSSInboxPath(path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0o700); err != nil {
		return nil, fmt.Errorf("create rss inbox directory %q: %w", filepath.Dir(resolved), err)
	}
	return &RSSInboxStore{path: resolved, now: time.Now}, nil
}

func (s *RSSInboxStore) SaveItems(items []RSSInboxItem) ([]RSSInboxItem, error) {
	if s == nil {
		return nil, errors.New("rss inbox store is nil")
	}
	if len(items) == 0 {
		return nil, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	return s.saveItemsLocked(snapshot, items)
}

func (s *RSSInboxStore) List(filter RSSInboxListFilter) ([]RSSInboxItem, error) {
	if s == nil {
		return nil, errors.New("rss inbox store is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	return listRSSInboxItems(snapshot.Items, newRSSInboxListQuery(filter)), nil
}

func (s *RSSInboxStore) Get(id string) (RSSInboxItem, error) {
	if s == nil {
		return RSSInboxItem{}, errors.New("rss inbox store is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot, err := s.loadLocked()
	if err != nil {
		return RSSInboxItem{}, err
	}
	return getRSSInboxItem(snapshot.Items, id)
}

func (s *RSSInboxStore) KnownDedupeKeys() (map[string]struct{}, error) {
	if s == nil {
		return nil, errors.New("rss inbox store is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	return buildRSSInboxSeenKeys(snapshot), nil
}

func (s *RSSInboxStore) MarkSeen(keys []string) error {
	if s == nil {
		return errors.New("rss inbox store is nil")
	}
	if len(keys) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot, err := s.loadLocked()
	if err != nil {
		return err
	}
	if !markRSSInboxSnapshotSeenKeys(&snapshot, keys) {
		return nil
	}
	return s.saveLocked(snapshot)
}

func (s *RSSInboxStore) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}
