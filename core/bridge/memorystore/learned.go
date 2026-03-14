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
	entry := normalizeLearnedInput(input, s.currentTime())
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
		id, scope_type, scope_id, source_kind, memory_type, content, summary,
		metadata_json, confidence, status, superseded_by, created_at, updated_at, last_used_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, ?, NULL)`,
		entry.ID,
		entry.ScopeType,
		entry.ScopeID,
		entry.SourceKind,
		entry.MemoryType,
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

func (s *Store) GetLearnedByIDs(ctx context.Context, ids []string) ([]MemoryEntry, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("memory store is not configured")
	}
	trimmed := normalizeIdentifierList(ids)
	if len(trimmed) == 0 {
		return nil, nil
	}
	filter, args := buildInFilter("id", trimmed)
	rows, err := s.db.QueryContext(ctx, `SELECT
		id, scope_type, scope_id, source_kind, memory_type, content, summary,
		metadata_json, confidence, status, created_at, updated_at, last_used_at
		FROM learned_memories
		WHERE `+filter+`
		ORDER BY updated_at DESC, id ASC`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("query learned memories by id: %w", err)
	}
	defer rows.Close()
	return scanMemoryEntryRows(rows)
}

func (s *Store) ListLearned(ctx context.Context, filter LearnedListFilter) ([]MemoryEntry, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, errors.New("memory store is not configured")
	}
	whereSQL, args := buildLearnedFilter(filter)
	total, err := s.count(ctx, `SELECT COUNT(1) FROM learned_memories WHERE `+whereSQL, args...)
	if err != nil {
		return nil, 0, err
	}
	args = append(args, filter.Limit, filter.Offset)
	rows, err := s.db.QueryContext(ctx, `SELECT
		id, scope_type, scope_id, source_kind, memory_type, content, summary,
		metadata_json, confidence, status, created_at, updated_at, last_used_at
		FROM learned_memories
		WHERE `+whereSQL+`
		ORDER BY updated_at DESC, id ASC
		LIMIT ? OFFSET ?`,
		args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list learned memories: %w", err)
	}
	defer rows.Close()
	items, err := scanMemoryEntryRows(rows)
	return items, total, err
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

func normalizeLearnedInput(input LearnedMemoryInput, now time.Time) MemoryEntry {
	scopeType := normalizeScopeType(input.ScopeType)
	if scopeType == "" {
		scopeType = ScopeTypeUser
	}
	return MemoryEntry{
		ID:         newMemoryID("mem"),
		ScopeType:  scopeType,
		ScopeID:    strings.TrimSpace(input.ScopeID),
		SourceKind: SourceKindLearned,
		MemoryType: resolveMemoryType(input.MemoryType),
		Content:    strings.TrimSpace(input.Content),
		Summary:    summarizeText(input.Summary, input.Content),
		Metadata:   cloneMetadata(input.Metadata),
		Confidence: normalizeConfidence(input.Confidence),
		Status:     MemoryStatusActive,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func buildLearnedFilter(filter LearnedListFilter) (string, []any) {
	clauses := []string{"1=1"}
	args := make([]any, 0, 8)
	if scopeType := normalizeScopeType(filter.ScopeType); scopeType != "" {
		clauses = append(clauses, "scope_type = ?")
		args = append(args, scopeType)
	}
	if scopeID := strings.TrimSpace(filter.ScopeID); scopeID != "" {
		clauses = append(clauses, "scope_id = ?")
		args = append(args, scopeID)
	}
	if memoryType := normalizeMemoryType(filter.MemoryType); memoryType != "" {
		clauses = append(clauses, "memory_type = ?")
		args = append(args, memoryType)
	}
	if statuses := normalizeStatusList(filter.Statuses); len(statuses) > 0 {
		clause, clauseArgs := buildInFilter("status", statuses)
		clauses = append(clauses, clause)
		args = append(args, clauseArgs...)
	}
	if terms := buildSearchTerms(filter.Query); len(terms) > 0 {
		clause, clauseArgs := buildLikeFilter([]string{"summary", "content"}, terms)
		clauses = append(clauses, clause)
		args = append(args, clauseArgs...)
	}
	return strings.Join(clauses, " AND "), args
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
