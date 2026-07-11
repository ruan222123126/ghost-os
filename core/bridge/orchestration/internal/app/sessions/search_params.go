package sessions

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/session"
)

const RecentSessionRunStateLimit = 30

var (
	ErrInvalidSessionRunStateLimit = errors.New("session run state limit must be between 1 and 30")
	ErrTooManySessionRunStateIDs   = errors.New("session_ids must contain at most 30 entries")
)

func ResolveSessionSearchParams(params api.SessionSearchParams) (string, int, error) {
	if params.Limit != nil && *params.Limit <= 0 {
		return "", 0, fmt.Errorf("%w: limit must be a positive integer", ErrInvalidSessionSearchQuery)
	}

	limit := 0
	if params.Limit != nil {
		limit = *params.Limit
	}
	return strings.TrimSpace(params.Query), limit, nil
}

func (s Service) RunStates(params api.SessionRunStatesGetRequest, traceID string) ([]api.SessionRunState, error) {
	query, err := resolveRunStateQuery(params)
	if err != nil {
		s.log(traceID, ActionRunStatesGet, "error", err)
		return nil, err
	}
	store, err := s.requireSessionStore()
	if err != nil {
		s.log(traceID, ActionRunStatesGet, "error", err)
		return nil, err
	}
	hiddenSessionIDs, err := s.hiddenSessionIDSet()
	if err != nil {
		s.log(traceID, ActionRunStatesGet, "error", err)
		return nil, err
	}
	storeQuery := query
	if len(hiddenSessionIDs) > 0 && len(query.SessionIDs) == 0 {
		storeQuery.Limit = 0
	}
	states, err := store.ListRunStates(storeQuery)
	if err != nil {
		s.log(traceID, ActionRunStatesGet, "error", err)
		return nil, err
	}
	states = filterHiddenSessionRunStates(states, hiddenSessionIDs)
	if len(query.SessionIDs) == 0 && len(states) > query.Limit {
		states = states[:query.Limit]
	}
	payload := make([]api.SessionRunState, 0, len(states))
	for _, state := range states {
		payload = append(payload, buildSessionRunStatePayload(state))
	}
	s.log(traceID, ActionRunStatesGet, "success", nil)
	return payload, nil
}

func resolveRunStateQuery(params api.SessionRunStatesGetRequest) (session.RunStateQuery, error) {
	ids := normalizeSessionRunStateIDs(params.SessionIDs)
	if len(ids) > RecentSessionRunStateLimit {
		return session.RunStateQuery{}, ErrTooManySessionRunStateIDs
	}
	if len(ids) > 0 {
		return session.RunStateQuery{SessionIDs: ids}, nil
	}
	limit := RecentSessionRunStateLimit
	if params.Limit != nil {
		limit = *params.Limit
	}
	if limit < 1 || limit > RecentSessionRunStateLimit {
		return session.RunStateQuery{}, ErrInvalidSessionRunStateLimit
	}
	return session.RunStateQuery{Limit: limit}, nil
}

func normalizeSessionRunStateIDs(raw []string) []string {
	seen := make(map[string]struct{}, len(raw))
	ids := make([]string, 0, len(raw))
	for _, value := range raw {
		id := strings.TrimSpace(value)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func filterHiddenSessionRunStates(states []session.SessionRunState, hidden map[string]struct{}) []session.SessionRunState {
	if len(states) == 0 || len(hidden) == 0 {
		return states
	}
	filtered := make([]session.SessionRunState, 0, len(states))
	for _, state := range states {
		if _, excluded := hidden[state.SessionID]; !excluded {
			filtered = append(filtered, state)
		}
	}
	return filtered
}

func buildSessionRunStatePayload(state session.SessionRunState) api.SessionRunState {
	return api.SessionRunState{
		SessionID: state.SessionID, Title: state.Title, Status: string(state.Status), TraceID: state.TraceID,
		StartedAt: formatSessionPayloadTime(state.StartedAt), UpdatedAt: formatSessionPayloadTime(state.UpdatedAt),
		TerminalAt: formatSessionPayloadTime(state.TerminalAt),
	}
}

func formatSessionPayloadTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
