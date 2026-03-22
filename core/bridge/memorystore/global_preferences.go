package memorystore

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

func (s *Store) FindGlobalPreference(ctx context.Context, memoryKey string) (MemoryEntry, error) {
	if s == nil || s.db == nil {
		return MemoryEntry{}, errors.New("memory store is not configured")
	}
	normalizedKey := normalizeMemoryKey(memoryKey)
	if normalizedKey == "" {
		return MemoryEntry{}, ErrNotFound
	}
	entry, err := s.findExplicitGlobalPreference(ctx, normalizedKey)
	if err == nil {
		return entry, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return MemoryEntry{}, err
	}
	entry, err = s.FindActiveLearnedByMemoryKey(ctx, ScopeTypeUser, s.defaultUserScopeID, normalizedKey)
	if err != nil {
		return MemoryEntry{}, err
	}
	return entry, nil
}

func (s *Store) ListGlobalPreferences(ctx context.Context, memoryKeys []string) ([]MemoryEntry, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("memory store is not configured")
	}
	keys := normalizeIdentifierList(memoryKeys)
	items := make([]MemoryEntry, 0, len(keys))
	for _, key := range keys {
		item, err := s.FindGlobalPreference(ctx, key)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) findExplicitGlobalPreference(ctx context.Context, memoryKey string) (MemoryEntry, error) {
	row := s.db.QueryRowContext(ctx, `SELECT
		uri, content, metadata_json, created_at, updated_at, last_used_at
		FROM memories
		WHERE COALESCE(json_extract(metadata_json, '$.memory_key'), '') = ?
			AND COALESCE(json_extract(metadata_json, '$.scope_type'), ?) = ?
			AND COALESCE(json_extract(metadata_json, '$.scope_id'), ?) = ?
		ORDER BY updated_at DESC, uri ASC
		LIMIT 1`,
		strings.TrimSpace(memoryKey),
		ScopeTypeUser,
		ScopeTypeUser,
		s.defaultUserScopeID,
		s.defaultUserScopeID,
	)
	record, err := scanRecord(row)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return MemoryEntry{}, ErrNotFound
		}
		return MemoryEntry{}, fmt.Errorf("find explicit global preference: %w", err)
	}
	return explicitRecordToMemoryEntry(record, s.defaultUserScopeID), nil
}
