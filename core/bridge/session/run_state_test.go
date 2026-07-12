package session

import (
	"testing"
	"time"
)

func TestSetLastRunStatePreservesTraceLifecycle(t *testing.T) {
	sess := &Session{ID: "session-1"}
	startedAt := time.Date(2026, time.July, 11, 8, 0, 0, 0, time.UTC)
	awaitingAt := startedAt.Add(time.Minute)
	terminalAt := awaitingAt.Add(time.Minute)

	if !sess.SetLastRunState(RunStatusRunning, "trace-1", startedAt) {
		t.Fatal("expected running state to change session")
	}
	if !sess.SetLastRunState(RunStatusAwaitingHuman, "trace-1", awaitingAt) {
		t.Fatal("expected awaiting_human state to change session")
	}
	if !sess.SetLastRunState(RunStatusSuccess, "trace-1", terminalAt) {
		t.Fatal("expected success state to change session")
	}

	state := sess.LastRunState
	if state == nil {
		t.Fatal("expected last run state")
	}
	if state.Status != RunStatusSuccess || state.TraceID != "trace-1" {
		t.Fatalf("unexpected terminal state: %+v", state)
	}
	if !state.StartedAt.Equal(startedAt) || !state.UpdatedAt.Equal(terminalAt) || !state.TerminalAt.Equal(terminalAt) {
		t.Fatalf("unexpected lifecycle timestamps: %+v", state)
	}
}

func TestSetLastRunStateStartsNewTrace(t *testing.T) {
	sess := &Session{}
	firstAt := time.Date(2026, time.July, 11, 8, 0, 0, 0, time.UTC)
	secondAt := firstAt.Add(time.Hour)
	sess.SetLastRunState(RunStatusSuccess, "trace-1", firstAt)
	sess.SetLastRunState(RunStatusRunning, "trace-2", secondAt)

	state := sess.LastRunState
	if state == nil || state.TraceID != "trace-2" || state.Status != RunStatusRunning {
		t.Fatalf("unexpected next run state: %+v", state)
	}
	if !state.StartedAt.Equal(secondAt) || !state.TerminalAt.IsZero() {
		t.Fatalf("new trace should reset terminal lifecycle: %+v", state)
	}
}
