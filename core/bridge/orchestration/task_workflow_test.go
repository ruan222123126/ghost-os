package orchestration

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestValidateTaskDefinitionWorkflowAcceptsStartToEnd(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: validWorkflowDefinition(),
	}
	if err := validateTaskDefinition(&task); err != nil {
		t.Fatalf("validate workflow task: %v", err)
	}
}

func TestValidateTaskDefinitionWorkflowRejectsInvalidCases(t *testing.T) {
	tests := []struct {
		name string
		task ScheduledTask
		want string
	}{
		{name: "missing workflow", task: ScheduledTask{TaskKind: taskKindWorkflow}, want: "workflow is required"},
		{name: "mixed message", task: ScheduledTask{TaskKind: taskKindWorkflow, Message: "x", Workflow: validWorkflowDefinition()}, want: "does not allow message"},
		{name: "unknown node type", task: ScheduledTask{TaskKind: taskKindWorkflow, Workflow: &WorkflowDefinition{Nodes: []WorkflowNode{{ID: "start-node", Type: "start"}, {ID: "end-node", Type: "noop"}}, Edges: []WorkflowEdge{{FromNodeID: "start-node", ToNodeID: "end-node"}}}}, want: "unsupported workflow node type"},
		{name: "extra edge", task: ScheduledTask{TaskKind: taskKindWorkflow, Workflow: &WorkflowDefinition{Nodes: []WorkflowNode{{ID: "start-node", Type: "start"}, {ID: "end-node", Type: "end"}}, Edges: []WorkflowEdge{{FromNodeID: "start-node", ToNodeID: "end-node"}, {FromNodeID: "start-node", ToNodeID: "end-node"}}}}, want: "exactly 1 edge"},
		{name: "dangling edge", task: ScheduledTask{TaskKind: taskKindWorkflow, Workflow: &WorkflowDefinition{Nodes: []WorkflowNode{{ID: "start-node", Type: "start"}, {ID: "end-node", Type: "end"}}, Edges: []WorkflowEdge{{FromNodeID: "start-node", ToNodeID: "missing-node"}}}}, want: "unknown to_node_id"},
		{name: "self loop", task: ScheduledTask{TaskKind: taskKindWorkflow, Workflow: &WorkflowDefinition{Nodes: []WorkflowNode{{ID: "start-node", Type: "start"}, {ID: "end-node", Type: "end"}}, Edges: []WorkflowEdge{{FromNodeID: "start-node", ToNodeID: "start-node"}}}}, want: "self-loop"},
		{name: "wrong edge direction", task: ScheduledTask{TaskKind: taskKindWorkflow, Workflow: &WorkflowDefinition{Nodes: []WorkflowNode{{ID: "start-node", Type: "start"}, {ID: "end-node", Type: "end"}}, Edges: []WorkflowEdge{{FromNodeID: "end-node", ToNodeID: "start-node"}}}}, want: "start node must be the only edge source"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			err := validateTaskDefinition(&test.task)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}
}

func TestTaskWorkflowCreateUpdateListAndRunNow(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        validWorkflowDefinition(),
		IntervalSeconds: 60,
	}, "trace-workflow-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create workflow task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)
	if created.TaskKind != taskKindWorkflow || created.Workflow == nil {
		t.Fatalf("unexpected created payload: %#v", created)
	}

	updatedWorkflow := &WorkflowDefinition{
		Nodes: []WorkflowNode{{ID: "entry", Type: "start"}, {ID: "finish", Type: "end"}},
		Edges: []WorkflowEdge{{FromNodeID: "entry", ToNodeID: "finish"}},
	}
	updatedRaw, code, err := service.executeTaskUpdateAction(taskUpdateParams{
		ID:       created.ID,
		Workflow: updatedWorkflow,
	}, "trace-workflow-update")
	if err != nil || code != http.StatusOK {
		t.Fatalf("update workflow task: code=%d err=%v", code, err)
	}
	updated := updatedRaw.(taskPayload)
	if updated.Workflow == nil || updated.Workflow.Edges[0].FromNodeID != "entry" {
		t.Fatalf("unexpected updated payload: %#v", updated)
	}

	listRaw, code, err := service.executeTaskListAction(taskListScopeUser, "trace-workflow-list")
	if err != nil || code != http.StatusOK {
		t.Fatalf("list workflow tasks: code=%d err=%v", code, err)
	}
	items := listRaw.([]taskPayload)
	if len(items) != 1 || items[0].TaskKind != taskKindWorkflow {
		t.Fatalf("unexpected list payload: %#v", items)
	}

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if run.Run.ResponsePreview != "workflow completed: entry -> finish" {
		t.Fatalf("unexpected run preview: %#v", run.Run)
	}
}

func TestTaskWorkflowCreateRejectsMixedFields(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	_, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Message:         "should-fail",
		Workflow:        validWorkflowDefinition(),
		IntervalSeconds: 60,
	}, "trace-workflow-invalid")
	if err == nil {
		t.Fatal("expected mixed-field workflow create to fail")
	}
	if code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d want %d", code, http.StatusBadRequest)
	}
	if !strings.Contains(err.Error(), "does not allow message") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskWorkflowGetReturnsStoredDefinition(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	createdRaw, _, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind: taskKindWorkflow,
		Workflow: validWorkflowDefinition(),
		CronExpr: "*/5 * * * *",
	}, "trace-workflow-get")
	if err != nil {
		t.Fatalf("create workflow task: %v", err)
	}
	created := createdRaw.(taskPayload)
	gotRaw, code, err := service.executeTaskGetAction(taskIDParams{ID: created.ID}, "trace-workflow-get")
	if err != nil || code != http.StatusOK {
		t.Fatalf("get workflow task: code=%d err=%v", code, err)
	}
	got := gotRaw.(taskPayload)
	if got.Workflow == nil {
		t.Fatalf("expected workflow in get payload: %#v", got)
	}
	encoded, err := json.Marshal(got.Workflow)
	if err != nil {
		t.Fatalf("marshal workflow payload: %v", err)
	}
	if !strings.Contains(string(encoded), `"start-node"`) || !strings.Contains(string(encoded), `"end-node"`) {
		t.Fatalf("unexpected workflow payload: %s", string(encoded))
	}
}

func TestTaskWorkflowSystemScopeExcludesWorkflow(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	if _, _, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        validWorkflowDefinition(),
		IntervalSeconds: 60,
	}, "trace-workflow-system-scope"); err != nil {
		t.Fatalf("create workflow task: %v", err)
	}
	raw, code, err := service.executeTaskListAction(taskListScopeSystem, "trace-system-scope")
	if err != nil || code != http.StatusOK {
		t.Fatalf("list system tasks: code=%d err=%v", code, err)
	}
	if items := raw.([]taskPayload); len(items) != 0 {
		t.Fatalf("expected empty system scope, got %#v", items)
	}
}

func validWorkflowDefinition() *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{{FromNodeID: "start-node", ToNodeID: "end-node"}},
	}
}

func TestTaskWorkflowRunnerDoesNotNeedAgentExecution(t *testing.T) {
	result := executeWorkflowTask(ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: validWorkflowDefinition(),
	})
	if result.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected workflow execution result: %#v", result)
	}
}

func TestTaskWorkflowRunNowUsesExistingScheduler(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	if _, _, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        validWorkflowDefinition(),
		IntervalSeconds: 60,
	}, "trace-workflow-scheduler"); err != nil {
		t.Fatalf("create workflow task: %v", err)
	}
	runner, code, err := service.requireTaskMutationRunner()
	if err != nil || code != http.StatusOK {
		t.Fatalf("require runner: code=%d err=%v", code, err)
	}
	tasks, err := service.taskStore.ListTasks()
	if err != nil || len(tasks) != 1 {
		t.Fatalf("list tasks: tasks=%d err=%v", len(tasks), err)
	}
	run, err := runner.RunNow(taskIDParams{ID: tasks[0].ID}, "trace-workflow-runner")
	if err != nil {
		t.Fatalf("run now through runner: %v", err)
	}
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected runner status: %#v", run.Run)
	}
}
