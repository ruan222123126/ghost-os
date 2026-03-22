package memorystore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (s *Store) CreateEventMemory(ctx context.Context, input EventMemoryInput, supersedesIDs []string) (EventMemory, error) {
	if s == nil || s.db == nil {
		return EventMemory{}, errors.New("memory store is not configured")
	}
	entry := normalizeEventMemoryInput(input, s.currentTime())
	if entry.EventID == "" {
		return EventMemory{}, fmt.Errorf("event_id is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return EventMemory{}, fmt.Errorf("begin event memory tx: %w", err)
	}
	if err := supersedeEventMemoriesInTx(ctx, tx, supersedesIDs, entry.ID, entry.UpdatedAt); err != nil {
		_ = tx.Rollback()
		return EventMemory{}, err
	}
	metadataValue, err := encodeMetadata(entry.Metadata)
	if err != nil {
		_ = tx.Rollback()
		return EventMemory{}, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO event_memories (
		id, event_id, memory_type, memory_key, content, summary, metadata_json,
		confidence, status, superseded_by, created_at, updated_at, last_used_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, ?, NULL)`,
		entry.ID,
		entry.EventID,
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
		return EventMemory{}, fmt.Errorf("insert event memory: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return EventMemory{}, fmt.Errorf("commit event memory tx: %w", err)
	}
	return entry, nil
}

func (s *Store) ListEventMemories(ctx context.Context, filter EventMemoryListFilter) ([]EventMemory, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, errors.New("memory store is not configured")
	}
	whereSQL, args := buildEventMemoryFilter(filter)
	total, err := s.count(ctx, `SELECT COUNT(1)
		FROM event_memories em
		JOIN event_nodes en ON en.id = em.event_id
		WHERE `+whereSQL, args...)
	if err != nil {
		return nil, 0, err
	}
	args = append(args, resolveEventMemoryLimit(filter.Limit), filter.Offset)
	rows, err := s.db.QueryContext(ctx, `SELECT
		em.id, em.event_id, en.session_id, em.memory_type, em.memory_key, em.content, em.summary,
		em.metadata_json, em.confidence, em.status, em.created_at, em.updated_at, em.last_used_at
		FROM event_memories em
		JOIN event_nodes en ON en.id = em.event_id
		WHERE `+whereSQL+`
		ORDER BY em.updated_at DESC, em.id ASC
		LIMIT ? OFFSET ?`,
		args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list event memories: %w", err)
	}
	defer rows.Close()
	items, err := scanEventMemoryRows(rows)
	return items, total, err
}

func (s *Store) RefreshEventMemory(ctx context.Context, id string, confidence float64) error {
	if s == nil || s.db == nil {
		return errors.New("memory store is not configured")
	}
	now := s.currentTime()
	_, err := s.db.ExecContext(ctx, `UPDATE event_memories
		SET updated_at = ?, confidence = CASE WHEN confidence > ? THEN confidence ELSE ? END
		WHERE id = ?`,
		formatTime(now),
		confidence,
		confidence,
		strings.TrimSpace(id),
	)
	if err != nil {
		return fmt.Errorf("refresh event memory: %w", err)
	}
	return nil
}

func (s *Store) TouchEventMemories(ctx context.Context, ids []string) error {
	return s.touchRows(ctx, "event_memories", "id", ids)
}

func (s *Store) FindActiveEventMemoryByKey(ctx context.Context, eventID string, memoryKey string) (EventMemory, error) {
	if s == nil || s.db == nil {
		return EventMemory{}, errors.New("memory store is not configured")
	}
	normalizedKey := normalizeMemoryKey(memoryKey)
	if normalizedKey == "" {
		return EventMemory{}, ErrNotFound
	}
	row := s.db.QueryRowContext(ctx, `SELECT
		em.id, em.event_id, en.session_id, em.memory_type, em.memory_key, em.content, em.summary,
		em.metadata_json, em.confidence, em.status, em.created_at, em.updated_at, em.last_used_at
		FROM event_memories em
		JOIN event_nodes en ON en.id = em.event_id
		WHERE em.event_id = ? AND em.memory_key = ? AND em.status = ?
		ORDER BY em.updated_at DESC, em.id ASC
		LIMIT 1`,
		strings.TrimSpace(eventID),
		normalizedKey,
		MemoryStatusActive,
	)
	entry, err := scanEventMemory(row)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return EventMemory{}, ErrNotFound
		}
		return EventMemory{}, fmt.Errorf("find event memory by key: %w", err)
	}
	return entry, nil
}

func normalizeEventMemoryInput(input EventMemoryInput, now time.Time) EventMemory {
	return EventMemory{
		ID:         newMemoryID("evtmem"),
		EventID:    strings.TrimSpace(input.EventID),
		MemoryType: resolveMemoryType(input.MemoryType),
		MemoryKey:  resolveLearnedMemoryKey(LearnedMemoryInput{MemoryType: input.MemoryType, MemoryKey: input.MemoryKey, Summary: input.Summary, Content: input.Content}),
		Content:    strings.TrimSpace(input.Content),
		Summary:    summarizeText(input.Summary, input.Content),
		Metadata:   cloneMetadata(input.Metadata),
		Confidence: normalizeConfidence(input.Confidence),
		Status:     MemoryStatusActive,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func buildEventMemoryFilter(filter EventMemoryListFilter) (string, []any) {
	clauses := []string{"1=1"}
	args := make([]any, 0, 8)
	if eventID := strings.TrimSpace(filter.EventID); eventID != "" {
		clauses = append(clauses, "em.event_id = ?")
		args = append(args, eventID)
	}
	if sessionID := strings.TrimSpace(filter.SessionID); sessionID != "" {
		clauses = append(clauses, "en.session_id = ?")
		args = append(args, sessionID)
	}
	if memoryType := normalizeMemoryType(filter.MemoryType); memoryType != "" {
		clauses = append(clauses, "em.memory_type = ?")
		args = append(args, memoryType)
	}
	if statuses := normalizeStatusList(filter.Statuses); len(statuses) > 0 {
		clause, clauseArgs := buildInFilter("em.status", statuses)
		clauses = append(clauses, clause)
		args = append(args, clauseArgs...)
	}
	if terms := buildSearchTerms(filter.Query); len(terms) > 0 {
		clause, clauseArgs := buildLikeFilter([]string{"em.summary", "em.content"}, terms)
		clauses = append(clauses, clause)
		args = append(args, clauseArgs...)
	}
	return strings.Join(clauses, " AND "), args
}

func supersedeEventMemoriesInTx(
	ctx context.Context,
	tx txRunner,
	ids []string,
	supersededBy string,
	now time.Time,
) error {
	trimmed := normalizeIdentifierList(ids)
	if len(trimmed) == 0 {
		return nil
	}
	filter, args := buildInFilter("id", trimmed)
	args = append([]any{MemoryStatusSuperseded, strings.TrimSpace(supersededBy), formatTime(now)}, args...)
	_, err := tx.ExecContext(ctx, `UPDATE event_memories
		SET status = ?, superseded_by = ?, updated_at = ?
		WHERE `+filter+` AND status = ?`,
		append(args, MemoryStatusActive)...,
	)
	if err != nil {
		return fmt.Errorf("supersede event memories: %w", err)
	}
	return nil
}

func resolveEventMemoryLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	return limit
}
