package session

import (
	"fmt"
	"strings"
)

func (s *Store) ListRunStates(query RunStateQuery) ([]SessionRunState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	statement, args := buildRunStateQuery(query)
	rows, err := s.db.Query(statement, args...)
	if err != nil {
		return nil, fmt.Errorf("list session run states: %w", err)
	}
	defer rows.Close()

	states := make([]SessionRunState, 0, runStateCapacity(query))
	for rows.Next() {
		state, scanErr := scanSessionRunState(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		states = append(states, state)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list session run states: %w", err)
	}
	return states, nil
}

func buildRunStateQuery(query RunStateQuery) (string, []any) {
	base := `SELECT id, updated_at, state_json FROM sessions`
	if len(query.SessionIDs) > 0 {
		placeholders := make([]string, 0, len(query.SessionIDs))
		args := make([]any, 0, len(query.SessionIDs))
		for _, sessionID := range query.SessionIDs {
			placeholders = append(placeholders, "?")
			args = append(args, sessionID)
		}
		return base + ` WHERE id IN (` + strings.Join(placeholders, ",") + `) ORDER BY updated_at DESC`, args
	}
	if query.Limit > 0 {
		return base + ` ORDER BY updated_at DESC LIMIT ?`, []any{query.Limit}
	}
	return base + ` ORDER BY updated_at DESC`, nil
}

func scanSessionRunState(scanner interface{ Scan(...any) error }) (SessionRunState, error) {
	var (
		sessionID string
		updatedAt string
		stateJSON string
	)
	if err := scanner.Scan(&sessionID, &updatedAt, &stateJSON); err != nil {
		return SessionRunState{}, fmt.Errorf("scan session run state: %w", err)
	}

	sessionUpdatedAt, err := parseSessionTime(updatedAt)
	if err != nil {
		return SessionRunState{}, err
	}
	stored, err := decodeSessionState(stateJSON)
	if err != nil {
		return SessionRunState{}, err
	}
	result := SessionRunState{
		SessionID: sessionID,
		Title:     strings.TrimSpace(stored.Title),
		Status:    RunStatusIdle,
		UpdatedAt: sessionUpdatedAt,
	}
	if stored.LastRunState == nil {
		return result, nil
	}
	result.Status = stored.LastRunState.Status
	result.TraceID = strings.TrimSpace(stored.LastRunState.TraceID)
	result.StartedAt = stored.LastRunState.StartedAt
	result.UpdatedAt = stored.LastRunState.UpdatedAt
	result.TerminalAt = stored.LastRunState.TerminalAt
	return result, nil
}

func runStateCapacity(query RunStateQuery) int {
	if len(query.SessionIDs) > 0 {
		return len(query.SessionIDs)
	}
	if query.Limit > 0 {
		return query.Limit
	}
	return 16
}
