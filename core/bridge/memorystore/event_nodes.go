package memorystore

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const defaultEventNodeListLimit = 12

func (s *Store) CreateEventNode(ctx context.Context, input EventNodeInput) (EventNode, error) {
	if s == nil || s.db == nil {
		return EventNode{}, errors.New("memory store is not configured")
	}
	now := s.currentTime()
	node := EventNode{
		ID:              newMemoryID("event"),
		SessionID:       strings.TrimSpace(input.SessionID),
		Title:           strings.TrimSpace(input.Title),
		Summary:         summarizeText(input.Summary, input.Title),
		Status:          resolveEventStatus(input.Status),
		CreatedAt:       now,
		UpdatedAt:       now,
		LastActivatedAt: now,
	}
	if node.SessionID == "" {
		return EventNode{}, fmt.Errorf("event session_id is required")
	}
	if node.Title == "" {
		return EventNode{}, fmt.Errorf("event title is required")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO event_nodes (
		id, session_id, title, summary, status, created_at, updated_at, last_activated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		node.ID,
		node.SessionID,
		node.Title,
		node.Summary,
		node.Status,
		formatTime(node.CreatedAt),
		formatTime(node.UpdatedAt),
		formatTime(node.LastActivatedAt),
	)
	if err != nil {
		return EventNode{}, fmt.Errorf("insert event node: %w", err)
	}
	return node, nil
}

func (s *Store) GetEventNode(ctx context.Context, id string) (EventNode, error) {
	if s == nil || s.db == nil {
		return EventNode{}, errors.New("memory store is not configured")
	}
	row := s.db.QueryRowContext(ctx, `SELECT
		id, session_id, title, summary, status, created_at, updated_at, last_activated_at
		FROM event_nodes
		WHERE id = ?`,
		strings.TrimSpace(id),
	)
	return scanEventNode(row)
}

func (s *Store) ListEventNodes(ctx context.Context, filter EventNodeFilter) ([]EventNode, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("memory store is not configured")
	}
	whereSQL, args := buildEventNodeFilter(filter)
	args = append(args, resolveEventNodeLimit(filter.Limit))
	rows, err := s.db.QueryContext(ctx, `SELECT
		id, session_id, title, summary, status, created_at, updated_at, last_activated_at
		FROM event_nodes
		WHERE `+whereSQL+`
		ORDER BY COALESCE(last_activated_at, updated_at) DESC, updated_at DESC, id ASC
		LIMIT ?`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("list event nodes: %w", err)
	}
	defer rows.Close()
	return scanEventNodeRows(rows)
}

func (s *Store) TouchEventNodes(ctx context.Context, ids []string) error {
	if s == nil || s.db == nil {
		return errors.New("memory store is not configured")
	}
	trimmed := normalizeIdentifierList(ids)
	if len(trimmed) == 0 {
		return nil
	}
	filter, args := buildInFilter("id", trimmed)
	now := formatTime(s.currentTime())
	args = append([]any{now, now, EventStatusActive}, args...)
	_, err := s.db.ExecContext(ctx, `UPDATE event_nodes
		SET last_activated_at = ?, updated_at = ?, status = ?
		WHERE `+filter,
		args...,
	)
	if err != nil {
		return fmt.Errorf("touch event nodes: %w", err)
	}
	return nil
}

func buildEventNodeFilter(filter EventNodeFilter) (string, []any) {
	clauses := []string{"1=1"}
	args := make([]any, 0, 6)
	if sessionID := strings.TrimSpace(filter.SessionID); sessionID != "" {
		clauses = append(clauses, "session_id = ?")
		args = append(args, sessionID)
	}
	if statuses := normalizeEventStatusList(filter.Statuses); len(statuses) > 0 {
		clause, clauseArgs := buildInFilter("status", statuses)
		clauses = append(clauses, clause)
		args = append(args, clauseArgs...)
	}
	if terms := buildSearchTerms(filter.Query); len(terms) > 0 {
		clause, clauseArgs := buildLikeFilter([]string{"title", "summary"}, terms)
		clauses = append(clauses, clause)
		args = append(args, clauseArgs...)
	}
	return strings.Join(clauses, " AND "), args
}

func resolveEventNodeLimit(limit int) int {
	if limit <= 0 {
		return defaultEventNodeListLimit
	}
	return limit
}
