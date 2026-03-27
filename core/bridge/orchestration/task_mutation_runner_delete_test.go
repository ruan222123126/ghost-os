package orchestration

import (
	"errors"
	"strings"
	"testing"
)

func TestTaskMutationRunnerDeleteRollsBackRegistrationWhenDeleteFails(t *testing.T) {
	original := mutationTestTask()
	store := &taskMutationStoreStub{
		loadedTask: original,
		deleteErrs: []error{errors.New("delete failed")},
	}
	scheduler := newTaskMutationSchedulerStub(original)
	runner := taskMutationRunner{store: store, scheduler: scheduler}

	_, err := runner.Delete(taskIDParams{ID: original.ID})
	if err == nil || !strings.Contains(err.Error(), "delete failed") {
		t.Fatalf("expected delete failure, got %v", err)
	}
	assertRegisteredTaskMessage(t, scheduler, original.ID, original.Message)
	if store.loadedTask.Message != original.Message {
		t.Fatalf("expected store rollback to restore original task, got %#v", store.loadedTask)
	}
	if len(store.savedTasks) != 1 || store.savedTasks[0].Message != original.Message {
		t.Fatalf("expected one rollback save with original task, got %#v", store.savedTasks)
	}
	if len(store.deletedIDs) != 1 || store.deletedIDs[0] != original.ID {
		t.Fatalf("expected delete attempt for %q, got %#v", original.ID, store.deletedIDs)
	}
	if len(scheduler.upserts) != 1 || scheduler.upserts[0].Message != original.Message {
		t.Fatalf("expected rollback upsert with original task, got %#v", scheduler.upserts)
	}
}
