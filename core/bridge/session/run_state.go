package session

import (
	"strings"
	"time"
)

type RunStatus string

const (
	RunStatusIdle          RunStatus = "idle"
	RunStatusRunning       RunStatus = "running"
	RunStatusAwaitingHuman RunStatus = "awaiting_human"
	RunStatusSuccess       RunStatus = "success"
	RunStatusError         RunStatus = "error"
	RunStatusCancelled     RunStatus = "cancelled"
)

type LastRunState struct {
	Status     RunStatus `json:"status"`
	TraceID    string    `json:"trace_id"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
	TerminalAt time.Time `json:"terminal_at,omitempty"`
}

type SessionRunState struct {
	SessionID  string
	Title      string
	Status     RunStatus
	TraceID    string
	StartedAt  time.Time
	UpdatedAt  time.Time
	TerminalAt time.Time
}

type RunStateQuery struct {
	Limit      int
	SessionIDs []string
}

func (s *Session) SetLastRunState(status RunStatus, traceID string, at time.Time) bool {
	if s == nil {
		return false
	}
	traceID = strings.TrimSpace(traceID)
	if traceID == "" || !isKnownRunStatus(status) || status == RunStatusIdle {
		return false
	}
	when := at.UTC()
	if when.IsZero() {
		when = time.Now().UTC()
	}

	next := cloneLastRunState(s.LastRunState)
	if next == nil || strings.TrimSpace(next.TraceID) != traceID {
		next = &LastRunState{
			TraceID:   traceID,
			StartedAt: when,
		}
	}
	if status == RunStatusRunning && next.StartedAt.IsZero() {
		next.StartedAt = when
	}
	next.Status = status
	next.TraceID = traceID
	next.UpdatedAt = when
	if isTerminalRunStatus(status) {
		next.TerminalAt = when
	} else {
		next.TerminalAt = time.Time{}
	}

	if equalLastRunState(s.LastRunState, next) {
		return false
	}
	s.LastRunState = next
	s.UpdatedAt = when
	return true
}

func cloneLastRunState(state *LastRunState) *LastRunState {
	if state == nil {
		return nil
	}
	cloned := *state
	cloned.TraceID = strings.TrimSpace(cloned.TraceID)
	return &cloned
}

func equalLastRunState(left *LastRunState, right *LastRunState) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.Status == right.Status &&
		strings.TrimSpace(left.TraceID) == strings.TrimSpace(right.TraceID) &&
		left.StartedAt.Equal(right.StartedAt) &&
		left.UpdatedAt.Equal(right.UpdatedAt) &&
		left.TerminalAt.Equal(right.TerminalAt)
}

func isKnownRunStatus(status RunStatus) bool {
	switch status {
	case RunStatusIdle,
		RunStatusRunning,
		RunStatusAwaitingHuman,
		RunStatusSuccess,
		RunStatusError,
		RunStatusCancelled:
		return true
	default:
		return false
	}
}

func isTerminalRunStatus(status RunStatus) bool {
	switch status {
	case RunStatusSuccess, RunStatusError, RunStatusCancelled:
		return true
	default:
		return false
	}
}
