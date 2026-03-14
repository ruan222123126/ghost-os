package tasks

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTaskStoreListTasksTolerantSkipsCorruptedFiles(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	good := ScheduledTask{
		ID:              "good-task",
		Message:         "healthy",
		ScheduleType:    ScheduleTypeInterval,
		IntervalSeconds: 60,
		Enabled:         true,
		CreatedAt:       time.Unix(1, 0).UTC(),
	}
	if err := store.SaveTask(&good); err != nil {
		t.Fatalf("save good task: %v", err)
	}

	if err := os.WriteFile(filepath.Join(store.TasksDir(), "bad-json.json"), []byte("{"), 0o600); err != nil {
		t.Fatalf("write bad json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.TasksDir(), "bad-config.json"), []byte(`{
  "id": "bad-config",
  "message": "broken",
  "schedule_type": "broken",
  "enabled": true,
  "created_at": "2026-03-08T00:00:00Z"
}`), 0o600); err != nil {
		t.Fatalf("write bad config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.TasksDir(), "bad-mismatch.json"), []byte(`{
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
	if got := issuesByID["bad-json"].Kind; got != LoadIssueDecodeError {
		t.Fatalf("unexpected bad-json kind: got %q want %q", got, LoadIssueDecodeError)
	}
	if got := issuesByID["bad-config"].Kind; got != LoadIssueInvalidConfig {
		t.Fatalf("unexpected bad-config kind: got %q want %q", got, LoadIssueInvalidConfig)
	}
	if got := issuesByID["bad-mismatch"].Kind; got != LoadIssueIDMismatch {
		t.Fatalf("unexpected bad-mismatch kind: got %q want %q", got, LoadIssueIDMismatch)
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

func TestTaskStoreAppendRunLogPrunesOldLogs(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	const taskID = "task-prune-old-logs"
	base := time.Unix(1_700_000_000, 0).UTC()
	total := defaultTaskRunLogRetention + 3
	for index := 1; index <= total; index++ {
		stamp := base.Add(time.Duration(index) * time.Minute)
		appendTaskRunLogForTest(t, store, TaskRunLog{
			TaskID:      taskID,
			RunID:       fmt.Sprintf("run-%03d", index),
			ScheduledAt: stamp,
			StartedAt:   stamp,
			Status:      RunStatusSuccess,
		})
	}

	if got := countTaskRunLogFiles(t, store, taskID); got != defaultTaskRunLogRetention {
		t.Fatalf("unexpected retained file count: got %d want %d", got, defaultTaskRunLogRetention)
	}

	logs, err := store.ListRunLogs(taskID, total)
	if err != nil {
		t.Fatalf("list run logs: %v", err)
	}
	if len(logs) != defaultTaskRunLogRetention {
		t.Fatalf("unexpected retained log count: got %d want %d", len(logs), defaultTaskRunLogRetention)
	}
	if logs[0].RunID != fmt.Sprintf("run-%03d", total) {
		t.Fatalf("unexpected newest run: got %q want %q", logs[0].RunID, fmt.Sprintf("run-%03d", total))
	}
	oldestKept := total - defaultTaskRunLogRetention + 1
	if logs[len(logs)-1].RunID != fmt.Sprintf("run-%03d", oldestKept) {
		t.Fatalf("unexpected oldest retained run: got %q want %q", logs[len(logs)-1].RunID, fmt.Sprintf("run-%03d", oldestKept))
	}
	if _, err := os.Stat(filepath.Join(store.LogsDir(), taskID, "run-001.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected oldest log file to be pruned, stat err=%v", err)
	}

	limited, err := store.ListRunLogs(taskID, 5)
	if err != nil {
		t.Fatalf("list limited run logs: %v", err)
	}
	if len(limited) != 5 {
		t.Fatalf("unexpected limited log count: got %d want %d", len(limited), 5)
	}
	if limited[0].RunID != fmt.Sprintf("run-%03d", total) || limited[4].RunID != fmt.Sprintf("run-%03d", total-4) {
		t.Fatalf("unexpected limited ordering: got first=%q fifth=%q", limited[0].RunID, limited[4].RunID)
	}
}

func TestTaskStoreAppendRunLogDoesNotPruneBelowLimit(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	const taskID = "task-no-prune"
	base := time.Unix(1_700_000_500, 0).UTC()
	for index := 1; index <= 3; index++ {
		stamp := base.Add(time.Duration(index) * time.Minute)
		appendTaskRunLogForTest(t, store, TaskRunLog{
			TaskID:      taskID,
			RunID:       fmt.Sprintf("run-%03d", index),
			ScheduledAt: stamp,
			StartedAt:   stamp,
			Status:      RunStatusSuccess,
		})
	}

	if got := countTaskRunLogFiles(t, store, taskID); got != 3 {
		t.Fatalf("unexpected retained file count: got %d want %d", got, 3)
	}
	logs, err := store.ListRunLogs(taskID, 10)
	if err != nil {
		t.Fatalf("list run logs: %v", err)
	}
	if len(logs) != 3 {
		t.Fatalf("unexpected log count: got %d want %d", len(logs), 3)
	}
	if logs[0].RunID != "run-003" || logs[2].RunID != "run-001" {
		t.Fatalf("unexpected ordering without prune: %#v", logs)
	}
}

func TestTaskStoreAppendRunLogKeepsLatestLogs(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	const taskID = "task-keep-latest"
	base := time.Unix(1_700_001_000, 0).UTC()
	appendTaskRunLogForTest(t, store, TaskRunLog{
		TaskID:      taskID,
		RunID:       "run-old-started",
		ScheduledAt: base.Add(3 * time.Hour),
		StartedAt:   base.Add(1 * time.Hour),
		Status:      RunStatusSuccess,
	})
	appendTaskRunLogForTest(t, store, TaskRunLog{
		TaskID:      taskID,
		RunID:       "run-mid-fallback",
		ScheduledAt: base.Add(2 * time.Hour),
		Status:      RunStatusSuccess,
	})
	appendTaskRunLogForTest(t, store, TaskRunLog{
		TaskID:      taskID,
		RunID:       "run-new-started",
		ScheduledAt: base.Add(1 * time.Hour),
		StartedAt:   base.Add(4 * time.Hour),
		Status:      RunStatusSuccess,
	})

	err = store.PruneRunLogs(taskID, 2)
	if err != nil {
		t.Fatalf("prune run logs: %v", err)
	}

	logs, err := store.ListRunLogs(taskID, 10)
	if err != nil {
		t.Fatalf("list run logs: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("unexpected retained log count: got %d want %d", len(logs), 2)
	}
	if logs[0].RunID != "run-new-started" || logs[1].RunID != "run-mid-fallback" {
		t.Fatalf("unexpected retained ordering: %#v", logs)
	}
	if _, err := os.Stat(filepath.Join(store.LogsDir(), taskID, "run-old-started.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected started-at oldest log to be pruned, stat err=%v", err)
	}
}

func TestTaskStoreAppendRunLogToleratesCorruptedLogFiles(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	const taskID = "task-corrupt-logs"
	base := time.Unix(1_700_002_000, 0).UTC()
	for index := 1; index <= defaultTaskRunLogRetention; index++ {
		stamp := base.Add(time.Duration(index) * time.Minute)
		appendTaskRunLogForTest(t, store, TaskRunLog{
			TaskID:      taskID,
			RunID:       fmt.Sprintf("run-%03d", index),
			ScheduledAt: stamp,
			StartedAt:   stamp,
			Status:      RunStatusSuccess,
		})
	}

	logDir := filepath.Join(store.LogsDir(), taskID)
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		t.Fatalf("mkdir log dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(logDir, "broken.json"), []byte("{"), 0o600); err != nil {
		t.Fatalf("write broken log: %v", err)
	}

	appendTaskRunLogForTest(t, store, TaskRunLog{
		TaskID:      taskID,
		RunID:       fmt.Sprintf("run-%03d", defaultTaskRunLogRetention+1),
		ScheduledAt: base.Add(time.Duration(defaultTaskRunLogRetention+1) * time.Minute),
		StartedAt:   base.Add(time.Duration(defaultTaskRunLogRetention+1) * time.Minute),
		Status:      RunStatusSuccess,
	})

	if got := countTaskRunLogFiles(t, store, taskID); got != defaultTaskRunLogRetention {
		t.Fatalf("unexpected retained file count: got %d want %d", got, defaultTaskRunLogRetention)
	}
	if _, err := os.Stat(filepath.Join(logDir, "broken.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected broken log file to be pruned, stat err=%v", err)
	}
	logs, err := store.ListRunLogs(taskID, defaultTaskRunLogRetention)
	if err != nil {
		t.Fatalf("list run logs after pruning broken file: %v", err)
	}
	if len(logs) != defaultTaskRunLogRetention {
		t.Fatalf("unexpected log count after prune: got %d want %d", len(logs), defaultTaskRunLogRetention)
	}
	if logs[0].RunID != fmt.Sprintf("run-%03d", defaultTaskRunLogRetention+1) {
		t.Fatalf("unexpected newest run after prune: got %q", logs[0].RunID)
	}
}

func appendTaskRunLogForTest(t *testing.T, store *TaskStore, run TaskRunLog) {
	t.Helper()
	if err := store.AppendRunLog(run); err != nil {
		t.Fatalf("append run log: %v", err)
	}
}

func countTaskRunLogFiles(t *testing.T, store *TaskStore, taskID string) int {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(store.LogsDir(), taskID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0
		}
		t.Fatalf("read log dir: %v", err)
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		count++
	}
	return count
}
