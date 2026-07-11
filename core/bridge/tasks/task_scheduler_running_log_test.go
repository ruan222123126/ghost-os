package tasks

import (
	"context"
	"testing"
	"time"
)

type blockingRunSessionExecutor struct {
	started chan struct{}
	release chan struct{}
}

func newBlockingRunSessionExecutor() *blockingRunSessionExecutor {
	return &blockingRunSessionExecutor{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
}

func (e *blockingRunSessionExecutor) PrepareRunSession(
	_ context.Context,
	_ ScheduledTask,
	_ string,
) (RunSession, error) {
	return RunSession{SessionID: "session-running"}, nil
}

func (e *blockingRunSessionExecutor) Execute(
	ctx context.Context,
	task ScheduledTask,
	_ string,
) ExecutionResult {
	if task.SessionID != "session-running" {
		return ExecutionResult{Status: RunStatusError, Error: "session id was not injected"}
	}
	close(e.started)
	select {
	case <-e.release:
		return ExecutionResult{Status: RunStatusSuccess, ResponsePreview: "done"}
	case <-ctx.Done():
		return ExecutionResult{
			Status: RunStatusError,
			Error:  ctx.Err().Error(),
			RunCards: []RunCard{{
				CardID: "cancel-card",
				Kind:   RunCardKindAgentTask,
				Status: RunStatusCancelled,
				Error:  ctx.Err().Error(),
			}},
		}
	}
}

func TestTaskSchedulerRunNowWritesRunningLogThenOverwritesFinal(t *testing.T) {
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
		ID:              "running-log-task",
		Message:         "run with visible session",
		ScheduleType:    ScheduleTypeInterval,
		IntervalSeconds: 60,
		Enabled:         true,
	}
	done := make(chan RunLog, 1)
	go func() {
		run, _ := scheduler.RunNow(task, "trace-running-log")
		done <- run
	}()

	select {
	case <-executor.started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for running task")
	}
	running := requireSingleRunLog(t, store, task.ID)
	if running.Status != RunStatusRunning || running.SessionIDOutput != "session-running" {
		t.Fatalf("unexpected running log: %#v", running)
	}

	close(executor.release)
	final := <-done
	listed := requireSingleRunLog(t, store, task.ID)
	if listed.RunID != running.RunID || final.RunID != running.RunID {
		t.Fatalf("run id was not reused: running=%q final=%q listed=%q", running.RunID, final.RunID, listed.RunID)
	}
	if listed.Status != RunStatusSuccess {
		t.Fatalf("unexpected final status: %#v", listed)
	}
}

func TestTaskSchedulerStartNowReturnsRunningLogBeforeExecutionFinishes(t *testing.T) {
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
		ID:              "start-now-task",
		Message:         "run without waiting",
		ScheduleType:    ScheduleTypeInterval,
		IntervalSeconds: 60,
		Enabled:         true,
	}

	run, err := scheduler.StartNow(task, "trace-start-now")
	if err != nil {
		t.Fatalf("start now: %v", err)
	}
	if run.Status != RunStatusRunning || run.SessionIDOutput != "session-running" {
		t.Fatalf("unexpected running payload: %#v", run)
	}

	select {
	case <-executor.started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for running task")
	}
	running := requireSingleRunLog(t, store, task.ID)
	if running.RunID != run.RunID {
		t.Fatalf("run id was not reused: returned=%q stored=%q", run.RunID, running.RunID)
	}
	if running.Status != RunStatusRunning {
		t.Fatalf("unexpected stored running log: %#v", running)
	}

	close(executor.release)
	requireRunStatus(t, store, task.ID, RunStatusSuccess)
}

func requireSingleRunLog(t *testing.T, store *Store, taskID string) RunLog {
	t.Helper()

	runs, err := store.ListRunLogs(taskID, 10)
	if err != nil {
		t.Fatalf("list run logs: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected one run log, got %d", len(runs))
	}
	return runs[0]
}

func requireRunStatus(t *testing.T, store *Store, taskID string, status string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		run := requireSingleRunLog(t, store, taskID)
		if run.Status == status {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for task %q status %q", taskID, status)
}
