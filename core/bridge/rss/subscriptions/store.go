package subscriptions

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	feedStoreDirPerm  = 0o700
	feedStoreFilePerm = 0o600
)

type FeedStore struct {
	path string
	now  func() time.Time
	mu   sync.Mutex
}

type feedStoreSnapshot struct {
	Feeds []FeedSubscription `json:"feeds,omitempty"`
}

func NewFeedStore(path string) (*FeedStore, error) {
	resolved, err := resolveFeedStorePath(path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(resolved), feedStoreDirPerm); err != nil {
		return nil, fmt.Errorf("create feed store directory %q: %w", filepath.Dir(resolved), err)
	}
	return &FeedStore{path: resolved, now: time.Now}, nil
}

func (s *FeedStore) Upsert(input FeedUpsertInput) (FeedSubscription, string, error) {
	if s == nil {
		return FeedSubscription{}, "", errors.New("feed store is nil")
	}
	values, err := normalizeFeedUpsertInput(input)
	if err != nil {
		return FeedSubscription{}, "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	feeds, err := s.loadLocked()
	if err != nil {
		return FeedSubscription{}, "", err
	}
	if feed, action, found, err := s.upsertExistingLocked(feeds, values, s.currentTime()); err != nil {
		return FeedSubscription{}, "", err
	} else if found {
		return feed, action, nil
	}

	created := newFeedSubscription(values, s.currentTime())
	feeds = append(feeds, created)
	if err := s.saveLocked(feeds); err != nil {
		return FeedSubscription{}, "", err
	}
	return cloneFeed(created), "created", nil
}

func (s *FeedStore) Update(feedID string, patch FeedUpdatePatch) (FeedSubscription, error) {
	if s == nil {
		return FeedSubscription{}, errors.New("feed store is nil")
	}
	feedID = strings.TrimSpace(feedID)
	if feedID == "" {
		return FeedSubscription{}, fmt.Errorf("feed_id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	feeds, err := s.loadLocked()
	if err != nil {
		return FeedSubscription{}, err
	}
	for index := range feeds {
		if feeds[index].ID != feedID {
			continue
		}
		updated, changed, err := applyFeedUpdate(feeds[index], patch)
		if err != nil {
			return FeedSubscription{}, err
		}
		if changed {
			updated.UpdatedAt = s.currentTime()
			feeds[index] = updated
			if err := s.saveLocked(feeds); err != nil {
				return FeedSubscription{}, err
			}
		}
		return cloneFeed(feeds[index]), nil
	}
	return FeedSubscription{}, fmt.Errorf("%w: %s", ErrFeedNotFound, feedID)
}

func (s *FeedStore) Delete(feedID string) (bool, error) {
	if s == nil {
		return false, errors.New("feed store is nil")
	}
	feedID = strings.TrimSpace(feedID)
	if feedID == "" {
		return false, fmt.Errorf("feed_id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	feeds, err := s.loadLocked()
	if err != nil {
		return false, err
	}
	for index := range feeds {
		if feeds[index].ID != feedID {
			continue
		}
		feeds = append(feeds[:index], feeds[index+1:]...)
		if err := s.saveLocked(feeds); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (s *FeedStore) List(filter FeedListFilter) ([]FeedSubscription, error) {
	if s == nil {
		return nil, errors.New("feed store is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	feeds, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	return filterFeedSubscriptions(feeds, filter)
}

func (s *FeedStore) loadLocked() ([]FeedSubscription, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read feed store %q: %w", s.path, err)
	}
	if strings.TrimSpace(string(raw)) == "" {
		return nil, nil
	}
	var snapshot feedStoreSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return nil, fmt.Errorf("decode feed store %q: %w", s.path, err)
	}
	return normalizeLoadedFeeds(snapshot.Feeds), nil
}

func (s *FeedStore) saveLocked(feeds []FeedSubscription) error {
	snapshot := feedStoreSnapshot{Feeds: cloneFeeds(feeds)}
	sort.Slice(snapshot.Feeds, func(i, j int) bool {
		return snapshot.Feeds[i].URL < snapshot.Feeds[j].URL
	})
	raw, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("encode feed store: %w", err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(s.path, raw, feedStoreFilePerm); err != nil {
		return fmt.Errorf("write feed store %q: %w", s.path, err)
	}
	return nil
}

func (s *FeedStore) currentTime() time.Time {
	if s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}
