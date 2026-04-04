package tasks

import (
	"context"
	"testing"
	"time"
)

func TestTaskRegistrationStopRetiresRegistration(t *testing.T) {
	task := ScheduledTask{
		ID:              "retired-task",
		Message:         "do not restart",
		ScheduleType:    ScheduleTypeInterval,
		IntervalSeconds: 60,
		Enabled:         true,
	}
	reg := &taskRegistration{task: task}

	reg.stop()
	reg.waitIdle()

	gotTask, runCtx, skipped, reason := reg.beginRun(task, time.Second)
	if !skipped {
		t.Fatal("expected retired registration to skip future runs")
	}
	if gotTask.ID != task.ID {
		t.Fatalf("unexpected task returned: got %q want %q", gotTask.ID, task.ID)
	}
	if runCtx != nil {
		t.Fatal("retired registration should not create a run context")
	}
	if reason != skipRunReasonRegistrationRetired {
		t.Fatalf("unexpected skip reason: got %q want %q", reason, skipRunReasonRegistrationRetired)
	}
}

func TestTaskSchedulerRunNowFallsBackFromRetiredRegistration(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	task := ScheduledTask{
		ID:              "retired-run-now-task",
		Message:         "run me once",
		ScheduleType:    ScheduleTypeInterval,
		IntervalSeconds: 60,
		Enabled:         true,
	}
	if err := store.SaveTask(&task); err != nil {
		t.Fatalf("save task: %v", err)
	}

	scheduler := NewTaskScheduler(store, nil)
	runCount := 0
	scheduler.execute = func(_ context.Context, got ScheduledTask, _ string) ExecutionResult {
		runCount++
		if got.ID != task.ID {
			t.Fatalf("unexpected task executed: got %q want %q", got.ID, task.ID)
		}
		return ExecutionResult{Status: RunStatusSuccess, SessionIDOutput: "session-ok"}
	}
	if err := scheduler.Start(); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}
	defer scheduler.Stop()

	if err := scheduler.Unregister(task.ID); err != nil {
		t.Fatalf("unregister running registration before stale injection: %v", err)
	}

	stale := &taskRegistration{task: task}
	stale.stop()
	scheduler.mu.Lock()
	scheduler.tasks[task.ID] = stale
	scheduler.mu.Unlock()

	run, err := scheduler.RunNow(task, "trace-retired-run-now")
	if err != nil {
		t.Fatalf("run now: %v", err)
	}
	if run.Status != RunStatusSuccess {
		t.Fatalf("unexpected run status: got %q want %q", run.Status, RunStatusSuccess)
	}
	if runCount != 1 {
		t.Fatalf("unexpected execute count: got %d want 1", runCount)
	}

	runs, err := store.ListRunLogs(task.ID, 1)
	if err != nil {
		t.Fatalf("list run logs: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run log, got %d", len(runs))
	}
	if runs[0].Status != RunStatusSuccess {
		t.Fatalf("unexpected persisted run status: got %q want %q", runs[0].Status, RunStatusSuccess)
	}
}
