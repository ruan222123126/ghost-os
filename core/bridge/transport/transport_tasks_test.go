package transport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type taskWorkflowDefinition struct {
	Nodes []taskWorkflowNode `json:"nodes"`
	Edges []taskWorkflowEdge `json:"edges"`
}

type taskWorkflowNode struct {
	ID    string                 `json:"id"`
	Type  string                 `json:"type"`
	Tool  *taskWorkflowToolNode  `json:"tool,omitempty"`
	LLM   *taskWorkflowLLMNode   `json:"llm,omitempty"`
	Agent *taskWorkflowAgentNode `json:"agent,omitempty"`
}

type taskWorkflowToolNode struct {
	ToolName  string         `json:"tool_name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type taskWorkflowLLMNode struct {
	Prompt       string `json:"prompt"`
	SystemPrompt string `json:"system_prompt,omitempty"`
}

type taskWorkflowAgentNode struct {
	Message string `json:"message"`
}

type taskWorkflowEdge struct {
	FromNodeID string `json:"from_node_id"`
	ToNodeID   string `json:"to_node_id"`
}

type taskResponsePayload struct {
	ID           string                  `json:"id"`
	TaskKind     string                  `json:"task_kind"`
	Workflow     *taskWorkflowDefinition `json:"workflow,omitempty"`
	IntervalSecs int                     `json:"interval_seconds,omitempty"`
	CronExpr     string                  `json:"cron_expr,omitempty"`
}

func TestHandleTasksCreateGetAndUpdateWorkflow(t *testing.T) {
	handler := newTestHandler(t, nil)

	create := serveRequest(handler, http.MethodPost, "/api/tasks", `{
		"task_kind":"workflow",
		"workflow":{
			"nodes":[
				{"id":"start-node","type":"start"},
				{"id":"end-node","type":"end"}
			],
			"edges":[
				{"from_node_id":"start-node","to_node_id":"end-node"}
			]
		},
		"interval_seconds":60
	}`, nil)
	if create.Code != http.StatusCreated {
		t.Fatalf("unexpected create status: got %d want %d", create.Code, http.StatusCreated)
	}
	created := decodeTaskResponsePayload(t, create)
	if created.TaskKind != "workflow" || created.Workflow == nil {
		t.Fatalf("unexpected created payload: %#v", created)
	}

	get := serveRequest(handler, http.MethodGet, "/api/tasks/"+created.ID, "", nil)
	if get.Code != http.StatusOK {
		t.Fatalf("unexpected get status: got %d want %d", get.Code, http.StatusOK)
	}
	got := decodeTaskResponsePayload(t, get)
	if got.Workflow == nil || got.Workflow.Edges[0].FromNodeID != "start-node" {
		t.Fatalf("unexpected get payload: %#v", got)
	}

	update := serveRequest(handler, http.MethodPatch, "/api/tasks/"+created.ID, `{
		"workflow":{
			"nodes":[
				{"id":"entry","type":"start"},
				{"id":"llm-step","type":"llm","llm":{"prompt":"Summarize","system_prompt":"Be concise"}},
				{"id":"finish","type":"end"}
			],
			"edges":[
				{"from_node_id":"entry","to_node_id":"llm-step"},
				{"from_node_id":"llm-step","to_node_id":"finish"}
			]
		},
		"cron_expr":"*/5 * * * *"
	}`, nil)
	if update.Code != http.StatusOK {
		t.Fatalf("unexpected update status: got %d want %d", update.Code, http.StatusOK)
	}
	updated := decodeTaskResponsePayload(t, update)
	if updated.Workflow == nil || updated.Workflow.Edges[0].FromNodeID != "entry" {
		t.Fatalf("unexpected updated payload: %#v", updated)
	}
	if updated.Workflow.Nodes[1].LLM == nil || updated.Workflow.Nodes[1].LLM.Prompt != "Summarize" {
		t.Fatalf("unexpected updated workflow nodes: %#v", updated.Workflow.Nodes)
	}
	if updated.CronExpr != "*/5 * * * *" {
		t.Fatalf("unexpected updated cron expr: %#v", updated)
	}
}

func TestHandleTasksRejectsInvalidWorkflowRequest(t *testing.T) {
	handler := newTestHandler(t, nil)
	recorder := serveRequest(handler, http.MethodPost, "/api/tasks", `{
		"task_kind":"workflow",
		"message":"bad",
		"workflow":{
			"nodes":[
				{"id":"start-node","type":"start"},
				{"id":"end-node","type":"end"}
			],
			"edges":[
				{"from_node_id":"start-node","to_node_id":"end-node"}
			]
		},
		"interval_seconds":60
	}`, nil)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusBadRequest)
	}
	body := decodeResponseBody(t, recorder)
	if body.Error == "" {
		t.Fatal("expected validation error")
	}
}

func TestHandleSystemTasksExcludesWorkflow(t *testing.T) {
	handler := newTestHandler(t, nil)

	create := serveRequest(handler, http.MethodPost, "/api/tasks", `{
		"task_kind":"workflow",
		"workflow":{
			"nodes":[
				{"id":"start-node","type":"start"},
				{"id":"end-node","type":"end"}
			],
			"edges":[
				{"from_node_id":"start-node","to_node_id":"end-node"}
			]
		},
		"interval_seconds":60
	}`, nil)
	if create.Code != http.StatusCreated {
		t.Fatalf("create workflow task: status=%d", create.Code)
	}

	systemList := serveRequest(handler, http.MethodGet, "/api/system/tasks", "", nil)
	if systemList.Code != http.StatusOK {
		t.Fatalf("unexpected system list status: got %d want %d", systemList.Code, http.StatusOK)
	}
	body := decodeResponseBody(t, systemList)
	data, err := json.Marshal(body.Payload)
	if err != nil {
		t.Fatalf("marshal system payload: %v", err)
	}
	var items []taskResponsePayload
	if err := json.Unmarshal(data, &items); err != nil {
		t.Fatalf("decode system task payloads: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected workflow to be excluded from system scope, got %#v", items)
	}
}

func decodeTaskResponsePayload(t *testing.T, recorder *httptest.ResponseRecorder) taskResponsePayload {
	t.Helper()
	body := decodeResponseBody(t, recorder)
	data, err := json.Marshal(body.Payload)
	if err != nil {
		t.Fatalf("marshal task payload: %v", err)
	}
	var payload taskResponsePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("decode task payload: %v", err)
	}
	return payload
}
