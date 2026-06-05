package tasks

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestTaskSchedulerStopRunCancelsRegisteredTask(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	executor := newBlockingRunSessionExecutor()
	scheduler := NewTaskScheduler(store, executor)
	if err := scheduler.Start(); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}
	defer scheduler.Stop()

	task := ScheduledTask{
		ID:              "stop-run-registered",
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
	if err := scheduler.Upsert(task); err != nil {
		t.Fatalf("upsert task: %v", err)
	}

	select {
	case <-executor.started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for registered task to start")
	}

	running := requireSingleRunLog(t, store, task.ID)
	stopped, err := scheduler.StopRun(context.Background(), task.ID, running.RunID)
	if err != nil {
		t.Fatalf("stop run: %v", err)
	}
	if stopped.Status != RunStatusCancelled {
		t.Fatalf("unexpected stop result: %#v", stopped)
	}
	assertCancelledRunHasNoError(t, stopped)
	requireRunStatus(t, store, task.ID, RunStatusCancelled)
}

func TestTaskSchedulerStopRunCancelsTemporaryManualTask(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	executor := newBlockingRunSessionExecutor()
	scheduler := NewTaskScheduler(store, executor)
	if err := scheduler.Start(); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}
	defer scheduler.Stop()

	task := ScheduledTask{
		ID:              "stop-run-manual",
		Message:         "manual task",
		ScheduleType:    ScheduleTypeInterval,
		IntervalSeconds: 60,
		Enabled:         true,
	}

	run, err := scheduler.StartNow(task, "trace-manual-stop")
	if err != nil {
		t.Fatalf("start now: %v", err)
	}
	select {
	case <-executor.started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for manual task to start")
	}

	stopped, err := scheduler.StopRun(context.Background(), task.ID, run.RunID)
	if err != nil {
		t.Fatalf("stop run: %v", err)
	}
	if stopped.RunID != run.RunID || stopped.Status != RunStatusCancelled {
		t.Fatalf("unexpected stopped run: %#v", stopped)
	}
	assertCancelledRunHasNoError(t, stopped)
	requireRunStatus(t, store, task.ID, RunStatusCancelled)
}

func TestTaskSchedulerStopRunRejectsStaleRunID(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	executor := newBlockingRunSessionExecutor()
	scheduler := NewTaskScheduler(store, executor)
	if err := scheduler.Start(); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}
	defer scheduler.Stop()

	task := ScheduledTask{
		ID:              "stop-run-stale",
		Message:         "stay running",
		ScheduleType:    ScheduleTypeInterval,
		IntervalSeconds: 60,
		Enabled:         true,
	}

	run, err := scheduler.StartNow(task, "trace-manual-stale")
	if err != nil {
		t.Fatalf("start now: %v", err)
	}
	select {
	case <-executor.started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for task to start")
	}

	_, err = scheduler.StopRun(context.Background(), task.ID, run.RunID+"-stale")
	if !errors.Is(err, ErrTaskRunNotRunning) {
		t.Fatalf("expected not running error, got %v", err)
	}
	if current := requireSingleRunLog(t, store, task.ID); current.Status != RunStatusRunning {
		t.Fatalf("unexpected running log after stale stop: %#v", current)
	}

	close(executor.release)
	requireRunStatus(t, store, task.ID, RunStatusSuccess)
}

func assertCancelledRunHasNoError(t *testing.T, run RunLog) {
	t.Helper()
	if run.Error != "" {
		t.Fatalf("cancelled run should not report error: %#v", run)
	}
	if len(run.RunCards) != 1 {
		t.Fatalf("expected one run card, got %#v", run.RunCards)
	}
	card := run.RunCards[0]
	if card.Status != RunStatusCancelled || card.Error != "" {
		t.Fatalf("cancelled run card should not report error: %#v", card)
	}
}
