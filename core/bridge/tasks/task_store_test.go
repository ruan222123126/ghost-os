package tasks

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTaskStoreListTasksTolerantSkipsCorruptedFiles(t *testing.T) {
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
	}
	if err := store.SaveTask(&good); err != nil {
		t.Fatalf("save good task: %v", err)
	}

	if err := os.WriteFile(filepath.Join(store.TasksDir(), "bad-json.json"), []byte("{"), 0o600); err != nil {
		t.Fatalf("write bad json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.TasksDir(), "bad-config.json"), []byte(`{
  "id": "bad-config",
  "message": "broken",
  "schedule_type": "broken",
  "enabled": true,
  "created_at": "2026-03-08T00:00:00Z"
}`), 0o600); err != nil {
		t.Fatalf("write bad config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.TasksDir(), "bad-mismatch.json"), []byte(`{
  "id": "other-task",
  "message": "mismatch",
  "schedule_type": "interval",
  "interval_seconds": 60,
  "enabled": true,
  "created_at": "2026-03-08T00:00:00Z"
}`), 0o600); err != nil {
		t.Fatalf("write mismatched task: %v", err)
	}

	tasks, issues, err := store.ListTasksTolerant()
	if err != nil {
		t.Fatalf("list tasks tolerant: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("unexpected task count: got %d want %d", len(tasks), 1)
	}
	if tasks[0].ID != good.ID {
		t.Fatalf("unexpected healthy task: got %q want %q", tasks[0].ID, good.ID)
	}
	if len(issues) != 3 {
		t.Fatalf("unexpected issue count: got %d want %d", len(issues), 3)
	}

	issuesByID := make(map[string]TaskLoadIssue, len(issues))
	for _, issue := range issues {
		issuesByID[issue.TaskID] = issue
		if issue.Path == "" {
			t.Fatalf("issue path should not be empty: %#v", issue)
		}
		if issue.Error == "" {
			t.Fatalf("issue error should not be empty: %#v", issue)
		}
	}
	if got := issuesByID["bad-json"].Kind; got != LoadIssueDecodeError {
		t.Fatalf("unexpected bad-json kind: got %q want %q", got, LoadIssueDecodeError)
	}
	if got := issuesByID["bad-config"].Kind; got != LoadIssueInvalidConfig {
		t.Fatalf("unexpected bad-config kind: got %q want %q", got, LoadIssueInvalidConfig)
	}
	if got := issuesByID["bad-mismatch"].Kind; got != LoadIssueIDMismatch {
		t.Fatalf("unexpected bad-mismatch kind: got %q want %q", got, LoadIssueIDMismatch)
	}
	if !strings.Contains(issuesByID["bad-mismatch"].Error, "payload id") {
		t.Fatalf("unexpected mismatch error: %q", issuesByID["bad-mismatch"].Error)
	}
	if !strings.HasSuffix(issuesByID["bad-json"].Path, filepath.Join("tasks", "bad-json.json")) {
		t.Fatalf("unexpected bad-json path: %q", issuesByID["bad-json"].Path)
	}

	_, err = store.ListTasks()
	if err == nil {
		t.Fatal("expected strict ListTasks to fail")
	}
	if !errors.Is(err, ErrInvalidTaskConfig) && !errors.Is(err, ErrTaskCorrupted) {
		t.Fatalf("unexpected strict ListTasks error: %v", err)
	}
}

func TestTaskStoreWorkflowRoundTrip(t *testing.T) {
	store, err := NewStore(t.TempDir(), workflowStoreValidator)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	task := ScheduledTask{
		ID:       "workflow-task",
		TaskKind: KindWorkflow,
		Workflow: &WorkflowDefinition{
			Nodes: []WorkflowNode{
				{ID: "start-node", Type: "start"},
				{ID: "end-node", Type: "end"},
			},
			Edges: []WorkflowEdge{{FromNodeID: "start-node", ToNodeID: "end-node"}},
		},
		ScheduleType:    ScheduleTypeInterval,
		IntervalSeconds: 60,
		Enabled:         true,
	}
	if err := store.SaveTask(&task); err != nil {
		t.Fatalf("save workflow task: %v", err)
	}

	loaded, err := store.LoadTask(task.ID)
	if err != nil {
		t.Fatalf("load workflow task: %v", err)
	}
	if loaded.TaskKind != KindWorkflow {
		t.Fatalf("unexpected task kind: got %q want %q", loaded.TaskKind, KindWorkflow)
	}
	if loaded.Workflow == nil || len(loaded.Workflow.Nodes) != 2 || len(loaded.Workflow.Edges) != 1 {
		t.Fatalf("unexpected workflow payload: %#v", loaded.Workflow)
	}
	if loaded.Workflow.Edges[0].FromNodeID != "start-node" || loaded.Workflow.Edges[0].ToNodeID != "end-node" {
		t.Fatalf("unexpected workflow edge: %#v", loaded.Workflow.Edges[0])
	}
}

func TestTaskStoreListTasksTolerantFlagsInvalidWorkflowFiles(t *testing.T) {
	store, err := NewStore(t.TempDir(), workflowStoreValidator)
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}

	if err := os.WriteFile(filepath.Join(store.TasksDir(), "bad-workflow-json.json"), []byte(`{
  "id": "bad-workflow-json",
  "task_kind": "workflow",
  "workflow": "broken",
  "schedule_type": "interval",
  "interval_seconds": 60,
  "enabled": true,
  "created_at": "2026-03-08T00:00:00Z"
}`), 0o600); err != nil {
		t.Fatalf("write invalid workflow json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.TasksDir(), "bad-workflow-config.json"), []byte(`{
  "id": "bad-workflow-config",
  "task_kind": "workflow",
  "workflow": {
    "nodes": [
      { "id": "start-node", "type": "start" },
      { "id": "end-node", "type": "end" }
    ],
    "edges": []
  },
  "schedule_type": "interval",
  "interval_seconds": 60,
  "enabled": true,
  "created_at": "2026-03-08T00:00:00Z"
}`), 0o600); err != nil {
		t.Fatalf("write invalid workflow config: %v", err)
	}

	_, issues, err := store.ListTasksTolerant()
	if err != nil {
		t.Fatalf("list tasks tolerant: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("unexpected issue count: got %d want %d", len(issues), 2)
	}
	issuesByID := make(map[string]TaskLoadIssue, len(issues))
	for _, issue := range issues {
		issuesByID[issue.TaskID] = issue
	}
	if issuesByID["bad-workflow-json"].Kind != LoadIssueDecodeError {
		t.Fatalf("unexpected workflow json issue: %#v", issuesByID["bad-workflow-json"])
	}
	if issuesByID["bad-workflow-config"].Kind != LoadIssueInvalidConfig {
		t.Fatalf("unexpected workflow config issue: %#v", issuesByID["bad-workflow-config"])
	}
}

func workflowStoreValidator(task *ScheduledTask) error {
	if NormalizeKind(task.TaskKind) != KindWorkflow {
		return nil
	}
	if task.Workflow == nil {
		return fmt.Errorf("%w: workflow is required", ErrInvalidTaskConfig)
	}
	if len(task.Workflow.Nodes) != 2 {
		return fmt.Errorf("%w: workflow requires 2 nodes", ErrInvalidTaskConfig)
	}
	if len(task.Workflow.Edges) != 1 {
		return fmt.Errorf("%w: workflow requires 1 edge", ErrInvalidTaskConfig)
	}
	return nil
}
