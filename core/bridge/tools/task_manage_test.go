package tools

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

type fakeTaskManager struct {
	createInput TaskCreateRequest
	updateInput TaskUpdateRequest
	deletedID   string
	getIDs      []string
	listCalled  bool

	createResult TaskPayload
	updateResult TaskPayload
	getResult    TaskPayload
	listResult   []TaskPayload
	deleteResult TaskDeleteResult

	createErr error
	updateErr error
	getErr    error
	listErr   error
	deleteErr error
}

func (f *fakeTaskManager) CreateAgentTask(_ context.Context, req TaskCreateRequest, _ string) (TaskPayload, error) {
	f.createInput = req
	return f.createResult, f.createErr
}

func (f *fakeTaskManager) UpdateAgentTask(_ context.Context, req TaskUpdateRequest, _ string) (TaskPayload, error) {
	f.updateInput = req
	return f.updateResult, f.updateErr
}

func (f *fakeTaskManager) GetTask(_ context.Context, id string, _ string) (TaskPayload, error) {
	f.getIDs = append(f.getIDs, id)
	return f.getResult, f.getErr
}

func (f *fakeTaskManager) ListTasks(_ context.Context, _ string) ([]TaskPayload, error) {
	f.listCalled = true
	return f.listResult, f.listErr
}

func (f *fakeTaskManager) DeleteTask(_ context.Context, id string, _ string) (TaskDeleteResult, error) {
	f.deletedID = id
	return f.deleteResult, f.deleteErr
}

func TestTaskManageToolCreateWithSessionID(t *testing.T) {
	manager := &fakeTaskManager{createResult: TaskPayload{ID: "task-1", Message: "hello", SessionID: "session-1", TaskKind: taskManageKindAgentMessage, ScheduleType: "interval", IntervalSeconds: 60, Enabled: true, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}}
	tool := NewTaskManageTool(manager)

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"operation":"create","message":"hello","session_id":"session-1","interval_seconds":60}`), "trace-task-create")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if manager.createInput.SessionID != "session-1" {
		t.Fatalf("unexpected session id: got %q want %q", manager.createInput.SessionID, "session-1")
	}
	if manager.createInput.IntervalSeconds != 60 {
		t.Fatalf("unexpected interval: got %d want %d", manager.createInput.IntervalSeconds, 60)
	}
	var payload TaskPayload
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if payload.ID != "task-1" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestTaskManageToolCreateDefaultsToNewConversation(t *testing.T) {
	manager := &fakeTaskManager{createResult: TaskPayload{ID: "task-2", Message: "follow up", TaskKind: taskManageKindAgentMessage, ScheduleType: "cron", CronExpr: "*/5 * * * *", Enabled: true, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}}
	tool := NewTaskManageTool(manager)

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"operation":"create","message":"follow up","cron_expr":"*/5 * * * *"}`), "trace-task-create")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if manager.createInput.SessionID != "" {
		t.Fatalf("expected empty session_id, got %q", manager.createInput.SessionID)
	}
	var payload TaskPayload
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if payload.SessionID != "" {
		t.Fatalf("expected empty payload session_id, got %q", payload.SessionID)
	}
}

func TestTaskManageToolCreateRejectsInvalidInput(t *testing.T) {
	tool := NewTaskManageTool(&fakeTaskManager{})

	tests := []string{
		`{"operation":"create","interval_seconds":60}`,
		`{"operation":"create","message":"x"}`,
		`{"operation":"create","message":"x","interval_seconds":60,"cron_expr":"*/5 * * * *"}`,
		`{"operation":"create","message":"x","interval_seconds":0}`,
		`{"operation":"create","message":"x","interval_seconds":60,"action":"TASK_CREATE"}`,
	}
	for _, input := range tests {
		if _, err := tool.Execute(context.Background(), json.RawMessage(input), "trace-task-create"); err == nil {
			t.Fatalf("expected error for input %s", input)
		}
	}
}

func TestTaskManageToolUpdateAllowsClearingSessionAndEnabled(t *testing.T) {
	manager := &fakeTaskManager{
		getResult:    TaskPayload{ID: "task-1", Message: "before", SessionID: "session-old", TaskKind: taskManageKindAgentMessage, ScheduleType: "interval", IntervalSeconds: 60, Enabled: true, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		updateResult: TaskPayload{ID: "task-1", Message: "after", TaskKind: taskManageKindAgentMessage, ScheduleType: "interval", IntervalSeconds: 120, Enabled: false, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
	}
	tool := NewTaskManageTool(manager)

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"operation":"update","id":"task-1","message":"after","session_id":"","interval_seconds":120,"enabled":false}`), "trace-task-update")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if !reflect.DeepEqual(manager.getIDs, []string{"task-1"}) {
		t.Fatalf("unexpected get ids: %+v", manager.getIDs)
	}
	if manager.updateInput.ID != "task-1" {
		t.Fatalf("unexpected update id: %q", manager.updateInput.ID)
	}
	if manager.updateInput.SessionID == nil || *manager.updateInput.SessionID != "" {
		t.Fatalf("expected cleared session_id, got %+v", manager.updateInput.SessionID)
	}
	if manager.updateInput.Enabled == nil || *manager.updateInput.Enabled {
		t.Fatalf("expected enabled=false, got %+v", manager.updateInput.Enabled)
	}
	var payload TaskPayload
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if payload.IntervalSeconds != 120 || payload.Enabled {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestTaskManageToolRejectsNonAgentTasks(t *testing.T) {
	manager := &fakeTaskManager{getResult: TaskPayload{ID: "task-sys", TaskKind: "system_action", ScheduleType: "interval", IntervalSeconds: 60, Enabled: true, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}}
	tool := NewTaskManageTool(manager)

	for _, input := range []string{
		`{"operation":"get","id":"task-sys"}`,
		`{"operation":"update","id":"task-sys","message":"x"}`,
		`{"operation":"delete","id":"task-sys"}`,
	} {
		if _, err := tool.Execute(context.Background(), json.RawMessage(input), "trace-task"); err == nil || !strings.Contains(err.Error(), "agent_message") {
			t.Fatalf("expected agent_message rejection for %s, got %v", input, err)
		}
	}
}

func TestTaskManageToolListFiltersSystemTasks(t *testing.T) {
	manager := &fakeTaskManager{listResult: []TaskPayload{
		{ID: "task-1", TaskKind: taskManageKindAgentMessage, ScheduleType: "interval", IntervalSeconds: 60, Enabled: true, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		{ID: "task-2", TaskKind: "system_action", ScheduleType: "interval", IntervalSeconds: 60, Enabled: true, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
	}}
	tool := NewTaskManageTool(manager)

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"operation":"list"}`), "trace-task-list")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if !manager.listCalled {
		t.Fatal("expected list to be called")
	}
	var payload []TaskPayload
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if len(payload) != 1 || payload[0].ID != "task-1" {
		t.Fatalf("unexpected filtered payload: %+v", payload)
	}
}

func TestTaskManageToolDeleteAndGet(t *testing.T) {
	manager := &fakeTaskManager{
		getResult:    TaskPayload{ID: "task-1", TaskKind: taskManageKindAgentMessage, ScheduleType: "interval", IntervalSeconds: 60, Enabled: true, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		deleteResult: TaskDeleteResult{ID: "task-1", Deleted: true},
	}
	tool := NewTaskManageTool(manager)

	getOutput, err := tool.Execute(context.Background(), json.RawMessage(`{"operation":"get","id":"task-1"}`), "trace-task-get")
	if err != nil {
		t.Fatalf("get returned error: %v", err)
	}
	deleteOutput, err := tool.Execute(context.Background(), json.RawMessage(`{"operation":"delete","id":"task-1"}`), "trace-task-delete")
	if err != nil {
		t.Fatalf("delete returned error: %v", err)
	}
	if manager.deletedID != "task-1" {
		t.Fatalf("unexpected deleted id: %q", manager.deletedID)
	}
	var deleted TaskDeleteResult
	if err := json.Unmarshal([]byte(deleteOutput), &deleted); err != nil {
		t.Fatalf("decode delete output: %v", err)
	}
	if !deleted.Deleted {
		t.Fatalf("unexpected delete payload: %+v", deleted)
	}
	if !strings.Contains(getOutput, `"id":"task-1"`) {
		t.Fatalf("unexpected get output: %s", getOutput)
	}
}
