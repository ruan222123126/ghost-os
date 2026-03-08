package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTaskStoreListTasksTolerantSkipsCorruptedFiles(t *testing.T) {
	store, err := NewTaskStore(t.TempDir())
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	good := ScheduledTask{
		ID:              "good-task",
		Message:         "healthy",
		ScheduleType:    taskScheduleTypeInterval,
		IntervalSeconds: 60,
		Enabled:         true,
		CreatedAt:       time.Unix(1, 0).UTC(),
	}
	if err := store.SaveTask(&good); err != nil {
		t.Fatalf("save good task: %v", err)
	}

	if err := os.WriteFile(filepath.Join(store.tasksDir, "bad-json.json"), []byte("{"), 0o600); err != nil {
		t.Fatalf("write bad json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.tasksDir, "bad-config.json"), []byte(`{
  "id": "bad-config",
  "message": "broken",
  "schedule_type": "broken",
  "enabled": true,
  "created_at": "2026-03-08T00:00:00Z"
}`), 0o600); err != nil {
		t.Fatalf("write bad config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.tasksDir, "bad-mismatch.json"), []byte(`{
  "id": "other-task",
  "message": "mismatch",
  "schedule_type": "interval",
  "interval_seconds": 60,
  "enabled": true,
  "created_at": "2026-03-08T00:00:00Z"
}`), 0o600); err != nil {
		t.Fatalf("write mismatched task: %v", err)
	}

	tasks, issues, err := store.ListTasksTolerant()
	if err != nil {
		t.Fatalf("list tasks tolerant: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("unexpected task count: got %d want %d", len(tasks), 1)
	}
	if tasks[0].ID != good.ID {
		t.Fatalf("unexpected healthy task: got %q want %q", tasks[0].ID, good.ID)
	}
	if len(issues) != 3 {
		t.Fatalf("unexpected issue count: got %d want %d", len(issues), 3)
	}

	issuesByID := make(map[string]TaskLoadIssue, len(issues))
	for _, issue := range issues {
		issuesByID[issue.TaskID] = issue
		if issue.Path == "" {
			t.Fatalf("issue path should not be empty: %#v", issue)
		}
		if issue.Error == "" {
			t.Fatalf("issue error should not be empty: %#v", issue)
		}
	}
	if got := issuesByID["bad-json"].Kind; got != taskLoadIssueDecodeError {
		t.Fatalf("unexpected bad-json kind: got %q want %q", got, taskLoadIssueDecodeError)
	}
	if got := issuesByID["bad-config"].Kind; got != taskLoadIssueInvalidConfig {
		t.Fatalf("unexpected bad-config kind: got %q want %q", got, taskLoadIssueInvalidConfig)
	}
	if got := issuesByID["bad-mismatch"].Kind; got != taskLoadIssueIDMismatch {
		t.Fatalf("unexpected bad-mismatch kind: got %q want %q", got, taskLoadIssueIDMismatch)
	}
	if !strings.Contains(issuesByID["bad-mismatch"].Error, "payload id") {
		t.Fatalf("unexpected mismatch error: %q", issuesByID["bad-mismatch"].Error)
	}
	if !strings.HasSuffix(issuesByID["bad-json"].Path, filepath.Join("tasks", "bad-json.json")) {
		t.Fatalf("unexpected bad-json path: %q", issuesByID["bad-json"].Path)
	}

	_, err = store.ListTasks()
	if err == nil {
		t.Fatal("expected strict ListTasks to fail")
	}
	if !errors.Is(err, ErrInvalidTaskConfig) && !errors.Is(err, ErrTaskCorrupted) {
		t.Fatalf("unexpected strict ListTasks error: %v", err)
	}
}
