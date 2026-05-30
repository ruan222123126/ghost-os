package orchestration

import (
	"context"
	"encoding/json"
	bridgeconfig "ghost-os/bridge/config"
	"net/http"
	"strings"
	"testing"
)

func TestValidateWorkflowTaskRuntimeAllowsAllToolsWhenAllowlistEmpty(t *testing.T) {
	definition := workflowWithToolNode("web_search")

	err := validateWorkflowTaskRuntime(definition, bridgeconfig.TaskConfig{})
	if err != nil {
		t.Fatalf("expected empty allowlist to allow all tools, got %v", err)
	}
}

func TestValidateWorkflowTaskRuntimeRejectsToolOutsideExplicitAllowlist(t *testing.T) {
	definition := workflowWithToolNode("web_search")
	cfg := bridgeconfig.TaskConfig{WorkflowToolAllowlist: []string{"script_exec"}}

	err := validateWorkflowTaskRuntime(definition, cfg)
	if err == nil {
		t.Fatal("expected workflow tool validation to fail")
	}
	if !strings.Contains(err.Error(), `workflow tool "web_search" is not allowed`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowTaskCreateRejectsInvalidAgentRuntimeOverrides(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	configureRuntimeOverrideProviders(t, service)

	workflow := &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{
				ID:   "agent-node",
				Type: workflowNodeTypeAgent,
				Agent: &WorkflowAgentNode{
					Message: "run agent",
					RuntimeOverrides: &TaskRuntimeOverrides{
						ProviderName: "openai-main",
					},
				},
			},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "agent-node"},
			{FromNodeID: "agent-node", ToNodeID: "end-node"},
		},
	}

	_, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflow,
		IntervalSeconds: 60,
	}, "trace-workflow-agent-runtime-invalid")
	if err == nil {
		t.Fatal("expected workflow agent runtime validation to fail")
	}
	if code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d want %d", code, http.StatusBadRequest)
	}
	if !strings.Contains(err.Error(), "provider_name requires model") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateTaskDefinitionWorkflowAcceptsLinearNodeChain(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: &WorkflowDefinition{
			Nodes: []WorkflowNode{
				{ID: "start-node", Type: workflowNodeTypeStart},
				{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: "script_exec"}},
				{ID: "llm-node", Type: workflowNodeTypeLLM, LLM: &WorkflowLLMNode{Prompt: "summarize"}},
				{ID: "agent-node", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "reply"}},
				{ID: "end-node", Type: workflowNodeTypeEnd},
			},
			Edges: []WorkflowEdge{
				{FromNodeID: "start-node", ToNodeID: "tool-node"},
				{FromNodeID: "tool-node", ToNodeID: "llm-node"},
				{FromNodeID: "llm-node", ToNodeID: "agent-node"},
				{FromNodeID: "agent-node", ToNodeID: "end-node"},
			},
		},
	}
	if err := validateTaskDefinition(&task); err != nil {
		t.Fatalf("validate workflow chain: %v", err)
	}
}

func TestValidateTaskDefinitionWorkflowAcceptsStartParallelBranches(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: &WorkflowDefinition{
			Nodes: []WorkflowNode{
				{ID: "start-node", Type: workflowNodeTypeStart},
				{ID: "agent-a", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "a"}},
				{ID: "agent-b", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "b"}},
				{ID: "end-node", Type: workflowNodeTypeEnd},
			},
			Edges: []WorkflowEdge{
				{FromNodeID: "start-node", ToNodeID: "agent-a"},
				{FromNodeID: "start-node", ToNodeID: "agent-b"},
				{FromNodeID: "agent-a", ToNodeID: "end-node"},
				{FromNodeID: "agent-b", ToNodeID: "end-node"},
			},
		},
	}
	if err := validateTaskDefinition(&task); err != nil {
		t.Fatalf("validate workflow start parallel branches: %v", err)
	}
}

func TestValidateTaskDefinitionWorkflowRejectsNodePayloadMismatch(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: &WorkflowDefinition{
			Nodes: []WorkflowNode{
				{ID: "start-node", Type: workflowNodeTypeStart},
				{ID: "tool-node", Type: workflowNodeTypeTool, LLM: &WorkflowLLMNode{Prompt: "bad"}},
				{ID: "end-node", Type: workflowNodeTypeEnd},
			},
			Edges: []WorkflowEdge{
				{FromNodeID: "start-node", ToNodeID: "tool-node"},
				{FromNodeID: "tool-node", ToNodeID: "end-node"},
			},
		},
	}
	err := validateTaskDefinition(&task)
	if err == nil || !strings.Contains(err.Error(), "payload does not match type") {
		t.Fatalf("expected payload mismatch, got %v", err)
	}
}

func TestValidateTaskDefinitionWorkflowRejectsBranchingFromNonControlNode(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: &WorkflowDefinition{
			Nodes: []WorkflowNode{
				{ID: "start-node", Type: workflowNodeTypeStart},
				{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: "script_exec"}},
				{ID: "llm-node", Type: workflowNodeTypeLLM, LLM: &WorkflowLLMNode{Prompt: "branch"}},
				{ID: "end-node", Type: workflowNodeTypeEnd},
			},
			Edges: []WorkflowEdge{
				{FromNodeID: "start-node", ToNodeID: "tool-node"},
				{FromNodeID: "tool-node", ToNodeID: "llm-node"},
				{FromNodeID: "tool-node", ToNodeID: "end-node"},
				{FromNodeID: "llm-node", ToNodeID: "end-node"},
			},
		},
	}
	err := validateTaskDefinition(&task)
	if err == nil || !strings.Contains(err.Error(), `must have in>=1 and out=1`) {
		t.Fatalf("expected non-control branching failure, got %v", err)
	}
}

func TestValidateTaskDefinitionWorkflowAcceptsIfAndLoopNodes(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: &WorkflowDefinition{
			Nodes: []WorkflowNode{
				{ID: "start-node", Type: workflowNodeTypeStart},
				{
					ID:   "if-node",
					Type: workflowNodeTypeIf,
					If: &WorkflowIfNode{
						Operator:    workflowIfOperatorIsEmpty,
						TrueNodeID:  "loop-node",
						FalseNodeID: "end-node",
					},
				},
				{
					ID:   "loop-node",
					Type: workflowNodeTypeLoop,
					Loop: &WorkflowLoopNode{
						MaxIterations: 2,
						BodyNodeID:    "tool-node",
						ExitNodeID:    "end-node",
					},
				},
				{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: "script_exec"}},
				{ID: "end-node", Type: workflowNodeTypeEnd},
			},
			Edges: []WorkflowEdge{
				{FromNodeID: "start-node", ToNodeID: "if-node"},
				{FromNodeID: "if-node", ToNodeID: "loop-node"},
				{FromNodeID: "if-node", ToNodeID: "end-node"},
				{FromNodeID: "loop-node", ToNodeID: "tool-node"},
				{FromNodeID: "loop-node", ToNodeID: "end-node"},
				{FromNodeID: "tool-node", ToNodeID: "loop-node"},
			},
		},
	}
	if err := validateTaskDefinition(&task); err != nil {
		t.Fatalf("validate workflow with if/loop: %v", err)
	}
}

func TestValidateTaskDefinitionWorkflowRejectsLoopWithoutCycle(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: &WorkflowDefinition{
			Nodes: []WorkflowNode{
				{ID: "start-node", Type: workflowNodeTypeStart},
				{
					ID:   "loop-node",
					Type: workflowNodeTypeLoop,
					Loop: &WorkflowLoopNode{
						MaxIterations: 3,
						BodyNodeID:    "tool-node",
						ExitNodeID:    "end-node",
					},
				},
				{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: "script_exec"}},
				{ID: "end-node", Type: workflowNodeTypeEnd},
			},
			Edges: []WorkflowEdge{
				{FromNodeID: "start-node", ToNodeID: "loop-node"},
				{FromNodeID: "loop-node", ToNodeID: "tool-node"},
				{FromNodeID: "loop-node", ToNodeID: "end-node"},
				{FromNodeID: "tool-node", ToNodeID: "end-node"},
			},
		},
	}
	err := validateTaskDefinition(&task)
	if err == nil || !strings.Contains(err.Error(), "body path must return to the loop node") {
		t.Fatalf("expected loop cycle validation failure, got %v", err)
	}
}

func TestTaskWorkflowCreateRejectsToolOutsideAllowlist(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	_, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflowWithToolNode("web_search"),
		IntervalSeconds: 60,
	}, "trace-workflow-tool-reject")
	if err == nil {
		t.Fatal("expected workflow tool allowlist validation to fail")
	}
	if code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d want %d", code, http.StatusBadRequest)
	}
	if !strings.Contains(err.Error(), `workflow tool "web_search" is not allowed`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateTaskDefinitionWorkflowAcceptsStartInputs(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: workflowWithStartInputs([]WorkflowInputVariable{
			{
				Name:        "topic",
				Type:        workflowInputTypeString,
				Required:    true,
				Default:     []byte(`"release notes"`),
				Description: "summary topic",
			},
			{
				Name:    "threshold",
				Type:    workflowInputTypeNumber,
				Default: []byte(`0.7`),
			},
			{
				Name:    "flags",
				Type:    workflowInputTypeObject,
				Default: []byte(`{"urgent":true}`),
			},
			{
				Name:    "tags",
				Type:    workflowInputTypeArray,
				Default: []byte(`["ops","weekly"]`),
			},
		}),
	}
	if err := validateTaskDefinition(&task); err != nil {
		t.Fatalf("validate workflow with start inputs: %v", err)
	}
}

func TestValidateTaskDefinitionWorkflowRejectsInvalidStartInputs(t *testing.T) {
	tests := []struct {
		name   string
		inputs []WorkflowInputVariable
		want   string
	}{
		{
			name: "invalid variable name",
			inputs: []WorkflowInputVariable{
				{Name: "1topic", Type: workflowInputTypeString},
			},
			want: "must match ^[A-Za-z_][A-Za-z0-9_]*$",
		},
		{
			name: "duplicate variable names",
			inputs: []WorkflowInputVariable{
				{Name: "topic", Type: workflowInputTypeString},
				{Name: "topic", Type: workflowInputTypeNumber},
			},
			want: "duplicate name",
		},
		{
			name: "default type mismatch",
			inputs: []WorkflowInputVariable{
				{Name: "retry_count", Type: workflowInputTypeNumber, Default: []byte(`"3"`)},
			},
			want: "must match declared type",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			task := ScheduledTask{
				TaskKind: taskKindWorkflow,
				Workflow: workflowWithStartInputs(test.inputs),
			}
			err := validateTaskDefinition(&task)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}
}

func workflowWithStartInputs(inputs []WorkflowInputVariable) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{
				ID:    "start-node",
				Type:  workflowNodeTypeStart,
				Start: &WorkflowStartNode{Inputs: inputs},
			},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{{FromNodeID: "start-node", ToNodeID: "end-node"}},
	}
}

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
		{name: "extra edge", task: ScheduledTask{TaskKind: taskKindWorkflow, Workflow: &WorkflowDefinition{Nodes: []WorkflowNode{{ID: "start-node", Type: "start"}, {ID: "end-node", Type: "end"}}, Edges: []WorkflowEdge{{FromNodeID: "start-node", ToNodeID: "end-node"}, {FromNodeID: "start-node", ToNodeID: "end-node"}}}}, want: "duplicate workflow edge"},
		{name: "dangling edge", task: ScheduledTask{TaskKind: taskKindWorkflow, Workflow: &WorkflowDefinition{Nodes: []WorkflowNode{{ID: "start-node", Type: "start"}, {ID: "end-node", Type: "end"}}, Edges: []WorkflowEdge{{FromNodeID: "start-node", ToNodeID: "missing-node"}}}}, want: "unknown to_node_id"},
		{name: "self loop", task: ScheduledTask{TaskKind: taskKindWorkflow, Workflow: &WorkflowDefinition{Nodes: []WorkflowNode{{ID: "start-node", Type: "start"}, {ID: "end-node", Type: "end"}}, Edges: []WorkflowEdge{{FromNodeID: "start-node", ToNodeID: "start-node"}}}}, want: "self-loop"},
		{name: "wrong edge direction", task: ScheduledTask{TaskKind: taskKindWorkflow, Workflow: &WorkflowDefinition{Nodes: []WorkflowNode{{ID: "start-node", Type: "start"}, {ID: "end-node", Type: "end"}}, Edges: []WorkflowEdge{{FromNodeID: "end-node", ToNodeID: "start-node"}}}}, want: "must have in"},
		{name: "start payload on non-start node", task: ScheduledTask{TaskKind: taskKindWorkflow, Workflow: &WorkflowDefinition{Nodes: []WorkflowNode{{ID: "start-node", Type: "start"}, {ID: "tool-node", Type: "tool", Start: &WorkflowStartNode{}, Tool: &WorkflowToolNode{ToolName: "script_exec"}}, {ID: "end-node", Type: "end"}}, Edges: []WorkflowEdge{{FromNodeID: "start-node", ToNodeID: "tool-node"}, {FromNodeID: "tool-node", ToNodeID: "end-node"}}}}, want: "payload does not match type"},
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

func TestTaskWorkflowRunnerDoesNotNeedAgentExecution(t *testing.T) {
	result := taskExecutorAdapter{}.executeWorkflowTask(context.Background(), ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: validWorkflowDefinition(),
	}, "trace-workflow-direct")
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
	taskStore, code, err := service.requireTaskStore()
	if err != nil || code != http.StatusOK {
		t.Fatalf("require task store: code=%d err=%v", code, err)
	}
	tasks, err := taskStore.ListTasks()
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
