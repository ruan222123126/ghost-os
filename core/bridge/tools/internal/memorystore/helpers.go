package memorystore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

func (s *Store) count(ctx context.Context, query string, args ...any) (int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count memories: %w", err)
	}
	return total, nil
}

func (s *Store) exists(ctx context.Context, uri string) (bool, error) {
	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM memories WHERE uri = ?`, uri).Scan(&exists)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, fmt.Errorf("check memory exists: %w", err)
}

func (s *Store) currentTime() time.Time {
	if s == nil || s.now == nil {
		return time.Now().UTC()
	}
	return s.now().UTC()
}

func resolveUpdatedMetadata(existing map[string]any, next map[string]any) map[string]any {
	if next != nil {
		return cloneMetadata(next)
	}
	return cloneMetadata(existing)
}

func cloneMetadata(source map[string]any) map[string]any {
	if source == nil {
		return nil
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}
