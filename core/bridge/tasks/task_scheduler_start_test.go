package tasks

import (
	"bytes"
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTaskSchedulerStartSkipsCorruptedTasks(t *testing.T) {
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
		NextRunAt:       time.Now().UTC().Add(50 * time.Millisecond),
	}
	if err := store.SaveTask(&good); err != nil {
		t.Fatalf("save good task: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.TasksDir(), "bad-json.json"), []byte("{"), 0o600); err != nil {
		t.Fatalf("write bad json: %v", err)
	}

	logOutput := captureTaskSchedulerLogs(t)
	scheduler := NewTaskScheduler(store, nil)
	ran := make(chan ScheduledTask, 1)
	scheduler.execute = func(_ context.Context, task ScheduledTask, _ string) ExecutionResult {
		select {
		case ran <- task:
		default:
		}
		return ExecutionResult{Status: RunStatusSuccess, SessionIDOutput: "session-ok"}
	}
	defer scheduler.Stop()

	if err := scheduler.Start(); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}
	select {
	case task := <-ran:
		if task.ID != good.ID {
			t.Fatalf("unexpected executed task: got %q want %q", task.ID, good.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for healthy task execution")
	}
	if scheduler.lookupTask(good.ID) == nil {
		t.Fatalf("expected healthy task %q to be registered", good.ID)
	}
	if scheduler.lookupTask("bad-json") != nil {
		t.Fatal("corrupted task should not be registered")
	}

	logs := logOutput.String()
	if !strings.Contains(logs, "task scheduler skipped corrupted task:") {
		t.Fatalf("expected corrupted-task log, got %q", logs)
	}
	if !strings.Contains(logs, "task_id=bad-json") {
		t.Fatalf("expected bad-json task id in logs, got %q", logs)
	}
	if !strings.Contains(logs, filepath.Join("tasks", "bad-json.json")) {
		t.Fatalf("expected bad-json path in logs, got %q", logs)
	}
}

func TestTaskSchedulerStartSkipsInvalidTaskConfig(t *testing.T) {
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
		NextRunAt:       time.Now().UTC().Add(50 * time.Millisecond),
	}
	if err := store.SaveTask(&good); err != nil {
		t.Fatalf("save good task: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.TasksDir(), "bad-cron.json"), []byte(`{
  "id": "bad-cron",
  "message": "broken cron",
  "schedule_type": "cron",
  "cron_expr": "not a cron",
  "enabled": true,
  "created_at": "2026-03-08T00:00:00Z"
}`), 0o600); err != nil {
		t.Fatalf("write bad cron: %v", err)
	}

	logOutput := captureTaskSchedulerLogs(t)
	scheduler := NewTaskScheduler(store, nil)
	ran := make(chan ScheduledTask, 1)
	scheduler.execute = func(_ context.Context, task ScheduledTask, _ string) ExecutionResult {
		select {
		case ran <- task:
		default:
		}
		return ExecutionResult{Status: RunStatusSuccess, SessionIDOutput: "session-ok"}
	}
	defer scheduler.Stop()

	if err := scheduler.Start(); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}
	select {
	case task := <-ran:
		if task.ID != good.ID {
			t.Fatalf("unexpected executed task: got %q want %q", task.ID, good.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for healthy task execution")
	}
	if scheduler.lookupTask("bad-cron") != nil {
		t.Fatal("invalid cron task should not be registered")
	}

	logs := logOutput.String()
	if !strings.Contains(logs, "task scheduler skipped invalid task during registration:") {
		t.Fatalf("expected invalid-registration log, got %q", logs)
	}
	if !strings.Contains(logs, "task_id=bad-cron") {
		t.Fatalf("expected bad-cron task id in logs, got %q", logs)
	}
}

func TestTaskSchedulerStartReturnsErrorOnTaskDirectoryFailure(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}
	if err := os.RemoveAll(store.TasksDir()); err != nil {
		t.Fatalf("remove tasks dir: %v", err)
	}
	if err := os.WriteFile(store.TasksDir(), []byte("not-a-directory"), 0o600); err != nil {
		t.Fatalf("replace tasks dir with file: %v", err)
	}

	scheduler := NewTaskScheduler(store, nil)
	err = scheduler.Start()
	if err == nil {
		t.Fatal("expected scheduler start to fail")
	}
	if !strings.Contains(err.Error(), "read task directory") {
		t.Fatalf("unexpected start error: %v", err)
	}

	scheduler.mu.Lock()
	running := scheduler.running
	scheduler.mu.Unlock()
	if running {
		t.Fatal("scheduler should reset running state after start failure")
	}
}

func captureTaskSchedulerLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buffer bytes.Buffer
	oldWriter := log.Writer()
	oldFlags := log.Flags()
	oldPrefix := log.Prefix()
	log.SetOutput(&buffer)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() {
		log.SetOutput(oldWriter)
		log.SetFlags(oldFlags)
		log.SetPrefix(oldPrefix)
	})
	return &buffer
}
