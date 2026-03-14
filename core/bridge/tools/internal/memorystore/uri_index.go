package memorystore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

func (s *Store) ListAllURIs(ctx context.Context, limit int, offset int) ([]string, int, error) {
	return s.listURIs(ctx, `SELECT uri FROM memories ORDER BY uri ASC LIMIT ? OFFSET ?`, "list memory uris", "scan memory uri", limit, offset)
}

func (s *Store) ListRecentURIs(ctx context.Context, limit int, offset int) ([]string, int, error) {
	return s.listURIs(ctx, `SELECT uri FROM memories ORDER BY updated_at DESC, uri ASC LIMIT ? OFFSET ?`, "list recent uris", "scan recent uri", limit, offset)
}

func normalizeURI(uri string) (string, error) {
	trimmed := strings.TrimSpace(uri)
	if trimmed == "" {
		return "", invalidURIError("uri is required")
	}
	if strings.IndexFunc(trimmed, unicode.IsControl) >= 0 {
		return "", invalidURIError("uri must not contain control characters")
	}
	return trimmed, nil
}

func initSchema(db *sql.DB) error {
	if db == nil {
		return errors.New("memory database is nil")
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS memories (
		uri TEXT PRIMARY KEY,
		content TEXT NOT NULL,
		metadata_json TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`); err != nil {
		return fmt.Errorf("create memories table: %w", err)
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_memories_updated_at ON memories(updated_at)`); err != nil {
		return fmt.Errorf("create memories index: %w", err)
	}
	return nil
}

func (s *Store) listURIs(ctx context.Context, listQuery string, listLabel string, scanLabel string, limit int, offset int) ([]string, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, errors.New("memory store is not configured")
	}
	total, err := s.count(ctx, `SELECT COUNT(1) FROM memories`)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, listQuery, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", listLabel, err)
	}
	defer rows.Close()
	items, err := scanURIRows(rows, scanLabel)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func scanURIRows(rows *sql.Rows, scanLabel string) ([]string, error) {
	items := make([]string, 0)
	for rows.Next() {
		var uri string
		if err := rows.Scan(&uri); err != nil {
			return nil, fmt.Errorf("%s: %w", scanLabel, err)
		}
		items = append(items, uri)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s rows: %w", scanLabel, err)
	}
	return items, nil
}

func invalidURIError(reason string) error {
	return fmt.Errorf("%w: %s", ErrInvalidURI, reason)
}
