package memorystore

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

type rowScanner interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

type txRunner interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func (s *Store) CreateLearningEvent(ctx context.Context, input LearningEventInput) (LearningEvent, error) {
	if s == nil || s.db == nil {
		return LearningEvent{}, errors.New("memory store is not configured")
	}
	id, err := newMemoryID("evt")
	if err != nil {
		return LearningEvent{}, err
	}
	now := s.currentTime()
	event := LearningEvent{
		ID:             id,
		SessionID:      strings.TrimSpace(input.SessionID),
		TraceID:        strings.TrimSpace(input.TraceID),
		Status:         strings.TrimSpace(input.Status),
		InputJSON:      strings.TrimSpace(input.InputJSON),
		FilteredJSON:   strings.TrimSpace(input.FilteredJSON),
		CandidatesJSON: strings.TrimSpace(input.CandidatesJSON),
		ResultJSON:     strings.TrimSpace(input.ResultJSON),
		ErrorText:      strings.TrimSpace(input.ErrorText),
		CreatedAt:      now,
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO memory_learning_events (
		id, session_id, trace_id, status, input_json, filtered_json,
		candidates_json, result_json, error_text, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID,
		event.SessionID,
		event.TraceID,
		event.Status,
		nullIfEmpty(event.InputJSON),
		nullIfEmpty(event.FilteredJSON),
		nullIfEmpty(event.CandidatesJSON),
		nullIfEmpty(event.ResultJSON),
		nullIfEmpty(event.ErrorText),
		formatTime(event.CreatedAt),
	)
	if err != nil {
		return LearningEvent{}, fmt.Errorf("insert learning event: %w", err)
	}
	return event, nil
}

func (s *Store) touchRows(ctx context.Context, table string, idColumn string, ids []string) error {
	if s == nil || s.db == nil {
		return errors.New("memory store is not configured")
	}
	trimmed := normalizeIdentifierList(ids)
	if len(trimmed) == 0 {
		return nil
	}
	filter, args := buildInFilter(idColumn, trimmed)
	args = append([]any{formatTime(s.currentTime())}, args...)
	_, err := s.db.ExecContext(ctx, `UPDATE `+table+` SET last_used_at = ? WHERE `+filter, args...)
	if err != nil {
		return fmt.Errorf("touch %s rows: %w", table, err)
	}
	return nil
}

func buildInFilter(column string, values []string) (string, []any) {
	placeholders := make([]string, 0, len(values))
	args := make([]any, 0, len(values))
	for _, value := range values {
		placeholders = append(placeholders, "?")
		args = append(args, value)
	}
	return column + " IN (" + strings.Join(placeholders, ",") + ")", args
}

func buildLikeFilter(columns []string, terms []string) (string, []any) {
	groups := make([]string, 0, len(terms))
	args := make([]any, 0, len(columns)*len(terms))
	for _, term := range terms {
		if term == "" {
			continue
		}
		parts := make([]string, 0, len(columns))
		for _, column := range columns {
			parts = append(parts, "LOWER("+column+") LIKE ?")
			args = append(args, "%"+term+"%")
		}
		groups = append(groups, "("+strings.Join(parts, " OR ")+")")
	}
	if len(groups) == 0 {
		return "1=1", nil
	}
	return "(" + strings.Join(groups, " OR ") + ")", args
}

func normalizeIdentifierList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func newMemoryID(prefix string) (string, error) {
	tag := strings.TrimSpace(prefix)
	if tag == "" {
		return "", fmt.Errorf("memory id prefix is required")
	}
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate memory id: %w", err)
	}
	return tag + "-" + hex.EncodeToString(raw), nil
}
