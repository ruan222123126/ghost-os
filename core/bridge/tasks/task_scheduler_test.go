package tasks

import (
	"context"
	"errors"
	"testing"
	"time"

	bridgeconfig "ghost-os/bridge/config"
)

func TestTaskSchedulerDefaultsToFiveMinutes(t *testing.T) {
	scheduler := NewTaskScheduler(nil, nil)
	want := 5 * time.Minute
	if got := scheduler.taskExecutionTimeout(); got != want {
		t.Fatalf("unexpected default execution timeout: got %s want %s", got, want)
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

func TestTaskSchedulerSetExecutionTimeoutAffectsNewRunsOnly(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	scheduler := NewTaskScheduler(store, nil)
	firstStarted := make(chan struct{})
	firstRelease := make(chan struct{})
	secondStarted := make(chan struct{})
	runIndex := 0
	scheduler.execute = func(ctx context.Context, _ ScheduledTask, _ string) ExecutionResult {
		runIndex++
		if runIndex == 1 {
			close(firstStarted)
			<-firstRelease
			<-ctx.Done()
			return ExecutionResult{Status: RunStatusError}
		}
		close(secondStarted)
		<-ctx.Done()
		return ExecutionResult{Status: RunStatusError}
	}
	if err := scheduler.Start(); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}
	defer scheduler.Stop()

	scheduler.SetExecutionTimeout(80 * time.Millisecond)
	task := ScheduledTask{
		ID:              "timeout-update-task",
		Message:         "timeout update",
		ScheduleType:    ScheduleTypeInterval,
		IntervalSeconds: 60,
		Enabled:         true,
	}
	firstDone := make(chan RunLog, 1)
	go func() {
		run, _ := scheduler.RunNow(task, "trace-first")
		firstDone <- run
	}()

	select {
	case <-firstStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first run to start")
	}

	scheduler.SetExecutionTimeout(20 * time.Millisecond)
	close(firstRelease)

	firstRun := <-firstDone
	if firstRun.Error != "task execution timed out after 80ms" {
		t.Fatalf("unexpected first run timeout: got %q", firstRun.Error)
	}

	secondRun, err := scheduler.RunNow(task, "trace-second")
	if err != nil {
		t.Fatalf("run now second: %v", err)
	}
	select {
	case <-secondStarted:
	default:
	}
	if secondRun.Error != "task execution timed out after 20ms" {
		t.Fatalf("unexpected second run timeout: got %q", secondRun.Error)
	}
}

func TestTaskSchedulerWithConfiguredTimeoutUsesProvidedValue(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	timeout := time.Duration(bridgeconfig.DefaultTaskExecutionTimeoutMS) * time.Millisecond
	scheduler := NewTaskSchedulerWithTimeout(store, nil, timeout)
	if got := scheduler.taskExecutionTimeout(); got != timeout {
		t.Fatalf("unexpected configured timeout: got %s want %s", got, timeout)
	}
}
