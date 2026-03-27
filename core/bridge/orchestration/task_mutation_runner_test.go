package orchestration

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestTaskMutationRunnerUpdateRollsBackRegistrationWhenSaveFails(t *testing.T) {
	original := mutationTestTask()
	store := &taskMutationStoreStub{
		loadedTask: original,
		saveErrs:   []error{errors.New("save failed")},
	}
	scheduler := newTaskMutationSchedulerStub(original)
	runner := taskMutationRunner{store: store, scheduler: scheduler}

	message := "updated message"
	_, err := runner.Update(taskUpdateParams{ID: original.ID, Message: &message})
	if err == nil || !strings.Contains(err.Error(), "save failed") {
		t.Fatalf("expected save failure, got %v", err)
	}
	assertRegisteredTaskMessage(t, scheduler, original.ID, original.Message)
	if store.loadedTask.Message != original.Message {
		t.Fatalf("expected store to keep original task, got %#v", store.loadedTask)
	}
	if len(store.savedTasks) != 1 || store.savedTasks[0].Message != message {
		t.Fatalf("expected one attempted save with new message, got %#v", store.savedTasks)
	}
	if len(scheduler.upserts) != 1 || scheduler.upserts[0].Message != original.Message {
		t.Fatalf("expected rollback upsert with original task, got %#v", scheduler.upserts)
	}
}

func TestTaskMutationRunnerUpdateRollsBackPersistedTaskWhenUpsertFails(t *testing.T) {
	original := mutationTestTask()
	store := &taskMutationStoreStub{loadedTask: original}
	scheduler := newTaskMutationSchedulerStub(original)
	scheduler.upsertErrs = []error{errors.New("upsert failed")}
	runner := taskMutationRunner{store: store, scheduler: scheduler}

	message := "updated message"
	_, err := runner.Update(taskUpdateParams{ID: original.ID, Message: &message})
	if err == nil || !strings.Contains(err.Error(), "upsert failed") {
		t.Fatalf("expected upsert failure, got %v", err)
	}
	assertRegisteredTaskMessage(t, scheduler, original.ID, original.Message)
	if store.loadedTask.Message != original.Message {
		t.Fatalf("expected store rollback to restore original task, got %#v", store.loadedTask)
	}
	if len(store.savedTasks) != 2 {
		t.Fatalf("expected update save and rollback save, got %#v", store.savedTasks)
	}
	if store.savedTasks[0].Message != message || store.savedTasks[1].Message != original.Message {
		t.Fatalf("unexpected saved task sequence: %#v", store.savedTasks)
	}
	if len(scheduler.upserts) != 2 {
		t.Fatalf("expected failed upsert and rollback upsert, got %#v", scheduler.upserts)
	}
	if scheduler.upserts[0].Message != message || scheduler.upserts[1].Message != original.Message {
		t.Fatalf("unexpected upsert sequence: %#v", scheduler.upserts)
	}
}

type taskMutationStoreStub struct {
	loadedTask ScheduledTask
	saveErrs   []error
	deleteErrs []error
	deletedIDs []string
	savedTasks []ScheduledTask
}

func (s *taskMutationStoreStub) LoadTask(_ string) (*ScheduledTask, error) {
	task := cloneScheduledTask(s.loadedTask)
	return &task, nil
}

func (s *taskMutationStoreStub) SaveTask(task *ScheduledTask) error {
	s.savedTasks = append(s.savedTasks, cloneScheduledTask(*task))
	if len(s.saveErrs) > 0 {
		err := s.saveErrs[0]
		s.saveErrs = s.saveErrs[1:]
		return err
	}
	s.loadedTask = cloneScheduledTask(*task)
	return nil
}

func (s *taskMutationStoreStub) DeleteTask(taskID string) error {
	s.deletedIDs = append(s.deletedIDs, taskID)
	if len(s.deleteErrs) > 0 {
		err := s.deleteErrs[0]
		s.deleteErrs = s.deleteErrs[1:]
		if err != nil {
			return err
		}
	}
	s.loadedTask = ScheduledTask{}
	return nil
}

type taskMutationSchedulerStub struct {
	registered    map[string]ScheduledTask
	upsertErrs    []error
	unregisterIDs []string
	upserts       []ScheduledTask
}

func newTaskMutationSchedulerStub(task ScheduledTask) *taskMutationSchedulerStub {
	return &taskMutationSchedulerStub{
		registered: map[string]ScheduledTask{task.ID: cloneScheduledTask(task)},
	}
}

func (s *taskMutationSchedulerStub) Upsert(task ScheduledTask) error {
	s.upserts = append(s.upserts, cloneScheduledTask(task))
	if len(s.upsertErrs) > 0 {
		err := s.upsertErrs[0]
		s.upsertErrs = s.upsertErrs[1:]
		if err != nil {
			return err
		}
	}
	s.registered[task.ID] = cloneScheduledTask(task)
	return nil
}

func (s *taskMutationSchedulerStub) Unregister(taskID string) error {
	s.unregisterIDs = append(s.unregisterIDs, taskID)
	delete(s.registered, taskID)
	return nil
}

func (s *taskMutationSchedulerStub) RunNow(task ScheduledTask, _ string) (TaskRunLog, error) {
	return TaskRunLog{TaskID: task.ID}, nil
}

func mutationTestTask() ScheduledTask {
	return ScheduledTask{
		ID:              "task-update-test",
		Message:         "original message",
		TaskKind:        taskKindAgentMessage,
		ScheduleType:    taskScheduleTypeInterval,
		IntervalSeconds: 60,
		Enabled:         true,
		CreatedAt:       time.Unix(100, 0).UTC(),
		UpdatedAt:       time.Unix(200, 0).UTC(),
		NextRunAt:       time.Unix(300, 0).UTC(),
	}
}

func assertRegisteredTaskMessage(
	t *testing.T,
	scheduler *taskMutationSchedulerStub,
	taskID string,
	want string,
) {
	t.Helper()
	task, ok := scheduler.registered[taskID]
	if !ok {
		t.Fatalf("expected task %q to stay registered", taskID)
	}
	if task.Message != want {
		t.Fatalf("unexpected registered task: %#v", task)
	}
}
