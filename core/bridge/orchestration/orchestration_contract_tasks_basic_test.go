package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestTaskSchemaDefinesKindSpecificContracts(t *testing.T) {
	defs := loadTaskSchemaDefs(t)
	assertSchemaOneOfRefs(t, defs, "taskCreateRequest",
		"agentMessageTaskCreateRequest",
		"workflowTaskCreateRequest",
		"orchestrationTaskCreateRequest",
	)
	assertSchemaOneOfRefs(t, defs, "taskPayload",
		"agentMessageTaskPayload",
		"workflowTaskPayload",
		"orchestrationTaskPayload",
	)
	assertSchemaRequired(t, defs, "agentMessageTaskCreateRequest", "message")
	assertSchemaRequired(t, defs, "workflowTaskCreateRequest", "workflow")
	assertSchemaRequired(t, defs, "orchestrationTaskCreateRequest", "name")
	assertSchemaRequired(t, defs, "orchestrationTaskCreateRequest", "orchestration")
	assertSchemaRequired(t, defs, "taskUpdateRequest", "id")
	assertSchemaProperty(t, defs, "taskUpdateRequest", "name")
	assertSchemaProperty(t, defs, "taskRuntimeOverrides", "provider_name")
	assertSchemaProperty(t, defs, "taskRuntimeOverrides", "model")
	assertSchemaProperty(t, defs, "taskRuntimeOverrides", "system_prompt")
	assertSchemaProperty(t, defs, "taskRuntimeOverrides", "preset_id")
	assertSchemaProperty(t, defs, "taskRuntimeOverrides", "tool_allowlist")
	assertSchemaProperty(t, defs, "taskRuntimeOverrides", "tool_allowlist_only")
	assertSchemaProperty(t, defs, "taskRuntimeOverrides", "max_turns")
	assertSchemaProperty(t, defs, "agentMessageTaskCreateRequest", "runtime_overrides")
	assertSchemaProperty(t, defs, "taskUpdateRequest", "runtime_overrides")
	assertSchemaProperty(t, defs, "agentMessageTaskPayload", "runtime_overrides")
	assertSchemaProperty(t, defs, "workflowNode", "start")
	assertSchemaProperty(t, defs, "workflowNode", "tool")
	assertSchemaProperty(t, defs, "workflowNode", "llm")
	assertSchemaProperty(t, defs, "workflowNode", "agent")
	assertSchemaProperty(t, defs, "workflowNode", "if")
	assertSchemaProperty(t, defs, "workflowNode", "loop")
	assertSchemaProperty(t, defs, "orchestrationNode", "group")
	assertSchemaProperty(t, defs, "orchestrationNode", "agent")
	assertSchemaProperty(t, defs, "workflowStartNode", "inputs")
	assertSchemaRequired(t, defs, "workflowInputVariable", "name")
	assertSchemaRequired(t, defs, "workflowInputVariable", "type")
	assertSchemaProperty(t, defs, "workflowInputVariable", "default")
	assertSchemaRequired(t, defs, "workflowToolNode", "tool_name")
	assertSchemaRequired(t, defs, "workflowLLMNode", "prompt")
	assertSchemaRequired(t, defs, "workflowAgentNode", "message")
	assertSchemaProperty(t, defs, "workflowAgentNode", "runtime_overrides")
	assertSchemaRequired(t, defs, "workflowIfNode", "operator")
	assertSchemaRequired(t, defs, "workflowIfNode", "true_node_id")
	assertSchemaRequired(t, defs, "workflowIfNode", "false_node_id")
	assertSchemaRequired(t, defs, "workflowLoopNode", "max_iterations")
	assertSchemaRequired(t, defs, "workflowLoopNode", "body_node_id")
	assertSchemaRequired(t, defs, "workflowLoopNode", "exit_node_id")
	assertSchemaRequired(t, defs, "taskRelayConfig", "max_rounds")
	assertSchemaRequired(t, defs, "orchestrationGroupNode", "title")
	assertSchemaRequired(t, defs, "orchestrationGroupNode", "shared_context")
	assertSchemaRequired(t, defs, "orchestrationGroupNode", "speaking_mode")
	assertSchemaRequired(t, defs, "orchestrationGroupNode", "max_rounds")
	assertSchemaProperty(t, defs, "orchestrationGroupNode", "owner_agent_id")
	assertSchemaRequired(t, defs, "orchestrationAgentNode", "title")
	assertSchemaRequired(t, defs, "orchestrationAgentNode", "message")
	assertSchemaRequired(t, defs, "orchestrationEdge", "kind")
}

func loadTaskSchemaDefs(t *testing.T) map[string]any {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve caller path")
	}
	path := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "shared", "schema", "defs", "tasks.json"))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read tasks schema: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("decode tasks schema: %v", err)
	}
	defs, ok := document["$defs"].(map[string]any)
	if !ok {
		t.Fatal("tasks schema defs missing")
	}
	return defs
}

func assertSchemaOneOfRefs(t *testing.T, defs map[string]any, name string, want ...string) {
	t.Helper()
	definition := schemaDefinition(t, defs, name)
	items, ok := definition["oneOf"].([]any)
	if !ok || len(items) != len(want) {
		t.Fatalf("unexpected oneOf in %s: %#v", name, definition["oneOf"])
	}
	for index, ref := range want {
		item, ok := items[index].(map[string]any)
		if !ok || item["$ref"] != "#/$defs/"+ref {
			t.Fatalf("unexpected oneOf[%d] in %s: %#v", index, name, items[index])
		}
	}
}

func assertSchemaRequired(t *testing.T, defs map[string]any, name string, field string) {
	t.Helper()
	required, ok := schemaDefinition(t, defs, name)["required"].([]any)
	if !ok {
		t.Fatalf("required missing in %s", name)
	}
	for _, item := range required {
		if item == field {
			return
		}
	}
	t.Fatalf("field %q is not required in %s: %#v", field, name, required)
}

func assertSchemaProperty(t *testing.T, defs map[string]any, name string, field string) {
	t.Helper()
	properties := schemaProperties(t, defs, name)
	if _, ok := properties[field]; !ok {
		t.Fatalf("field %q missing in %s", field, name)
	}
}

func schemaDefinition(t *testing.T, defs map[string]any, name string) map[string]any {
	t.Helper()
	definition, ok := defs[name].(map[string]any)
	if !ok {
		t.Fatalf("schema definition %q missing", name)
	}
	return definition
}

func schemaProperties(t *testing.T, defs map[string]any, name string) map[string]any {
	t.Helper()
	properties, ok := schemaDefinition(t, defs, name)["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties missing in %s", name)
	}
	return properties
}

func TestValidateTaskDefinitionRejectsUnsupportedTaskKind(t *testing.T) {
	task := ScheduledTask{
		TaskKind: "unexpected_kind",
		Message:  "hello",
	}

	err := validateTaskDefinition(&task)
	if err == nil || !strings.Contains(err.Error(), `unsupported task_kind "unexpected_kind"`) {
		t.Fatalf("expected unsupported task_kind error, got %v", err)
	}
}

func TestTaskCreateRejectsUnsupportedTaskKind(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	_, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        "unexpected_kind",
		Message:         "hello",
		IntervalSeconds: 60,
	}, "trace-invalid-kind")
	if err == nil {
		t.Fatal("expected task create to fail")
	}
	if code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d want %d", code, http.StatusBadRequest)
	}
	if !strings.Contains(err.Error(), `unsupported task_kind "unexpected_kind"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskExecutorAdapterRejectsUnsupportedTaskKind(t *testing.T) {
	result := taskExecutorAdapter{}.Execute(context.Background(), ScheduledTask{
		TaskKind: "unexpected_kind",
		Message:  "hello",
	}, "trace-invalid-kind")
	if result.Status != taskRunStatusError {
		t.Fatalf("unexpected execution status: got %q want %q", result.Status, taskRunStatusError)
	}
	if result.Error != `unsupported task_kind "unexpected_kind"` {
		t.Fatalf("unexpected execution error: %q", result.Error)
	}
}

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

func (s *taskMutationSchedulerStub) StartNow(task ScheduledTask, _ string) (TaskRunLog, error) {
	return TaskRunLog{TaskID: task.ID, Status: taskRunStatusRunning}, nil
}

func (s *taskMutationSchedulerStub) StopRun(_ context.Context, taskID string, runID string) (TaskRunLog, error) {
	return TaskRunLog{TaskID: taskID, RunID: runID, Status: taskRunStatusCancelled}, nil
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
