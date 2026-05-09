package tasks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeKindDefaultsEmptyToAgentMessage(t *testing.T) {
	if got := NormalizeKind("  "); got != KindAgentMessage {
		t.Fatalf("unexpected normalized kind: got %q want %q", got, KindAgentMessage)
	}
}

func TestNormalizeKindPreservesUnsupportedValue(t *testing.T) {
	const kind = "unexpected_kind"
	if got := NormalizeKind(kind); got != kind {
		t.Fatalf("unexpected normalized kind: got %q want %q", got, kind)
	}
}

func TestNormalizeScheduledTaskRejectsUnsupportedTaskKind(t *testing.T) {
	task := ScheduledTask{
		ID:              "task-invalid-kind",
		Message:         "hello",
		TaskKind:        "unexpected_kind",
		ScheduleType:    ScheduleTypeInterval,
		IntervalSeconds: 60,
		Enabled:         true,
	}

	err := NormalizeScheduledTask(&task, strictTaskKindValidator)
	if err == nil || !strings.Contains(err.Error(), `unsupported task_kind "unexpected_kind"`) {
		t.Fatalf("expected unsupported task_kind error, got %v", err)
	}
}

func TestTaskStoreListTasksTolerantFlagsUnsupportedTaskKind(t *testing.T) {
	store, err := NewStore(t.TempDir(), strictTaskKindValidator)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	if err := os.WriteFile(filepath.Join(store.TasksDir(), "bad-kind.json"), []byte(`{
  "id": "bad-kind",
  "message": "broken",
  "task_kind": "unexpected_kind",
  "schedule_type": "interval",
  "interval_seconds": 60,
  "enabled": true,
  "created_at": "2026-03-08T00:00:00Z"
}`), 0o600); err != nil {
		t.Fatalf("write invalid task: %v", err)
	}

	tasks, issues, err := store.ListTasksTolerant()
	if err != nil {
		t.Fatalf("list tasks tolerant: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("unexpected healthy tasks: %#v", tasks)
	}
	if len(issues) != 1 {
		t.Fatalf("unexpected issue count: got %d want %d", len(issues), 1)
	}
	if issues[0].Kind != LoadIssueInvalidConfig {
		t.Fatalf("unexpected issue kind: got %q want %q", issues[0].Kind, LoadIssueInvalidConfig)
	}
	if !strings.Contains(issues[0].Error, `unsupported task_kind "unexpected_kind"`) {
		t.Fatalf("unexpected issue error: %q", issues[0].Error)
	}
}

func strictTaskKindValidator(task *ScheduledTask) error {
	switch task.TaskKind {
	case KindAgentMessage, KindSystemAction, KindWorkflow, KindOrchestration:
		return nil
	default:
		return fmt.Errorf("%w: unsupported task_kind %q", ErrInvalidTaskConfig, task.TaskKind)
	}
}
