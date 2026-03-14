package memorystore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

func (s *Store) Create(ctx context.Context, uri string, content string, metadata map[string]any) (Record, error) {
	if s == nil || s.db == nil {
		return Record{}, errors.New("memory store is not configured")
	}
	normalizedURI, err := normalizeURI(uri)
	if err != nil {
		return Record{}, err
	}
	exists, err := s.explicitExists(ctx, normalizedURI)
	if err != nil {
		return Record{}, err
	}
	if exists {
		return Record{}, ErrAlreadyExists
	}
	createdAt := s.currentTime()
	normalizedMetadata := normalizeExplicitMetadata(metadata, s.defaultUserScopeID)
	metadataValue, err := encodeMetadata(normalizedMetadata)
	if err != nil {
		return Record{}, err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO memories (
		uri, content, metadata_json, created_at, updated_at, last_used_at
	) VALUES (?, ?, ?, ?, ?, NULL)`,
		normalizedURI,
		content,
		metadataValue,
		formatTime(createdAt),
		formatTime(createdAt),
	)
	if err != nil {
		return Record{}, fmt.Errorf("insert memory: %w", err)
	}
	return Record{
		URI:       normalizedURI,
		Content:   content,
		Metadata:  normalizedMetadata,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}, nil
}

func (s *Store) Update(ctx context.Context, uri string, content string, metadata map[string]any) (Record, error) {
	if s == nil || s.db == nil {
		return Record{}, errors.New("memory store is not configured")
	}
	normalizedURI, err := normalizeURI(uri)
	if err != nil {
		return Record{}, err
	}
	existing, err := s.Read(ctx, normalizedURI)
	if err != nil {
		return Record{}, err
	}
	updatedAt := s.currentTime()
	normalizedMetadata := resolveUpdatedExplicitMetadata(existing.Metadata, metadata, s.defaultUserScopeID)
	metadataValue, err := encodeMetadata(normalizedMetadata)
	if err != nil {
		return Record{}, err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE memories
		SET content = ?, metadata_json = ?, updated_at = ?
		WHERE uri = ?`,
		content,
		metadataValue,
		formatTime(updatedAt),
		normalizedURI,
	)
	if err != nil {
		return Record{}, fmt.Errorf("update memory: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Record{}, fmt.Errorf("update memory: %w", err)
	}
	if affected == 0 {
		return Record{}, ErrNotFound
	}
	return s.Read(ctx, normalizedURI)
}

func (s *Store) Delete(ctx context.Context, uri string) (bool, error) {
	if s == nil || s.db == nil {
		return false, errors.New("memory store is not configured")
	}
	normalizedURI, err := normalizeURI(uri)
	if err != nil {
		return false, err
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM memories WHERE uri = ?`, normalizedURI)
	if err != nil {
		return false, fmt.Errorf("delete memory: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("delete memory: %w", err)
	}
	return affected > 0, nil
}

func (s *Store) Read(ctx context.Context, uri string) (Record, error) {
	if s == nil || s.db == nil {
		return Record{}, errors.New("memory store is not configured")
	}
	normalizedURI, err := normalizeURI(uri)
	if err != nil {
		return Record{}, err
	}
	row := s.db.QueryRowContext(ctx, `SELECT
		uri, content, metadata_json, created_at, updated_at, last_used_at
		FROM memories WHERE uri = ?`,
		normalizedURI,
	)
	return scanRecord(row)
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

func invalidURIError(reason string) error {
	return fmt.Errorf("%w: %s", ErrInvalidURI, reason)
}

func (s *Store) explicitExists(ctx context.Context, uri string) (bool, error) {
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

func resolveUpdatedExplicitMetadata(existing map[string]any, next map[string]any, defaultUserScopeID string) map[string]any {
	if next == nil {
		return normalizeExplicitMetadata(existing, defaultUserScopeID)
	}
	return normalizeExplicitMetadata(next, defaultUserScopeID)
}

func normalizeExplicitMetadata(source map[string]any, defaultUserScopeID string) map[string]any {
	metadata := cloneMetadata(source)
	if metadata == nil {
		metadata = make(map[string]any, 3)
	}
	metadata["source_kind"] = SourceKindExplicit
	scopeType := normalizeScopeType(metadataValue(metadata, "scope_type"))
	scopeID := strings.TrimSpace(metadataValue(metadata, "scope_id"))
	if scopeType == "" {
		scopeType = ScopeTypeUser
	}
	if scopeID == "" {
		scopeID = normalizeDefaultUserScopeID(defaultUserScopeID)
	}
	metadata["scope_type"] = scopeType
	metadata["scope_id"] = scopeID
	return metadata
}

func metadataValue(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	value, ok := metadata[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}
