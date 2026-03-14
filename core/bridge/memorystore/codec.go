package memorystore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func encodeMetadata(metadata map[string]any) (any, error) {
	if metadata == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("encode metadata: %w", err)
	}
	return string(encoded), nil
}

func decodeMetadata(raw sql.NullString) (map[string]any, error) {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return nil, nil
	}
	var metadata map[string]any
	if err := json.Unmarshal([]byte(raw.String), &metadata); err != nil {
		return nil, fmt.Errorf("decode metadata: %w", err)
	}
	return metadata, nil
}

func scanRecord(scanner interface{ Scan(dest ...any) error }) (Record, error) {
	var record Record
	var metadata sql.NullString
	var createdRaw string
	var updatedRaw string
	var lastUsedRaw sql.NullString
	if err := scanner.Scan(
		&record.URI,
		&record.Content,
		&metadata,
		&createdRaw,
		&updatedRaw,
		&lastUsedRaw,
	); err != nil {
		if err == sql.ErrNoRows {
			return Record{}, ErrNotFound
		}
		return Record{}, fmt.Errorf("scan memory: %w", err)
	}
	createdAt, err := parseTime(createdRaw)
	if err != nil {
		return Record{}, err
	}
	updatedAt, err := parseTime(updatedRaw)
	if err != nil {
		return Record{}, err
	}
	lastUsedAt, err := parseNullableTime(lastUsedRaw)
	if err != nil {
		return Record{}, err
	}
	record.Metadata, err = decodeMetadata(metadata)
	if err != nil {
		return Record{}, err
	}
	record.CreatedAt = createdAt
	record.UpdatedAt = updatedAt
	record.LastUsedAt = lastUsedAt
	return record, nil
}

func scanMemoryEntry(scanner interface{ Scan(dest ...any) error }) (MemoryEntry, error) {
	var entry MemoryEntry
	var metadata sql.NullString
	var createdRaw string
	var updatedRaw string
	var lastUsedRaw sql.NullString
	if err := scanner.Scan(
		&entry.ID,
		&entry.ScopeType,
		&entry.ScopeID,
		&entry.SourceKind,
		&entry.MemoryType,
		&entry.Content,
		&entry.Summary,
		&metadata,
		&entry.Confidence,
		&entry.Status,
		&createdRaw,
		&updatedRaw,
		&lastUsedRaw,
	); err != nil {
		if err == sql.ErrNoRows {
			return MemoryEntry{}, ErrNotFound
		}
		return MemoryEntry{}, fmt.Errorf("scan memory entry: %w", err)
	}
	createdAt, err := parseTime(createdRaw)
	if err != nil {
		return MemoryEntry{}, err
	}
	updatedAt, err := parseTime(updatedRaw)
	if err != nil {
		return MemoryEntry{}, err
	}
	lastUsedAt, err := parseNullableTime(lastUsedRaw)
	if err != nil {
		return MemoryEntry{}, err
	}
	entry.Metadata, err = decodeMetadata(metadata)
	if err != nil {
		return MemoryEntry{}, err
	}
	entry.CreatedAt = createdAt
	entry.UpdatedAt = updatedAt
	entry.LastUsedAt = lastUsedAt
	return entry, nil
}

func parseTime(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, fmt.Errorf("memory timestamp is empty")
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err == nil {
		return parsed, nil
	}
	parsed, err = time.Parse(time.RFC3339, value)
	if err == nil {
		return parsed, nil
	}
	return time.Time{}, fmt.Errorf("parse memory time %q: %w", value, err)
}

func parseNullableTime(raw sql.NullString) (time.Time, error) {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return time.Time{}, nil
	}
	return parseTime(raw.String)
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
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
