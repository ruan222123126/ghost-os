package tasks

import (
	"errors"
	"testing"
)

func TestTaskSchedulerUpsertReturnsErrorAfterStop(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	scheduler := NewTaskScheduler(store, nil)
	if err := scheduler.Start(); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}
	scheduler.Stop()

	task := ScheduledTask{ID: "stopped-upsert", ScheduleType: ScheduleTypeInterval, IntervalSeconds: 60, Enabled: true}
	err = scheduler.Upsert(task)
	if !errors.Is(err, ErrTaskSchedulerStopped) {
		t.Fatalf("expected scheduler stopped error, got %v", err)
	}
	if scheduler.lookupTask(task.ID) != nil {
		t.Fatalf("expected task %q to stay unregistered", task.ID)
	}
}

func TestTaskSchedulerRunNowReturnsErrorAfterStop(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	scheduler := NewTaskScheduler(store, nil)
	if err := scheduler.Start(); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}
	scheduler.Stop()

	task := ScheduledTask{ID: "stopped-run-now", ScheduleType: ScheduleTypeInterval, IntervalSeconds: 60, Enabled: true}
	_, err = scheduler.RunNow(task, "trace-stopped-run")
	if !errors.Is(err, ErrTaskSchedulerStopped) {
		t.Fatalf("expected scheduler stopped error, got %v", err)
	}
}
