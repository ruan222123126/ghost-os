package rss

import (
	"crypto/sha256"
	"encoding/hex"
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
	seenDedupe := make(map[string]struct{}, len(snapshot.Items))
	for _, key := range snapshot.SeenKeys {
		if trimmed := strings.TrimSpace(key); trimmed != "" {
			seenDedupe[trimmed] = struct{}{}
		}
	}
	seenIDs := make(map[string]struct{}, len(snapshot.Items))
	for _, item := range snapshot.Items {
		if key := strings.TrimSpace(item.DedupeKey); key != "" {
			seenDedupe[key] = struct{}{}
		}
		if id := strings.TrimSpace(item.ID); id != "" {
			seenIDs[id] = struct{}{}
		}
	}

	saved := make([]RSSInboxItem, 0, len(items))
	now := s.currentTime().UTC()
	for _, item := range items {
		normalized, err := normalizeRSSInboxItem(item, now)
		if err != nil {
			return nil, err
		}
		if _, exists := seenDedupe[normalized.DedupeKey]; exists {
			continue
		}
		if _, exists := seenIDs[normalized.ID]; exists {
			continue
		}
		snapshot.Items = append(snapshot.Items, normalized)
		snapshot.SeenKeys = append(snapshot.SeenKeys, normalized.DedupeKey)
		saved = append(saved, normalized)
		seenDedupe[normalized.DedupeKey] = struct{}{}
		seenIDs[normalized.ID] = struct{}{}
	}

	if len(saved) == 0 {
		return nil, nil
	}
	if err := s.saveLocked(snapshot); err != nil {
		return nil, err
	}
	return saved, nil
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
	filtered := make([]RSSInboxItem, 0, len(snapshot.Items))
	for _, item := range snapshot.Items {
		if !matchesRSSInboxFilter(item, filter) {
			continue
		}
		filtered = append(filtered, cloneRSSInboxItem(item))
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].SavedAt.Equal(filtered[j].SavedAt) {
			return filtered[i].ID > filtered[j].ID
		}
		return filtered[i].SavedAt.After(filtered[j].SavedAt)
	})
	limit := filter.Limit
	if limit <= 0 {
		limit = defaultRSSInboxLimit
	}
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, nil
}

func (s *RSSInboxStore) Get(id string) (RSSInboxItem, error) {
	if s == nil {
		return RSSInboxItem{}, errors.New("rss inbox store is nil")
	}
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return RSSInboxItem{}, fmt.Errorf("rss inbox item id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot, err := s.loadLocked()
	if err != nil {
		return RSSInboxItem{}, err
	}
	for _, item := range snapshot.Items {
		if item.ID == trimmedID {
			return cloneRSSInboxItem(item), nil
		}
	}
	return RSSInboxItem{}, fmt.Errorf("%w: %s", ErrRSSInboxItemNotFound, trimmedID)
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
	keys := make(map[string]struct{}, len(snapshot.Items))
	for _, key := range snapshot.SeenKeys {
		if trimmed := strings.TrimSpace(key); trimmed != "" {
			keys[trimmed] = struct{}{}
		}
	}
	for _, item := range snapshot.Items {
		if key := strings.TrimSpace(item.DedupeKey); key != "" {
			keys[key] = struct{}{}
		}
	}
	return keys, nil
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
	seen := make(map[string]struct{}, len(snapshot.SeenKeys)+len(snapshot.Items))
	for _, key := range snapshot.SeenKeys {
		if trimmed := strings.TrimSpace(key); trimmed != "" {
			seen[trimmed] = struct{}{}
		}
	}
	for _, item := range snapshot.Items {
		if trimmed := strings.TrimSpace(item.DedupeKey); trimmed != "" {
			seen[trimmed] = struct{}{}
		}
	}
	changed := false
	for _, raw := range keys {
		key := strings.TrimSpace(raw)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		snapshot.SeenKeys = append(snapshot.SeenKeys, key)
		changed = true
	}
	if !changed {
		return nil
	}
	return s.saveLocked(snapshot)
}

func normalizeRSSInboxItem(item RSSInboxItem, fallbackSavedAt time.Time) (RSSInboxItem, error) {
	item.FeedID = strings.TrimSpace(item.FeedID)
	item.SourceFeedID = strings.TrimSpace(item.SourceFeedID)
	item.FeedURL = strings.TrimSpace(item.FeedURL)
	item.SourceTitle = strings.TrimSpace(item.SourceTitle)
	item.ItemTitle = strings.TrimSpace(item.ItemTitle)
	item.ItemLink = strings.TrimSpace(item.ItemLink)
	item.RawSummary = strings.TrimSpace(item.RawSummary)
	item.AISummary = strings.TrimSpace(item.AISummary)
	item.Reason = strings.TrimSpace(item.Reason)
	item.TraceID = strings.TrimSpace(item.TraceID)
	item.ContentHash = strings.TrimSpace(item.ContentHash)
	item.DedupeKey = strings.TrimSpace(item.DedupeKey)
	item.Importance = normalizeRSSInboxImportance(item.Importance)
	item.Tags = normalizeRSSInboxTags(item.Tags)
	if item.FeedID == "" {
		return RSSInboxItem{}, fmt.Errorf("feed_id is required")
	}
	if item.SourceFeedID == "" {
		item.SourceFeedID = item.FeedID
	}
	if item.DedupeKey == "" {
		return RSSInboxItem{}, fmt.Errorf("dedupe_key is required")
	}
	if item.ContentHash == "" {
		return RSSInboxItem{}, fmt.Errorf("content_hash is required")
	}
	if item.SavedAt.IsZero() {
		item.SavedAt = fallbackSavedAt
	}
	item.SavedAt = item.SavedAt.UTC()
	if !item.PublishedAt.IsZero() {
		item.PublishedAt = item.PublishedAt.UTC()
	}
	if strings.TrimSpace(item.ID) == "" {
		item.ID = newRSSInboxItemID(item.DedupeKey, item.ContentHash)
	}
	return item, nil
}

func matchesRSSInboxFilter(item RSSInboxItem, filter RSSInboxListFilter) bool {
	if feedID := strings.TrimSpace(filter.FeedID); feedID != "" && item.FeedID != feedID {
		return false
	}
	if importance := normalizeRSSInboxImportance(filter.Importance); strings.TrimSpace(filter.Importance) != "" && item.Importance != importance {
		return false
	}
	if tag := strings.ToLower(strings.TrimSpace(filter.Tag)); tag != "" && !containsRSSInboxTag(item.Tags, tag) {
		return false
	}
	if !filter.SavedAfter.IsZero() && item.SavedAt.Before(filter.SavedAfter.UTC()) {
		return false
	}
	if !filter.SavedBefore.IsZero() && item.SavedAt.After(filter.SavedBefore.UTC()) {
		return false
	}
	if !filter.PublishedAfter.IsZero() {
		if item.PublishedAt.IsZero() || item.PublishedAt.Before(filter.PublishedAfter.UTC()) {
			return false
		}
	}
	if !filter.PublishedBefore.IsZero() {
		if item.PublishedAt.IsZero() || item.PublishedAt.After(filter.PublishedBefore.UTC()) {
			return false
		}
	}
	return true
}

func containsRSSInboxTag(tags []string, expected string) bool {
	for _, tag := range tags {
		if strings.EqualFold(strings.TrimSpace(tag), expected) {
			return true
		}
	}
	return false
}

func normalizeRSSInboxTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, raw := range tags {
		tag := strings.ToLower(strings.TrimSpace(raw))
		if tag == "" {
			continue
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		normalized = append(normalized, tag)
	}
	sort.Strings(normalized)
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeRSSInboxImportance(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "low", "high":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return "normal"
	}
}

func cloneRSSInboxItem(item RSSInboxItem) RSSInboxItem {
	item.Tags = append([]string(nil), item.Tags...)
	return item
}

func (s *RSSInboxStore) loadLocked() (rssInboxSnapshot, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return rssInboxSnapshot{}, nil
		}
		return rssInboxSnapshot{}, fmt.Errorf("read rss inbox snapshot %q: %w", s.path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return rssInboxSnapshot{}, nil
	}
	var snapshot rssInboxSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return rssInboxSnapshot{}, fmt.Errorf("decode rss inbox snapshot %q: %w", s.path, err)
	}
	return snapshot, nil
}

func (s *RSSInboxStore) saveLocked(snapshot rssInboxSnapshot) error {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("encode rss inbox snapshot %q: %w", s.path, err)
	}
	data = append(data, '\n')
	tempPath := fmt.Sprintf("%s.tmp-%d", s.path, time.Now().UnixNano())
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return fmt.Errorf("write temp rss inbox snapshot %q: %w", tempPath, err)
	}
	if err := os.Rename(tempPath, s.path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("replace rss inbox snapshot %q: %w", s.path, err)
	}
	return nil
}

func (s *RSSInboxStore) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}

func resolveRSSInboxPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		trimmed = defaultRSSInboxPath
	}
	resolved, err := resolveUserPath(trimmed)
	if err != nil {
		return "", fmt.Errorf("resolve rss inbox path: %w", err)
	}
	return resolved, nil
}

func newRSSInboxItemID(dedupeKey string, contentHash string) string {
	hash := sha256.Sum256([]byte(strings.TrimSpace(dedupeKey) + "\n" + strings.TrimSpace(contentHash)))
	return "rss_" + hex.EncodeToString(hash[:12])
}
