package memorystore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (s *Store) CreateLearned(ctx context.Context, input LearnedMemoryInput, supersedesIDs []string) (MemoryEntry, error) {
	if s == nil || s.db == nil {
		return MemoryEntry{}, errors.New("memory store is not configured")
	}
	entry, err := normalizeLearnedInput(input, s.currentTime())
	if err != nil {
		return MemoryEntry{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return MemoryEntry{}, fmt.Errorf("begin learned memory tx: %w", err)
	}
	if err := supersedeLearnedInTx(ctx, tx, supersedesIDs, entry.ID, entry.UpdatedAt); err != nil {
		_ = tx.Rollback()
		return MemoryEntry{}, err
	}
	metadataValue, err := encodeMetadata(entry.Metadata)
	if err != nil {
		_ = tx.Rollback()
		return MemoryEntry{}, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO learned_memories (
		id, scope_type, scope_id, source_kind, memory_type, memory_key, content, summary,
		metadata_json, confidence, status, superseded_by, created_at, updated_at, last_used_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, ?, NULL)`,
		entry.ID,
		entry.ScopeType,
		entry.ScopeID,
		entry.SourceKind,
		entry.MemoryType,
		nullIfEmpty(entry.MemoryKey),
		entry.Content,
		entry.Summary,
		metadataValue,
		entry.Confidence,
		entry.Status,
		formatTime(entry.CreatedAt),
		formatTime(entry.UpdatedAt),
	)
	if err != nil {
		_ = tx.Rollback()
		return MemoryEntry{}, fmt.Errorf("insert learned memory: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return MemoryEntry{}, fmt.Errorf("commit learned memory tx: %w", err)
	}
	return entry, nil
}

func (s *Store) RefreshLearned(ctx context.Context, id string, confidence float64) error {
	if s == nil || s.db == nil {
		return errors.New("memory store is not configured")
	}
	now := s.currentTime()
	_, err := s.db.ExecContext(ctx, `UPDATE learned_memories
		SET updated_at = ?, confidence = CASE WHEN confidence > ? THEN confidence ELSE ? END
		WHERE id = ?`,
		formatTime(now),
		confidence,
		confidence,
		strings.TrimSpace(id),
	)
	if err != nil {
		return fmt.Errorf("refresh learned memory: %w", err)
	}
	return nil
}

func (s *Store) TouchLearned(ctx context.Context, ids []string) error {
	return s.touchRows(ctx, "learned_memories", "id", ids)
}

func normalizeLearnedInput(input LearnedMemoryInput, now time.Time) (MemoryEntry, error) {
	id, err := newMemoryID("mem")
	if err != nil {
		return MemoryEntry{}, err
	}
	scopeType := normalizeScopeType(input.ScopeType)
	if scopeType == "" {
		scopeType = ScopeTypeUser
	}
	return MemoryEntry{
		ID:         id,
		ScopeType:  scopeType,
		ScopeID:    strings.TrimSpace(input.ScopeID),
		SourceKind: SourceKindLearned,
		MemoryType: resolveMemoryType(input.MemoryType),
		MemoryKey:  resolveLearnedMemoryKey(input),
		Content:    strings.TrimSpace(input.Content),
		Summary:    summarizeText(input.Summary, input.Content),
		Metadata:   cloneMetadata(input.Metadata),
		Confidence: normalizeConfidence(input.Confidence),
		Status:     MemoryStatusActive,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func resolveLearnedMemoryKey(input LearnedMemoryInput) string {
	key := normalizeMemoryKey(input.MemoryKey)
	if key != "" {
		return key
	}
	return defaultMemoryKey(input.MemoryType, summarizeText(input.Summary, input.Content))
}

func supersedeLearnedInTx(ctx context.Context, tx txRunner, ids []string, supersededBy string, now time.Time) error {
	trimmed := normalizeIdentifierList(ids)
	if len(trimmed) == 0 {
		return nil
	}
	filter, args := buildInFilter("id", trimmed)
	args = append([]any{MemoryStatusSuperseded, strings.TrimSpace(supersededBy), formatTime(now)}, args...)
	_, err := tx.ExecContext(ctx, `UPDATE learned_memories
		SET status = ?, superseded_by = ?, updated_at = ?
		WHERE `+filter+` AND status = ?`,
		append(args, MemoryStatusActive)...,
	)
	if err != nil {
		return fmt.Errorf("supersede learned memory: %w", err)
	}
	return nil
}

func scanMemoryEntryRows(rows rowScanner) ([]MemoryEntry, error) {
	items := make([]MemoryEntry, 0)
	for rows.Next() {
		entry, err := scanMemoryEntry(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan memory entries: %w", err)
	}
	return items, nil
}
