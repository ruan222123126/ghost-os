package tasks

import (
	"bytes"
	"context"
	"errors"
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

func TestTaskSchedulerStopCancelsRunningTaskUnit(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	task := ScheduledTask{
		ID:              "stop-cancel-task",
		Message:         "cancel me",
		ScheduleType:    ScheduleTypeInterval,
		IntervalSeconds: 60,
		Enabled:         true,
		CreatedAt:       time.Unix(1, 0).UTC(),
		NextRunAt:       time.Now().UTC().Add(25 * time.Millisecond),
	}
	if err := store.SaveTask(&task); err != nil {
		t.Fatalf("save task: %v", err)
	}

	scheduler := NewTaskScheduler(store, nil)
	scheduler.executionTimeout = 5 * time.Second
	started := make(chan struct{})
	finished := make(chan struct{})
	scheduler.execute = func(ctx context.Context, _ ScheduledTask, _ string) ExecutionResult {
		select {
		case <-started:
		default:
			close(started)
		}
		<-ctx.Done()
		close(finished)
		return ExecutionResult{Status: RunStatusError, Error: ctx.Err().Error()}
	}

	if err := scheduler.Start(); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for task to start")
	}

	stopStarted := time.Now()
	scheduler.Stop()
	if elapsed := time.Since(stopStarted); elapsed > time.Second {
		t.Fatalf("expected Stop to return promptly after cancellation, duration=%s", elapsed)
	}

	select {
	case <-finished:
	default:
		t.Fatal("running task should receive cancellation before Stop returns")
	}

	runs, err := store.ListRunLogs(task.ID, 1)
	if err != nil {
		t.Fatalf("list run logs: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run log, got %d", len(runs))
	}
	if runs[0].Status != RunStatusCancelled {
		t.Fatalf("unexpected run status: got %q want %q", runs[0].Status, RunStatusCancelled)
	}
	if runs[0].Error != "task execution cancelled" {
		t.Fatalf("unexpected cancellation error: got %q", runs[0].Error)
	}

	stored, err := store.LoadTask(task.ID)
	if err != nil {
		t.Fatalf("load task: %v", err)
	}
	if stored.LastError != "task execution cancelled" {
		t.Fatalf("unexpected last_error: got %q", stored.LastError)
	}
}

func TestTaskSchedulerRunMarksTimeout(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	scheduler := NewTaskScheduler(store, nil)
	scheduler.executionTimeout = 50 * time.Millisecond
	scheduler.execute = func(ctx context.Context, _ ScheduledTask, _ string) ExecutionResult {
		<-ctx.Done()
		return ExecutionResult{Status: RunStatusSuccess}
	}
	if err := scheduler.Start(); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}
	defer scheduler.Stop()

	task := ScheduledTask{
		ID:              "timeout-task",
		Message:         "too slow",
		ScheduleType:    ScheduleTypeInterval,
		IntervalSeconds: 60,
		Enabled:         true,
	}

	run, err := scheduler.RunNow(task, "trace-timeout")
	if err != nil {
		t.Fatalf("run now: %v", err)
	}
	if run.Status != RunStatusError {
		t.Fatalf("unexpected timeout status: got %q want %q", run.Status, RunStatusError)
	}
	wantError := "task execution timed out after 50ms"
	if run.Error != wantError {
		t.Fatalf("unexpected timeout error: got %q want %q", run.Error, wantError)
	}

	stored, err := store.LoadTask(task.ID)
	if err != nil {
		t.Fatalf("load task: %v", err)
	}
	if stored.LastError != wantError {
		t.Fatalf("unexpected last_error: got %q want %q", stored.LastError, wantError)
	}

	runs, err := store.ListRunLogs(task.ID, 1)
	if err != nil {
		t.Fatalf("list run logs: %v", err)
	}
	if len(runs) != 1 || runs[0].Error != wantError {
		t.Fatalf("unexpected timeout logs: %#v", runs)
	}
}

func TestTaskSchedulerUnregisterCancelsRunningTask(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	scheduler := NewTaskScheduler(store, nil)
	scheduler.executionTimeout = 5 * time.Second
	started := make(chan struct{})
	finished := make(chan struct{})
	scheduler.execute = func(ctx context.Context, _ ScheduledTask, _ string) ExecutionResult {
		select {
		case <-started:
		default:
			close(started)
		}
		<-ctx.Done()
		close(finished)
		return ExecutionResult{Status: RunStatusError, Error: ctx.Err().Error()}
	}
	if err := scheduler.Start(); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}
	defer scheduler.Stop()

	task := ScheduledTask{
		ID:              "unregister-cancel-task",
		Message:         "cancel me too",
		ScheduleType:    ScheduleTypeInterval,
		IntervalSeconds: 60,
		Enabled:         true,
		CreatedAt:       time.Unix(1, 0).UTC(),
		NextRunAt:       time.Now().UTC().Add(25 * time.Millisecond),
	}
	if err := store.SaveTask(&task); err != nil {
		t.Fatalf("save task: %v", err)
	}
	if err := scheduler.Upsert(task); err != nil {
		t.Fatalf("upsert task: %v", err)
	}

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for task to start")
	}

	if err := scheduler.Unregister(task.ID); err != nil {
		t.Fatalf("unregister task: %v", err)
	}
	select {
	case <-finished:
	default:
		t.Fatal("running task should receive cancellation before Unregister returns")
	}
	if scheduler.lookupTask(task.ID) != nil {
		t.Fatalf("expected task %q to be removed from scheduler", task.ID)
	}

	runs, err := store.ListRunLogs(task.ID, 1)
	if err != nil {
		t.Fatalf("list run logs: %v", err)
	}
	if len(runs) != 1 || runs[0].Status != RunStatusCancelled {
		t.Fatalf("unexpected unregister logs: %#v", runs)
	}
}

func TestTaskSchedulerRunNowHonorsTimeout(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	scheduler := NewTaskScheduler(store, nil)
	scheduler.executionTimeout = 30 * time.Millisecond
	scheduler.execute = func(ctx context.Context, _ ScheduledTask, _ string) ExecutionResult {
		<-ctx.Done()
		return ExecutionResult{Status: RunStatusError, Error: ctx.Err().Error()}
	}
	if err := scheduler.Start(); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}
	defer scheduler.Stop()

	task := ScheduledTask{
		ID:              "run-now-timeout-task",
		Message:         "manual too slow",
		ScheduleType:    ScheduleTypeInterval,
		IntervalSeconds: 60,
		Enabled:         true,
	}

	run, err := scheduler.RunNow(task, "trace-run-now")
	if err != nil {
		t.Fatalf("run now: %v", err)
	}
	if run.Status != RunStatusError {
		t.Fatalf("unexpected run-now status: got %q want %q", run.Status, RunStatusError)
	}
	if run.Error != "task execution timed out after 30ms" {
		t.Fatalf("unexpected run-now timeout error: got %q", run.Error)
	}
}

func TestTaskSchedulerRunKeepsNormalErrorsAsError(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	scheduler := NewTaskScheduler(store, nil)
	wantErr := errors.New("boom")
	scheduler.execute = func(_ context.Context, _ ScheduledTask, _ string) ExecutionResult {
		return ExecutionResult{Status: RunStatusError, Error: wantErr.Error()}
	}
	if err := scheduler.Start(); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}
	defer scheduler.Stop()

	task := ScheduledTask{
		ID:              "normal-error-task",
		Message:         "fail me",
		ScheduleType:    ScheduleTypeInterval,
		IntervalSeconds: 60,
		Enabled:         true,
	}

	run, err := scheduler.RunNow(task, "trace-error")
	if err != nil {
		t.Fatalf("run now: %v", err)
	}
	if run.Status != RunStatusError {
		t.Fatalf("unexpected error status: got %q want %q", run.Status, RunStatusError)
	}
	if run.Error != wantErr.Error() {
		t.Fatalf("unexpected error text: got %q want %q", run.Error, wantErr.Error())
	}

	stored, err := store.LoadTask(task.ID)
	if err != nil {
		t.Fatalf("load task: %v", err)
	}
	if stored.LastError != wantErr.Error() {
		t.Fatalf("unexpected last_error: got %q want %q", stored.LastError, wantErr.Error())
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
