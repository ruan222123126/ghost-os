package tasks

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTaskStoreListTasksTolerantFlagsInvalidTaskFilename(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	fileName := "bad id.json"
	path := filepath.Join(store.TasksDir(), fileName)
	if err := os.WriteFile(path, []byte(`{"id":"good-task","schedule_type":"interval","interval_seconds":60,"enabled":true}`), 0o600); err != nil {
		t.Fatalf("write invalid task file: %v", err)
	}

	_, issues, err := store.ListTasksTolerant()
	if err != nil {
		t.Fatalf("list tasks tolerant: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("unexpected issue count: got %d want 1", len(issues))
	}
	issue := issues[0]
	if issue.Kind != LoadIssueInvalidFilename {
		t.Fatalf("unexpected issue kind: got %q want %q", issue.Kind, LoadIssueInvalidFilename)
	}
	if issue.TaskID != "bad id" {
		t.Fatalf("unexpected issue task id: got %q want %q", issue.TaskID, "bad id")
	}
	if !strings.HasSuffix(issue.Path, filepath.Join("tasks", fileName)) {
		t.Fatalf("unexpected issue path: %q", issue.Path)
	}
	if !strings.Contains(issue.Error, "invalid task id") {
		t.Fatalf("unexpected issue error: %q", issue.Error)
	}
}

func TestTaskStoreListTasksFailsOnInvalidTaskFilename(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	path := filepath.Join(store.TasksDir(), "bad id.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write invalid task file: %v", err)
	}

	_, err = store.ListTasks()
	if !errors.Is(err, ErrInvalidTaskID) {
		t.Fatalf("expected invalid task id error, got %v", err)
	}
}
