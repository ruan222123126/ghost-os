package memorystore

import (
	"database/sql"
	"encoding/json"
	"errors"
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
	if err := scanner.Scan(&record.URI, &record.Content, &metadata, &createdRaw, &updatedRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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
	metadataValue, err := decodeMetadata(metadata)
	if err != nil {
		return Record{}, err
	}
	record.CreatedAt = createdAt
	record.UpdatedAt = updatedAt
	record.Metadata = metadataValue
	return record, nil
}

func scanRecords(rows *sql.Rows) ([]Record, error) {
	items := make([]Record, 0)
	for rows.Next() {
		record, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan memories: %w", err)
	}
	return items, nil
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

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
