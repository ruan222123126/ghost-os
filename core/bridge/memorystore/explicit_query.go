package memorystore

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

func (s *Store) List(ctx context.Context, prefix string, limit int, offset int) ([]Record, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, errors.New("memory store is not configured")
	}
	pattern := strings.TrimSpace(prefix) + "%"
	total, err := s.count(ctx, `SELECT COUNT(1) FROM memories WHERE uri LIKE ?`, pattern)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT
		uri, content, metadata_json, created_at, updated_at, last_used_at
		FROM memories
		WHERE uri LIKE ?
		ORDER BY uri ASC
		LIMIT ? OFFSET ?`,
		pattern,
		limit,
		offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list memories: %w", err)
	}
	defer rows.Close()
	return scanRecordRows(rows, total)
}

func (s *Store) Search(ctx context.Context, query string, limit int, offset int) ([]Record, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, errors.New("memory store is not configured")
	}
	sqlFilter, args := buildExplicitSearchFilter(query)
	countSQL := `SELECT COUNT(1) FROM memories WHERE ` + sqlFilter
	total, err := s.count(ctx, countSQL, args...)
	if err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx, `SELECT
		uri, content, metadata_json, created_at, updated_at, last_used_at
		FROM memories
		WHERE `+sqlFilter+`
		ORDER BY updated_at DESC, uri ASC
		LIMIT ? OFFSET ?`,
		args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("search memories: %w", err)
	}
	defer rows.Close()
	return scanRecordRows(rows, total)
}

func (s *Store) ListAllURIs(ctx context.Context, limit int, offset int) ([]string, int, error) {
	return s.listURIs(ctx, `SELECT uri FROM memories ORDER BY uri ASC LIMIT ? OFFSET ?`, limit, offset)
}

func (s *Store) ListRecentURIs(ctx context.Context, limit int, offset int) ([]string, int, error) {
	return s.listURIs(ctx, `SELECT uri FROM memories ORDER BY updated_at DESC, uri ASC LIMIT ? OFFSET ?`, limit, offset)
}

func (s *Store) TouchExplicitRecords(ctx context.Context, uris []string) error {
	return s.touchRows(ctx, "memories", "uri", uris)
}

func scanRecordRows(rows rowScanner, total int) ([]Record, int, error) {
	items := make([]Record, 0)
	for rows.Next() {
		record, err := scanRecord(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("scan memories: %w", err)
	}
	return items, total, nil
}

func buildExplicitSearchFilter(query string) (string, []any) {
	terms := buildSearchTerms(query)
	if len(terms) == 0 {
		return "1=1", nil
	}
	return buildLikeFilter([]string{"uri", "content", "COALESCE(metadata_json, '')"}, terms)
}

func explicitRecordToMemoryEntry(record Record, defaultUserScopeID string) MemoryEntry {
	scopeType := normalizeScopeType(metadataValue(record.Metadata, "scope_type"))
	scopeID := strings.TrimSpace(metadataValue(record.Metadata, "scope_id"))
	if scopeType == "" {
		scopeType = ScopeTypeUser
	}
	if scopeID == "" {
		scopeID = normalizeDefaultUserScopeID(defaultUserScopeID)
	}
	return MemoryEntry{
		ID:         "explicit:" + strings.TrimSpace(record.URI),
		ScopeType:  scopeType,
		ScopeID:    scopeID,
		SourceKind: SourceKindExplicit,
		MemoryType: resolveMemoryType(metadataValue(record.Metadata, "memory_type")),
		Content:    record.Content,
		Summary:    summarizeText(metadataValue(record.Metadata, "summary"), record.Content),
		Metadata:   cloneMetadata(record.Metadata),
		Confidence: 1,
		Status:     MemoryStatusActive,
		CreatedAt:  record.CreatedAt,
		UpdatedAt:  record.UpdatedAt,
		LastUsedAt: record.LastUsedAt,
	}
}

func (s *Store) listURIs(ctx context.Context, listQuery string, limit int, offset int) ([]string, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, errors.New("memory store is not configured")
	}
	total, err := s.count(ctx, `SELECT COUNT(1) FROM memories`)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, listQuery, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list memory uris: %w", err)
	}
	defer rows.Close()
	items := make([]string, 0)
	for rows.Next() {
		var uri string
		if err := rows.Scan(&uri); err != nil {
			return nil, 0, fmt.Errorf("scan memory uri: %w", err)
		}
		items = append(items, uri)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("scan memory uri rows: %w", err)
	}
	return items, total, nil
}
