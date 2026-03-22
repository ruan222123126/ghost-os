package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
)

func TestTaskWorkflowRunNowExecutesToolLLMAndAgentNodes(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	tool := &workflowTestTool{name: "script_exec", output: `{"status":"ok"}`}
	completer := &workflowTestCompleter{
		response: &llm.CompletionResponse{Message: llm.Message{Role: llm.RoleAssistant, Text: "llm summary"}},
	}
	registry := tools.NewRegistry()
	registry.Register(tool)
	service.runtimeFactory = proTestRuntimeFactory{
		deps: NewRuntimeDependencies(Config{Task: TaskConfig{WorkflowToolAllowlist: []string{"script_exec"}}}, completer, registry, "", nil),
	}
	agentRunner := &workflowTestRunner{message: "agent done", sessionID: "workflow-session"}
	service.agentRunner = agentRunner

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflowWithToolLLMAndAgentNodes("script_exec"),
		IntervalSeconds: 60,
	}, "trace-workflow-node-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create workflow task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-node-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if run.Run.ResponsePreview != "workflow completed at agent-node: agent done" {
		t.Fatalf("unexpected run preview: %#v", run.Run)
	}
	if run.Run.SessionIDOutput != "workflow-session" {
		t.Fatalf("unexpected session output: %#v", run.Run)
	}
	if len(tool.calls) != 1 || tool.calls[0]["command"] != "pwd" {
		t.Fatalf("unexpected tool calls: %#v", tool.calls)
	}
	if len(completer.requests) != 1 {
		t.Fatalf("unexpected llm requests: %#v", completer.requests)
	}
	request := completer.requests[0]
	if len(request.Tools) != 0 || !request.ConversationState.IsZero() {
		t.Fatalf("expected direct llm request without tools/state, got %#v", request)
	}
	if len(request.Messages) != 2 || request.Messages[0].Role != llm.RoleSystem || request.Messages[1].Role != llm.RoleUser {
		t.Fatalf("unexpected llm messages: %#v", request.Messages)
	}
	if len(agentRunner.calls) != 1 || agentRunner.calls[0].sessionID != "" || agentRunner.calls[0].message != "fixed agent message" {
		t.Fatalf("unexpected agent runner calls: %#v", agentRunner.calls)
	}
}

func TestTaskWorkflowRunNowStopsOnAgentAwaitingHuman(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	service.agentRunner = &workflowTestRunner{
		sessionID: "await-session",
		err: &agent.ErrAwaitingHuman{
			QuestionID: "q-1",
			Prompt:     "Need approval",
		},
	}

	createdRaw, _, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflowWithAgentNode(),
		IntervalSeconds: 60,
	}, "trace-workflow-await-create")
	if err != nil {
		t.Fatalf("create workflow task: %v", err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-await-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusAwaitingHuman {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if run.Run.SessionIDOutput != "await-session" {
		t.Fatalf("unexpected session output: %#v", run.Run)
	}
	if !strings.Contains(run.Run.ResponsePreview, "Need approval") {
		t.Fatalf("unexpected awaiting preview: %#v", run.Run)
	}
}

func TestTaskWorkflowRunNowFailsWhenAllowlistChanges(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	registry := tools.NewRegistry()
	registry.Register(&workflowTestTool{name: "script_exec", output: `{"status":"ok"}`})
	service.runtimeFactory = proTestRuntimeFactory{
		deps: NewRuntimeDependencies(Config{Task: TaskConfig{WorkflowToolAllowlist: []string{"script_exec"}}}, &workflowTestCompleter{}, registry, "", nil),
	}

	createdRaw, _, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflowWithToolNode("script_exec"),
		IntervalSeconds: 60,
	}, "trace-workflow-config-shift-create")
	if err != nil {
		t.Fatalf("create workflow task: %v", err)
	}
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "")
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-config-shift-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusError {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if !strings.Contains(run.Run.Error, `workflow tool "script_exec" is not allowed`) {
		t.Fatalf("unexpected run error: %#v", run.Run)
	}
}

type workflowTestTool struct {
	name   string
	output string
	calls  []map[string]any
}

func (t *workflowTestTool) Name() string                { return t.name }
func (t *workflowTestTool) Description() string         { return "workflow test tool" }
func (t *workflowTestTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }

func (t *workflowTestTool) Execute(_ context.Context, args json.RawMessage, _ string) (string, error) {
	decoded := make(map[string]any)
	if err := json.Unmarshal(args, &decoded); err != nil {
		return "", err
	}
	t.calls = append(t.calls, decoded)
	return t.output, nil
}

type workflowTestCompleter struct {
	response *llm.CompletionResponse
	err      error
	requests []llm.CompletionRequest
}

func (c *workflowTestCompleter) Complete(_ context.Context, request llm.CompletionRequest) (*llm.CompletionResponse, error) {
	c.requests = append(c.requests, request)
	if c.err != nil {
		return nil, c.err
	}
	if c.response == nil {
		return nil, errors.New("unexpected llm completion")
	}
	return c.response, nil
}

type workflowRunnerCall struct {
	message   string
	sessionID string
}

type workflowTestRunner struct {
	message   string
	sessionID string
	err       error
	calls     []workflowRunnerCall
}

func (r *workflowTestRunner) RunTurn(_ context.Context, message string, sessionID string, _ string) (string, string, error) {
	r.calls = append(r.calls, workflowRunnerCall{message: message, sessionID: sessionID})
	return r.message, r.sessionID, r.err
}

func (r *workflowTestRunner) RunTurnStream(
	ctx context.Context,
	message string,
	sessionID string,
	traceID string,
	_ streaming.Sink,
) (string, string, error) {
	return r.RunTurn(ctx, message, sessionID, traceID)
}

func workflowWithToolNode(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: toolName, Arguments: map[string]any{"command": "pwd"}}},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "tool-node"},
			{FromNodeID: "tool-node", ToNodeID: "end-node"},
		},
	}
}

func workflowWithAgentNode() *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{ID: "agent-node", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "fixed agent message"}},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "agent-node"},
			{FromNodeID: "agent-node", ToNodeID: "end-node"},
		},
	}
}

func workflowWithToolLLMAndAgentNodes(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: toolName, Arguments: map[string]any{"command": "pwd"}}},
			{ID: "llm-node", Type: workflowNodeTypeLLM, LLM: &WorkflowLLMNode{Prompt: "Summarize the previous result", SystemPrompt: "Be concise"}},
			{ID: "agent-node", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "fixed agent message"}},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "tool-node"},
			{FromNodeID: "tool-node", ToNodeID: "llm-node"},
			{FromNodeID: "llm-node", ToNodeID: "agent-node"},
			{FromNodeID: "agent-node", ToNodeID: "end-node"},
		},
	}
}
