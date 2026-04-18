package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
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
		deps: NewRuntimeDependencies(bridgeconfig.Config{Task: bridgeconfig.TaskConfig{WorkflowToolAllowlist: []string{"script_exec"}}}, completer, registry, "", nil),
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

func TestTaskWorkflowRunNowRoutesIfNodeByToolOutput(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	tool := &workflowTestTool{name: "script_exec", output: "status=ok"}
	registry := tools.NewRegistry()
	registry.Register(tool)
	service.runtimeFactory = proTestRuntimeFactory{
		deps: NewRuntimeDependencies(bridgeconfig.Config{Task: bridgeconfig.TaskConfig{WorkflowToolAllowlist: []string{"script_exec"}}}, &workflowTestCompleter{}, registry, "", nil),
	}
	agentRunner := &workflowTestRunner{message: "if branch done", sessionID: "if-session"}
	service.agentRunner = agentRunner

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflowWithIfNode("script_exec"),
		IntervalSeconds: 60,
	}, "trace-workflow-if-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create workflow task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-if-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if run.Run.SessionIDOutput != "if-session" {
		t.Fatalf("unexpected session output: %#v", run.Run)
	}
	if len(agentRunner.calls) != 1 || agentRunner.calls[0].message != "true branch message" {
		t.Fatalf("unexpected agent branch: %#v", agentRunner.calls)
	}
}

func TestTaskWorkflowRunNowResolvesTemplateInIfValue(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	tool := &workflowTestTool{name: "script_exec", output: "status=ok"}
	registry := tools.NewRegistry()
	registry.Register(tool)
	service.runtimeFactory = proTestRuntimeFactory{
		deps: NewRuntimeDependencies(
			bridgeconfig.Config{Task: bridgeconfig.TaskConfig{WorkflowToolAllowlist: []string{"script_exec"}}},
			&workflowTestCompleter{},
			registry,
			"",
			nil,
		),
	}
	agentRunner := &workflowTestRunner{message: "if template branch done", sessionID: "if-template-session"}
	service.agentRunner = agentRunner

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflowWithTemplatedIfNode("script_exec"),
		IntervalSeconds: 60,
	}, "trace-workflow-if-template-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create workflow task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-if-template-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if run.Run.SessionIDOutput != "if-template-session" {
		t.Fatalf("unexpected session output: %#v", run.Run)
	}
	if len(agentRunner.calls) != 1 || agentRunner.calls[0].message != "true branch message" {
		t.Fatalf("unexpected agent branch: %#v", agentRunner.calls)
	}
}

func TestTaskWorkflowRunNowExecutesLoopBodyByMaxIterations(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	tool := &workflowTestTool{name: "script_exec", output: "loop body"}
	registry := tools.NewRegistry()
	registry.Register(tool)
	service.runtimeFactory = proTestRuntimeFactory{
		deps: NewRuntimeDependencies(bridgeconfig.Config{Task: bridgeconfig.TaskConfig{WorkflowToolAllowlist: []string{"script_exec"}}}, &workflowTestCompleter{}, registry, "", nil),
	}

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflowWithLoopNode("script_exec", 3),
		IntervalSeconds: 60,
	}, "trace-workflow-loop-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create workflow task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-loop-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if len(tool.calls) != 3 {
		t.Fatalf("unexpected loop execution count: got %d want 3", len(tool.calls))
	}
}

func TestTaskWorkflowRunNowExecutesStartBranchesInParallel(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	tool := newWorkflowParallelProbeTool("script_exec")
	registry := tools.NewRegistry()
	registry.Register(tool)
	service.runtimeFactory = proTestRuntimeFactory{
		deps: NewRuntimeDependencies(
			bridgeconfig.Config{Task: bridgeconfig.TaskConfig{WorkflowToolAllowlist: []string{"script_exec"}}},
			&workflowTestCompleter{},
			registry,
			"",
			nil,
		),
	}

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflowWithParallelStartToolBranches("script_exec"),
		IntervalSeconds: 60,
	}, "trace-workflow-parallel-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create workflow task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	type runNowOutcome struct {
		raw  any
		code int
		err  error
	}
	outcomeCh := make(chan runNowOutcome, 1)
	go func() {
		raw, runCode, runErr := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-parallel-run")
		outcomeCh <- runNowOutcome{raw: raw, code: runCode, err: runErr}
	}()

	if !tool.waitStarted(2, 2*time.Second) {
		tool.release()
		t.Fatal("expected both start branches to run before release")
	}
	tool.release()
	outcome := <-outcomeCh
	if outcome.err != nil || outcome.code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", outcome.code, outcome.err)
	}
	run := outcome.raw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if !strings.Contains(run.Run.ResponsePreview, "workflow completed in parallel") {
		t.Fatalf("unexpected run preview: %#v", run.Run)
	}
	if tool.maxConcurrency() < 2 {
		t.Fatalf("expected concurrent execution, got max=%d", tool.maxConcurrency())
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

func TestTaskWorkflowRunNowAllowsAllToolsWhenAllowlistCleared(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	registry := tools.NewRegistry()
	registry.Register(&workflowTestTool{name: "script_exec", output: `{"status":"ok"}`})
	service.runtimeFactory = proTestRuntimeFactory{
		deps: NewRuntimeDependencies(bridgeconfig.Config{Task: bridgeconfig.TaskConfig{WorkflowToolAllowlist: []string{"script_exec"}}}, &workflowTestCompleter{}, registry, "", nil),
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
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if strings.TrimSpace(run.Run.Error) != "" {
		t.Fatalf("unexpected run error: %#v", run.Run)
	}
}

func TestTaskWorkflowRunNowResolvesTemplatedToolArguments(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	tool := &workflowTestTool{
		name:    "script_exec",
		outputs: []string{`{"cwd":"/workspace","ok":true}`},
		output:  `{"status":"done"}`,
	}
	registry := tools.NewRegistry()
	registry.Register(tool)
	service.runtimeFactory = proTestRuntimeFactory{
		deps: NewRuntimeDependencies(
			bridgeconfig.Config{Task: bridgeconfig.TaskConfig{WorkflowToolAllowlist: []string{"script_exec"}}},
			&workflowTestCompleter{},
			registry,
			"",
			nil,
		),
	}

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflowWithTemplatedToolChain("script_exec"),
		IntervalSeconds: 60,
	}, "trace-workflow-template-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create workflow task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-template-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if len(tool.calls) != 2 {
		t.Fatalf("unexpected tool call count: %#v", tool.calls)
	}
	if tool.calls[0]["command"] != "pwd" {
		t.Fatalf("unexpected first tool args: %#v", tool.calls[0])
	}
	if tool.calls[1]["command"] != "echo /workspace" {
		t.Fatalf("unexpected second tool command: %#v", tool.calls[1])
	}
	if literal, ok := tool.calls[1]["ok"].(bool); !ok || !literal {
		t.Fatalf("unexpected second tool bool arg: %#v", tool.calls[1])
	}
	if tool.calls[1]["text"] != "run pwd" {
		t.Fatalf("unexpected second tool text arg: %#v", tool.calls[1])
	}
}

func TestTaskWorkflowRunNowFailsOnUndefinedTemplateVariable(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	registry := tools.NewRegistry()
	registry.Register(&workflowTestTool{name: "script_exec", output: `{"status":"ok"}`})
	service.runtimeFactory = proTestRuntimeFactory{
		deps: NewRuntimeDependencies(
			bridgeconfig.Config{Task: bridgeconfig.TaskConfig{WorkflowToolAllowlist: []string{"script_exec"}}},
			&workflowTestCompleter{},
			registry,
			"",
			nil,
		),
	}

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflowWithUndefinedTemplate("script_exec"),
		IntervalSeconds: 60,
	}, "trace-workflow-template-missing-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create workflow task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-template-missing-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusError {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if !strings.Contains(run.Run.Error, `workflow template variable "outputs.missing.status" is not defined`) {
		t.Fatalf("unexpected run error: %#v", run.Run)
	}
}

type workflowTestTool struct {
	name    string
	output  string
	outputs []string
	calls   []map[string]any
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
	callIndex := len(t.calls) - 1
	if callIndex >= 0 && callIndex < len(t.outputs) {
		return t.outputs[callIndex], nil
	}
	return t.output, nil
}

type workflowParallelProbeTool struct {
	name        string
	startedCh   chan struct{}
	releaseOnce sync.Once
	releaseCh   chan struct{}
	mu          sync.Mutex
	active      int
	maxActive   int
}

func newWorkflowParallelProbeTool(name string) *workflowParallelProbeTool {
	return &workflowParallelProbeTool{
		name:      name,
		startedCh: make(chan struct{}, 8),
		releaseCh: make(chan struct{}),
	}
}

func (t *workflowParallelProbeTool) Name() string        { return t.name }
func (t *workflowParallelProbeTool) Description() string { return "workflow parallel probe tool" }
func (t *workflowParallelProbeTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}

func (t *workflowParallelProbeTool) Execute(_ context.Context, _ json.RawMessage, _ string) (string, error) {
	t.mu.Lock()
	t.active++
	if t.active > t.maxActive {
		t.maxActive = t.active
	}
	t.mu.Unlock()
	t.startedCh <- struct{}{}
	<-t.releaseCh
	t.mu.Lock()
	t.active--
	t.mu.Unlock()
	return `{"status":"ok"}`, nil
}

func (t *workflowParallelProbeTool) waitStarted(count int, timeout time.Duration) bool {
	for i := 0; i < count; i++ {
		select {
		case <-t.startedCh:
		case <-time.After(timeout):
			return false
		}
	}
	return true
}

func (t *workflowParallelProbeTool) release() {
	t.releaseOnce.Do(func() {
		close(t.releaseCh)
	})
}

func (t *workflowParallelProbeTool) maxConcurrency() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.maxActive
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

func workflowWithParallelStartToolBranches(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{ID: "tool-a", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: toolName, Arguments: map[string]any{"command": "echo a"}}},
			{ID: "tool-b", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: toolName, Arguments: map[string]any{"command": "echo b"}}},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "tool-a"},
			{FromNodeID: "start-node", ToNodeID: "tool-b"},
			{FromNodeID: "tool-a", ToNodeID: "end-node"},
			{FromNodeID: "tool-b", ToNodeID: "end-node"},
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

func workflowWithIfNode(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: toolName, Arguments: map[string]any{"command": "pwd"}}},
			{
				ID:   "if-node",
				Type: workflowNodeTypeIf,
				If: &WorkflowIfNode{
					SourceNodeID: "tool-node",
					Operator:     workflowIfOperatorContains,
					Value:        "ok",
					TrueNodeID:   "agent-true",
					FalseNodeID:  "agent-false",
				},
			},
			{ID: "agent-true", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "true branch message"}},
			{ID: "agent-false", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "false branch message"}},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "tool-node"},
			{FromNodeID: "tool-node", ToNodeID: "if-node"},
			{FromNodeID: "if-node", ToNodeID: "agent-true"},
			{FromNodeID: "if-node", ToNodeID: "agent-false"},
			{FromNodeID: "agent-true", ToNodeID: "end-node"},
			{FromNodeID: "agent-false", ToNodeID: "end-node"},
		},
	}
}

func workflowWithTemplatedIfNode(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{
				ID:   "start-node",
				Type: workflowNodeTypeStart,
				Start: &WorkflowStartNode{Inputs: []WorkflowInputVariable{
					{Name: "token", Type: workflowInputTypeString, Default: []byte(`"ok"`)},
				}},
			},
			{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: toolName, Arguments: map[string]any{"command": "pwd"}}},
			{
				ID:   "if-node",
				Type: workflowNodeTypeIf,
				If: &WorkflowIfNode{
					SourceNodeID: "tool-node",
					Operator:     workflowIfOperatorContains,
					Value:        "${inputs.token}",
					TrueNodeID:   "agent-true",
					FalseNodeID:  "agent-false",
				},
			},
			{ID: "agent-true", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "true branch message"}},
			{ID: "agent-false", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "false branch message"}},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "tool-node"},
			{FromNodeID: "tool-node", ToNodeID: "if-node"},
			{FromNodeID: "if-node", ToNodeID: "agent-true"},
			{FromNodeID: "if-node", ToNodeID: "agent-false"},
			{FromNodeID: "agent-true", ToNodeID: "end-node"},
			{FromNodeID: "agent-false", ToNodeID: "end-node"},
		},
	}
}

func workflowWithLoopNode(toolName string, maxIterations int) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{
				ID:   "loop-node",
				Type: workflowNodeTypeLoop,
				Loop: &WorkflowLoopNode{
					MaxIterations: maxIterations,
					BodyNodeID:    "tool-node",
					ExitNodeID:    "end-node",
				},
			},
			{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: toolName, Arguments: map[string]any{"command": "pwd"}}},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "loop-node"},
			{FromNodeID: "loop-node", ToNodeID: "tool-node"},
			{FromNodeID: "loop-node", ToNodeID: "end-node"},
			{FromNodeID: "tool-node", ToNodeID: "loop-node"},
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

func workflowWithTemplatedToolChain(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{
				ID:   "start-node",
				Type: workflowNodeTypeStart,
				Start: &WorkflowStartNode{Inputs: []WorkflowInputVariable{
					{
						Name:    "command",
						Type:    workflowInputTypeString,
						Default: []byte(`"pwd"`),
					},
				}},
			},
			{
				ID:   "tool-read",
				Type: workflowNodeTypeTool,
				Tool: &WorkflowToolNode{
					ToolName:  toolName,
					Arguments: map[string]any{"command": "${inputs.command}"},
				},
			},
			{
				ID:   "tool-use",
				Type: workflowNodeTypeTool,
				Tool: &WorkflowToolNode{
					ToolName: toolName,
					Arguments: map[string]any{
						"command": "echo ${outputs.tool-read.cwd}",
						"ok":      "${outputs.tool-read.ok}",
						"text":    "run ${inputs.command}",
					},
				},
			},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "tool-read"},
			{FromNodeID: "tool-read", ToNodeID: "tool-use"},
			{FromNodeID: "tool-use", ToNodeID: "end-node"},
		},
	}
}

func workflowWithUndefinedTemplate(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{
				ID:   "tool-node",
				Type: workflowNodeTypeTool,
				Tool: &WorkflowToolNode{
					ToolName:  toolName,
					Arguments: map[string]any{"command": "${outputs.missing.status}"},
				},
			},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "tool-node"},
			{FromNodeID: "tool-node", ToNodeID: "end-node"},
		},
	}
}
