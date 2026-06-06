package taskstore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveDeprecatedSystemTasksDeletesOnlySystemTaskDocuments(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, "legacy", `{"task_kind":"system_action"}`)
	writeTaskFile(t, dir, "agent", `{"task_kind":"agent_message"}`)
	writeTaskFile(t, dir, "broken", `{`)

	store := &fakeStore{dir: dir}
	scheduler := &fakeScheduler{}
	if err := RemoveDeprecatedSystemTasks(store, scheduler); err != nil {
		t.Fatalf("remove deprecated system tasks: %v", err)
	}

	if len(store.deleted) != 1 || store.deleted[0] != "legacy" {
		t.Fatalf("unexpected deleted tasks: %#v", store.deleted)
	}
	if len(scheduler.unregistered) != 1 || scheduler.unregistered[0] != "legacy" {
		t.Fatalf("unexpected unregistered tasks: %#v", scheduler.unregistered)
	}
	if _, err := os.Stat(filepath.Join(dir, "agent.json")); err != nil {
		t.Fatalf("expected agent task to remain: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "broken.json")); err != nil {
		t.Fatalf("expected broken task to remain: %v", err)
	}
}

func writeTaskFile(t *testing.T, dir string, id string, data string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, id+".json"), []byte(data), 0o600); err != nil {
		t.Fatalf("write task file: %v", err)
	}
}

type fakeStore struct {
	dir     string
	deleted []string
}

func (s *fakeStore) TasksDir() string {
	return s.dir
}

func (s *fakeStore) DeleteTask(taskID string) error {
	s.deleted = append(s.deleted, taskID)
	return os.Remove(filepath.Join(s.dir, taskID+".json"))
}

type fakeScheduler struct {
	unregistered []string
}

func (s *fakeScheduler) Unregister(taskID string) error {
	s.unregistered = append(s.unregistered, taskID)
	return nil
}
