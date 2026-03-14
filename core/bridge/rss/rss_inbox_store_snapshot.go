package rss

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type rssInboxSeenIndex struct {
	keys map[string]struct{}
	ids  map[string]struct{}
}

func (s *RSSInboxStore) saveItemsLocked(snapshot rssInboxSnapshot, items []RSSInboxItem) ([]RSSInboxItem, error) {
	index := newRSSInboxSeenIndex(snapshot)
	saved := make([]RSSInboxItem, 0, len(items))
	now := s.currentTime().UTC()
	for _, item := range items {
		normalized, err := normalizeRSSInboxItem(item, now)
		if err != nil {
			return nil, err
		}
		if index.contains(normalized) {
			continue
		}
		snapshot.Items = append(snapshot.Items, normalized)
		snapshot.SeenKeys = append(snapshot.SeenKeys, normalized.DedupeKey)
		saved = append(saved, normalized)
		index.add(normalized)
	}
	if len(saved) == 0 {
		return nil, nil
	}
	if err := s.saveLocked(snapshot); err != nil {
		return nil, err
	}
	return saved, nil
}

func getRSSInboxItem(items []RSSInboxItem, id string) (RSSInboxItem, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return RSSInboxItem{}, fmt.Errorf("rss inbox item id is required")
	}
	for _, item := range items {
		if item.ID == trimmedID {
			return cloneRSSInboxItem(item), nil
		}
	}
	return RSSInboxItem{}, fmt.Errorf("%w: %s", ErrRSSInboxItemNotFound, trimmedID)
}

func newRSSInboxSeenIndex(snapshot rssInboxSnapshot) rssInboxSeenIndex {
	index := rssInboxSeenIndex{
		keys: buildRSSInboxSeenKeys(snapshot),
		ids:  make(map[string]struct{}, len(snapshot.Items)),
	}
	for _, item := range snapshot.Items {
		if id := strings.TrimSpace(item.ID); id != "" {
			index.ids[id] = struct{}{}
		}
	}
	return index
}

func (i rssInboxSeenIndex) contains(item RSSInboxItem) bool {
	if _, exists := i.keys[item.DedupeKey]; exists {
		return true
	}
	_, exists := i.ids[item.ID]
	return exists
}

func (i *rssInboxSeenIndex) add(item RSSInboxItem) {
	i.keys[item.DedupeKey] = struct{}{}
	i.ids[item.ID] = struct{}{}
}

func buildRSSInboxSeenKeys(snapshot rssInboxSnapshot) map[string]struct{} {
	seen := make(map[string]struct{}, len(snapshot.SeenKeys)+len(snapshot.Items))
	for _, key := range snapshot.SeenKeys {
		if trimmed := strings.TrimSpace(key); trimmed != "" {
			seen[trimmed] = struct{}{}
		}
	}
	for _, item := range snapshot.Items {
		if key := strings.TrimSpace(item.DedupeKey); key != "" {
			seen[key] = struct{}{}
		}
	}
	return seen
}

func markRSSInboxSnapshotSeenKeys(snapshot *rssInboxSnapshot, keys []string) bool {
	seen := buildRSSInboxSeenKeys(*snapshot)
	changed := false
	for _, raw := range keys {
		key := strings.TrimSpace(raw)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		snapshot.SeenKeys = append(snapshot.SeenKeys, key)
		seen[key] = struct{}{}
		changed = true
	}
	return changed
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
