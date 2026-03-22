package memorystore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

func scanEventNode(scanner interface{ Scan(dest ...any) error }) (EventNode, error) {
	var (
		node             EventNode
		createdRaw       string
		updatedRaw       string
		lastActivatedRaw sql.NullString
	)
	if err := scanner.Scan(
		&node.ID,
		&node.SessionID,
		&node.Title,
		&node.Summary,
		&node.Status,
		&createdRaw,
		&updatedRaw,
		&lastActivatedRaw,
	); err != nil {
		if err == sql.ErrNoRows {
			return EventNode{}, ErrNotFound
		}
		return EventNode{}, fmt.Errorf("scan event node: %w", err)
	}
	createdAt, err := parseTime(createdRaw)
	if err != nil {
		return EventNode{}, err
	}
	updatedAt, err := parseTime(updatedRaw)
	if err != nil {
		return EventNode{}, err
	}
	lastActivatedAt, err := parseNullableTime(lastActivatedRaw)
	if err != nil {
		return EventNode{}, err
	}
	node.CreatedAt = createdAt
	node.UpdatedAt = updatedAt
	node.LastActivatedAt = lastActivatedAt
	return node, nil
}

func scanEventNodeRows(rows rowScanner) ([]EventNode, error) {
	items := make([]EventNode, 0)
	for rows.Next() {
		item, err := scanEventNode(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan event node rows: %w", err)
	}
	return items, nil
}

func scanEventEdge(scanner interface{ Scan(dest ...any) error }) (EventEdge, error) {
	var (
		edge       EventEdge
		createdRaw string
		updatedRaw string
	)
	if err := scanner.Scan(
		&edge.SessionID,
		&edge.FromEventID,
		&edge.ToEventID,
		&edge.EdgeType,
		&edge.Confidence,
		&createdRaw,
		&updatedRaw,
	); err != nil {
		if err == sql.ErrNoRows {
			return EventEdge{}, ErrNotFound
		}
		return EventEdge{}, fmt.Errorf("scan event edge: %w", err)
	}
	createdAt, err := parseTime(createdRaw)
	if err != nil {
		return EventEdge{}, err
	}
	updatedAt, err := parseTime(updatedRaw)
	if err != nil {
		return EventEdge{}, err
	}
	edge.CreatedAt = createdAt
	edge.UpdatedAt = updatedAt
	return edge, nil
}

func scanEventEdgeRows(rows rowScanner) ([]EventEdge, error) {
	items := make([]EventEdge, 0)
	for rows.Next() {
		item, err := scanEventEdge(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan event edge rows: %w", err)
	}
	return items, nil
}

func scanEventMemory(scanner interface{ Scan(dest ...any) error }) (EventMemory, error) {
	var (
		entry       EventMemory
		metadata    sql.NullString
		createdRaw  string
		updatedRaw  string
		lastUsedRaw sql.NullString
	)
	if err := scanner.Scan(
		&entry.ID,
		&entry.EventID,
		&entry.SessionID,
		&entry.MemoryType,
		&entry.MemoryKey,
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
			return EventMemory{}, ErrNotFound
		}
		return EventMemory{}, fmt.Errorf("scan event memory: %w", err)
	}
	createdAt, err := parseTime(createdRaw)
	if err != nil {
		return EventMemory{}, err
	}
	updatedAt, err := parseTime(updatedRaw)
	if err != nil {
		return EventMemory{}, err
	}
	lastUsedAt, err := parseNullableTime(lastUsedRaw)
	if err != nil {
		return EventMemory{}, err
	}
	entry.Metadata, err = decodeMetadata(metadata)
	if err != nil {
		return EventMemory{}, err
	}
	entry.CreatedAt = createdAt
	entry.UpdatedAt = updatedAt
	entry.LastUsedAt = lastUsedAt
	return entry, nil
}

func scanEventMemoryRows(rows rowScanner) ([]EventMemory, error) {
	items := make([]EventMemory, 0)
	for rows.Next() {
		item, err := scanEventMemory(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan event memory rows: %w", err)
	}
	return items, nil
}

func decodeJSONObject(raw sql.NullString) (map[string]any, error) {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return nil, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(raw.String), &decoded); err != nil {
		return nil, fmt.Errorf("decode json object: %w", err)
	}
	return decoded, nil
}
