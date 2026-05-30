package tasks

import (
	"testing"
	"time"
)

func TestRunningRunLogWriterOverwritesRunCardsOnSameRunID(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	base := TaskRunLog{
		TaskID:          "task-progress",
		RunID:           "run-progress",
		TraceID:         "trace-progress",
		ScheduledAt:     time.Unix(1_700_100_000, 0).UTC(),
		StartedAt:       time.Unix(1_700_100_001, 0).UTC(),
		Status:          RunStatusRunning,
		SessionIDOutput: "session-progress",
	}
	appendTaskRunLogForTest(t, store, base)

	writer := newRunningRunLogWriter(store, base)
	if err := writer.WriteRunningRunLog(RunningRunLogUpdate{
		RunCards: []RunCard{{
			CardID:    "card-1",
			Kind:      RunCardKindWorkflowLLM,
			Title:     "llm-node",
			StartedAt: time.Unix(1_700_100_002, 0).UTC(),
			Status:    RunStatusRunning,
		}},
	}); err != nil {
		t.Fatalf("write running log: %v", err)
	}

	logs, err := store.ListRunLogs(base.TaskID, 1)
	if err != nil {
		t.Fatalf("list run logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("unexpected log count: got %d want 1", len(logs))
	}
	if logs[0].RunID != base.RunID || len(logs[0].RunCards) != 1 {
		t.Fatalf("unexpected persisted running log: %#v", logs[0])
	}
}
