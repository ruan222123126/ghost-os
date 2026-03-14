package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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
	if err := os.MkdirAll(filepath.Dir(resolved), 0o700); err != nil {
		return nil, fmt.Errorf("create feed store directory %q: %w", filepath.Dir(resolved), err)
	}
	return &FeedStore{path: resolved, now: time.Now}, nil
}

func (s *FeedStore) Upsert(input FeedUpsertInput) (FeedSubscription, string, error) {
	if s == nil {
		return FeedSubscription{}, "", errors.New("feed store is nil")
	}
	urlValue, err := normalizeFeedURL(input.URL)
	if err != nil {
		return FeedSubscription{}, "", err
	}
	title := strings.TrimSpace(input.Title)
	probeTitle := strings.TrimSpace(input.ProbeTitle)
	tags := normalizeFeedTags(input.Tags)
	priority := defaultFeedPriority
	prioritySet := false
	if strings.TrimSpace(input.Priority) != "" {
		priority, err = normalizeFeedPriority(input.Priority)
		if err != nil {
			return FeedSubscription{}, "", err
		}
		prioritySet = true
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	feeds, err := s.loadLocked()
	if err != nil {
		return FeedSubscription{}, "", err
	}
	now := s.currentTime()
	for index := range feeds {
		if feeds[index].URL != urlValue {
			continue
		}
		changed := false
		if title != "" && feeds[index].Title != title {
			feeds[index].Title = title
			changed = true
		} else if feeds[index].Title == "" && probeTitle != "" {
			feeds[index].Title = probeTitle
			changed = true
		}
		mergedTags := mergeFeedTags(feeds[index].Tags, tags)
		if !equalStringSlices(feeds[index].Tags, mergedTags) {
			feeds[index].Tags = mergedTags
			changed = true
		}
		if prioritySet && feeds[index].Priority != priority {
			feeds[index].Priority = priority
			changed = true
		}
		if input.Enabled != nil && feeds[index].Enabled != *input.Enabled {
			feeds[index].Enabled = *input.Enabled
			changed = true
		}
		if changed {
			feeds[index].UpdatedAt = now
			if err := s.saveLocked(feeds); err != nil {
				return FeedSubscription{}, "", err
			}
			return cloneFeed(feeds[index]), "updated_existing", nil
		}
		return cloneFeed(feeds[index]), "exists", nil
	}

	feed := FeedSubscription{
		ID:        newFeedID(urlValue),
		URL:       urlValue,
		Title:     feedFirstNonEmpty(title, probeTitle),
		Tags:      tags,
		Priority:  priority,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if input.Enabled != nil {
		feed.Enabled = *input.Enabled
	}
	feeds = append(feeds, feed)
	if err := s.saveLocked(feeds); err != nil {
		return FeedSubscription{}, "", err
	}
	return cloneFeed(feed), "created", nil
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
		changed := false
		if patch.Title != nil {
			title := strings.TrimSpace(*patch.Title)
			if feeds[index].Title != title {
				feeds[index].Title = title
				changed = true
			}
		}
		if patch.Tags != nil {
			tags := normalizeFeedTags(*patch.Tags)
			if !equalStringSlices(feeds[index].Tags, tags) {
				feeds[index].Tags = tags
				changed = true
			}
		}
		if patch.Priority != nil {
			priority, err := normalizeFeedPriority(*patch.Priority)
			if err != nil {
				return FeedSubscription{}, err
			}
			if feeds[index].Priority != priority {
				feeds[index].Priority = priority
				changed = true
			}
		}
		if patch.Enabled != nil && feeds[index].Enabled != *patch.Enabled {
			feeds[index].Enabled = *patch.Enabled
			changed = true
		}
		if changed {
			feeds[index].UpdatedAt = s.currentTime()
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
	tag := strings.ToLower(strings.TrimSpace(filter.Tag))
	priority := ""
	if strings.TrimSpace(filter.Priority) != "" {
		priority, err = normalizeFeedPriority(filter.Priority)
		if err != nil {
			return nil, err
		}
	}
	result := make([]FeedSubscription, 0, len(feeds))
	for _, feed := range feeds {
		if filter.Enabled != nil && feed.Enabled != *filter.Enabled {
			continue
		}
		if priority != "" && feed.Priority != priority {
			continue
		}
		if tag != "" && !feedHasTag(feed, tag) {
			continue
		}
		result = append(result, cloneFeed(feed))
	}
	sortFeedSubscriptions(result)
	return result, nil
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
	feeds := make([]FeedSubscription, 0, len(snapshot.Feeds))
	for _, feed := range snapshot.Feeds {
		if strings.TrimSpace(feed.ID) == "" || strings.TrimSpace(feed.URL) == "" {
			continue
		}
		feed.URL, err = normalizeFeedURL(feed.URL)
		if err != nil {
			continue
		}
		feed.Title = strings.TrimSpace(feed.Title)
		feed.Tags = normalizeFeedTags(feed.Tags)
		priority, priorityErr := normalizeFeedPriority(feed.Priority)
		if priorityErr != nil {
			priority = defaultFeedPriority
		}
		feed.Priority = priority
		feeds = append(feeds, feed)
	}
	return feeds, nil
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
	if err := os.WriteFile(s.path, raw, 0o600); err != nil {
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

func resolveFeedStorePath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errors.New("feed store path is empty")
	}
	if trimmed == "~" || strings.HasPrefix(trimmed, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user home directory: %w", err)
		}
		if trimmed == "~" {
			return homeDir, nil
		}
		return filepath.Join(homeDir, strings.TrimPrefix(trimmed, "~/")), nil
	}
	return filepath.Clean(trimmed), nil
}

func normalizeFeedURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("url is required")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid url: %w", err)
	}
	parsed.Fragment = ""
	return parsed.String(), nil
}

func normalizeFeedPriority(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "low":
		return "low", nil
	case "", "normal":
		return "normal", nil
	case "high":
		return "high", nil
	default:
		return "", fmt.Errorf("priority must be one of low, normal, or high")
	}
}

func normalizeFeedTags(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, value := range raw {
		tag := strings.TrimSpace(value)
		if tag == "" {
			continue
		}
		key := strings.ToLower(tag)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, tag)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	if len(out) == 0 {
		return nil
	}
	return out
}

func mergeFeedTags(existing, incoming []string) []string {
	if len(incoming) == 0 {
		return normalizeFeedTags(existing)
	}
	merged := append(cloneStringSlice(existing), incoming...)
	return normalizeFeedTags(merged)
}

func sortFeedSubscriptions(feeds []FeedSubscription) {
	sort.SliceStable(feeds, func(i, j int) bool {
		left := feeds[i]
		right := feeds[j]
		if left.Enabled != right.Enabled {
			return left.Enabled
		}
		leftPriority := feedPriorityRank(left.Priority)
		rightPriority := feedPriorityRank(right.Priority)
		if leftPriority != rightPriority {
			return leftPriority > rightPriority
		}
		if !left.UpdatedAt.Equal(right.UpdatedAt) {
			return left.UpdatedAt.After(right.UpdatedAt)
		}
		leftLabel := strings.ToLower(feedFirstNonEmpty(left.Title, left.URL))
		rightLabel := strings.ToLower(feedFirstNonEmpty(right.Title, right.URL))
		if leftLabel != rightLabel {
			return leftLabel < rightLabel
		}
		return left.ID < right.ID
	})
}

func feedPriorityRank(priority string) int {
	switch priority {
	case "high":
		return 3
	case "normal":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func newFeedID(urlValue string) string {
	sum := sha256.Sum256([]byte(urlValue))
	return "feed-" + hex.EncodeToString(sum[:6])
}

func feedHasTag(feed FeedSubscription, wanted string) bool {
	for _, tag := range feed.Tags {
		if strings.EqualFold(strings.TrimSpace(tag), wanted) {
			return true
		}
	}
	return false
}

func cloneFeeds(raw []FeedSubscription) []FeedSubscription {
	if len(raw) == 0 {
		return nil
	}
	out := make([]FeedSubscription, 0, len(raw))
	for _, item := range raw {
		out = append(out, cloneFeed(item))
	}
	return out
}

func cloneFeed(feed FeedSubscription) FeedSubscription {
	feed.Tags = cloneStringSlice(feed.Tags)
	return feed
}

func cloneStringSlice(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	out := make([]string, len(raw))
	copy(out, raw)
	return out
}

func equalStringSlices(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func feedFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
