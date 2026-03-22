package memorystore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (s *Store) UpsertEventEdge(ctx context.Context, input EventEdgeInput) (EventEdge, error) {
	if s == nil || s.db == nil {
		return EventEdge{}, errors.New("memory store is not configured")
	}
	edge := normalizeEventEdgeInput(input, s.currentTime())
	if edge.SessionID == "" {
		return EventEdge{}, fmt.Errorf("event edge session_id is required")
	}
	if edge.FromEventID == "" || edge.ToEventID == "" {
		return EventEdge{}, fmt.Errorf("event edge endpoints are required")
	}
	if edge.EdgeType == "" {
		return EventEdge{}, fmt.Errorf("event edge_type is required")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO event_edges (
		session_id, from_event_id, to_event_id, edge_type, confidence, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(session_id, from_event_id, to_event_id, edge_type)
	DO UPDATE SET confidence = excluded.confidence, updated_at = excluded.updated_at`,
		edge.SessionID,
		edge.FromEventID,
		edge.ToEventID,
		edge.EdgeType,
		edge.Confidence,
		formatTime(edge.CreatedAt),
		formatTime(edge.UpdatedAt),
	)
	if err != nil {
		return EventEdge{}, fmt.Errorf("upsert event edge: %w", err)
	}
	return edge, nil
}

func (s *Store) ListEventEdges(ctx context.Context, sessionID string, eventIDs []string) ([]EventEdge, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("memory store is not configured")
	}
	trimmedSessionID := strings.TrimSpace(sessionID)
	if trimmedSessionID == "" {
		return nil, nil
	}
	trimmedIDs := normalizeIdentifierList(eventIDs)
	if len(trimmedIDs) == 0 {
		rows, err := s.db.QueryContext(ctx, `SELECT
			session_id, from_event_id, to_event_id, edge_type, confidence, created_at, updated_at
			FROM event_edges
			WHERE session_id = ?
			ORDER BY updated_at DESC, from_event_id ASC, to_event_id ASC`,
			trimmedSessionID,
		)
		if err != nil {
			return nil, fmt.Errorf("list event edges: %w", err)
		}
		defer rows.Close()
		return scanEventEdgeRows(rows)
	}
	filter, args := buildInFilter("from_event_id", trimmedIDs)
	toFilter, toArgs := buildInFilter("to_event_id", trimmedIDs)
	queryArgs := make([]any, 0, len(args)+len(toArgs)+1)
	queryArgs = append(queryArgs, trimmedSessionID)
	queryArgs = append(queryArgs, args...)
	queryArgs = append(queryArgs, toArgs...)
	rows, err := s.db.QueryContext(ctx, `SELECT
		session_id, from_event_id, to_event_id, edge_type, confidence, created_at, updated_at
		FROM event_edges
		WHERE session_id = ? AND (`+filter+` OR `+toFilter+`)
		ORDER BY updated_at DESC, from_event_id ASC, to_event_id ASC`,
		queryArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("list filtered event edges: %w", err)
	}
	defer rows.Close()
	return scanEventEdgeRows(rows)
}

func normalizeEventEdgeInput(input EventEdgeInput, nowTime time.Time) EventEdge {
	return EventEdge{
		SessionID:   strings.TrimSpace(input.SessionID),
		FromEventID: strings.TrimSpace(input.FromEventID),
		ToEventID:   strings.TrimSpace(input.ToEventID),
		EdgeType:    normalizeEventEdgeType(input.EdgeType),
		Confidence:  normalizeConfidence(input.Confidence),
		CreatedAt:   nowTime,
		UpdatedAt:   nowTime,
	}
}
