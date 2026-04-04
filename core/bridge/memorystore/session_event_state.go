package memorystore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

func (s *Store) LoadSessionEventState(ctx context.Context, sessionID string) (SessionEventState, error) {
	if s == nil || s.db == nil {
		return SessionEventState{}, errors.New("memory store is not configured")
	}
	row := s.db.QueryRowContext(ctx, `SELECT
		session_id, primary_event_id, active_event_ids_json, planner_snapshot_json, updated_at
		FROM session_event_state
		WHERE session_id = ?`,
		strings.TrimSpace(sessionID),
	)
	return scanSessionEventState(row)
}

func (s *Store) SaveSessionEventState(ctx context.Context, state SessionEventState) error {
	if s == nil || s.db == nil {
		return errors.New("memory store is not configured")
	}
	sessionID := strings.TrimSpace(state.SessionID)
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	now := s.currentTime()
	activeIDs := normalizeIdentifierList(state.ActiveEventIDs)
	snapshot, err := encodeMetadata(state.PlannerSnapshot)
	if err != nil {
		return err
	}
	activeIDsJSON, err := marshalString(activeIDs)
	if err != nil {
		return fmt.Errorf("encode active event ids: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO session_event_state (
		session_id, primary_event_id, active_event_ids_json, planner_snapshot_json, updated_at
	) VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(session_id)
		DO UPDATE SET
			primary_event_id = excluded.primary_event_id,
			active_event_ids_json = excluded.active_event_ids_json,
			planner_snapshot_json = excluded.planner_snapshot_json,
			updated_at = excluded.updated_at`,
		sessionID,
		nullIfEmpty(state.PrimaryEventID),
		activeIDsJSON,
		snapshot,
		formatTime(now),
	)
	if err != nil {
		return fmt.Errorf("save session event state: %w", err)
	}
	return nil
}

func scanSessionEventState(scanner interface{ Scan(dest ...any) error }) (SessionEventState, error) {
	var (
		state             SessionEventState
		primaryEventID    sql.NullString
		activeEventIDsRaw string
		plannerSnapshot   sql.NullString
		updatedRaw        string
	)
	if err := scanner.Scan(
		&state.SessionID,
		&primaryEventID,
		&activeEventIDsRaw,
		&plannerSnapshot,
		&updatedRaw,
	); err != nil {
		if err == sql.ErrNoRows {
			return SessionEventState{}, ErrNotFound
		}
		return SessionEventState{}, fmt.Errorf("scan session event state: %w", err)
	}
	state.PrimaryEventID = strings.TrimSpace(primaryEventID.String)
	if err := unmarshalStringSlice(activeEventIDsRaw, &state.ActiveEventIDs); err != nil {
		return SessionEventState{}, err
	}
	decodedSnapshot, err := decodeJSONObject(plannerSnapshot)
	if err != nil {
		return SessionEventState{}, err
	}
	updatedAt, err := parseTime(updatedRaw)
	if err != nil {
		return SessionEventState{}, err
	}
	state.PlannerSnapshot = decodedSnapshot
	state.UpdatedAt = updatedAt
	return state, nil
}
