package memorystore

import (
	"context"
	"errors"
	"fmt"
)

func (s *Store) FindActiveLearnedByMemoryKey(
	ctx context.Context,
	scopeType string,
	scopeID string,
	memoryKey string,
) (MemoryEntry, error) {
	if s == nil || s.db == nil {
		return MemoryEntry{}, errors.New("memory store is not configured")
	}
	normalizedKey := normalizeMemoryKey(memoryKey)
	if normalizedKey == "" {
		return MemoryEntry{}, ErrNotFound
	}
	row := s.db.QueryRowContext(ctx, `SELECT
		id, scope_type, scope_id, source_kind, memory_type, memory_key, content, summary,
		metadata_json, confidence, status, created_at, updated_at, last_used_at
		FROM learned_memories
		WHERE scope_type = ? AND scope_id = ? AND memory_key = ? AND status = ?
		ORDER BY updated_at DESC, id ASC
		LIMIT 1`,
		normalizeScopeType(scopeType),
		scopeID,
		normalizedKey,
		MemoryStatusActive,
	)
	entry, err := scanMemoryEntry(row)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return MemoryEntry{}, ErrNotFound
		}
		return MemoryEntry{}, fmt.Errorf("find learned memory by key: %w", err)
	}
	return entry, nil
}
