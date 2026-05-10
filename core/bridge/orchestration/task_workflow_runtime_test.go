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
	assertRunTranscriptSession(t, service, run.Run.SessionIDOutput, []string{"workflow-session"}, []string{
		taskRunTranscriptEventMarker,
		"节点 tool-node 使用工具：script_exec",
		"节点 llm-node（LLM）",
		"节点 agent-node（agent）",
		"agent done",
	})
	assertWorkflowNodeResultsSequence(
		t,
		run.Run.NodeResults,
		[]string{"start-node", "tool-node", "llm-node", "agent-node", "end-node"},
	)
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

func TestTaskWorkflowRunNowPassesAgentRuntimeOverrides(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	configureRuntimeOverrideProviders(t, service)
	agentRunner := &workflowTestRunner{message: "agent done", sessionID: "workflow-session"}
	service.agentRunner = agentRunner

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind: taskKindWorkflow,
		Workflow: &WorkflowDefinition{
			Nodes: []WorkflowNode{
				{ID: "start-node", Type: workflowNodeTypeStart},
				{
					ID:   "agent-node",
					Type: workflowNodeTypeAgent,
					Agent: &WorkflowAgentNode{
						Message: "run agent",
						RuntimeOverrides: &TaskRuntimeOverrides{
							ProviderName:      "openai-main",
							Model:             "gpt-5.4",
							SystemPrompt:      "override prompt",
							ToolAllowlistOnly: boolPointer(true),
							ToolAllowlist:     []string{"script_exec"},
							MaxTurns:          intPointer(2),
						},
					},
				},
				{ID: "end-node", Type: workflowNodeTypeEnd},
			},
			Edges: []WorkflowEdge{
				{FromNodeID: "start-node", ToNodeID: "agent-node"},
				{FromNodeID: "agent-node", ToNodeID: "end-node"},
			},
		},
		IntervalSeconds: 60,
	}, "trace-workflow-agent-override-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create workflow task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-agent-override-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if len(agentRunner.calls) != 1 {
		t.Fatalf("expected one agent runner call, got %#v", agentRunner.calls)
	}
	if agentRunner.calls[0].runtimeOverrides == nil {
		t.Fatalf("expected runtime overrides to be forwarded, got %#v", agentRunner.calls[0])
	}
	if agentRunner.calls[0].runtimeOverrides.ProviderName != "openai-main" ||
		agentRunner.calls[0].runtimeOverrides.Model != "gpt-5.4" {
		t.Fatalf("unexpected provider/model overrides: %#v", agentRunner.calls[0].runtimeOverrides)
	}
	if agentRunner.calls[0].runtimeOverrides.MaxTurns == nil || *agentRunner.calls[0].runtimeOverrides.MaxTurns != 2 {
		t.Fatalf("unexpected max_turns override: %#v", agentRunner.calls[0].runtimeOverrides)
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
	assertRunTranscriptSession(t, service, run.Run.SessionIDOutput, []string{"if-session"}, []string{
		"工作流 if 判断：if-node branch=true",
		"节点 agent-true（agent）",
		"if branch done",
	})
	if len(agentRunner.calls) != 1 || agentRunner.calls[0].message != "true branch message" {
		t.Fatalf("unexpected agent branch: %#v", agentRunner.calls)
	}
}

func TestTaskWorkflowRunNowKeepsIfValueLiteral(t *testing.T) {
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
	assertRunTranscriptSession(t, service, run.Run.SessionIDOutput, []string{"if-template-session"}, []string{
		"工作流 if 判断：if-node branch=false",
		"value=${inputs.token}",
		"节点 agent-false（agent）",
	})
	if len(agentRunner.calls) != 1 || agentRunner.calls[0].message != "false branch message" {
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
	assertRunTranscriptSession(t, service, run.Run.SessionIDOutput, nil, []string{
		"工作流进入循环：loop-node 第 1/3 轮",
		"工作流退出循环：loop-node",
	})
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
	assertWorkflowNodeResultsMonotonic(t, run.Run.NodeResults)
	for _, node := range run.Run.NodeResults {
		if !strings.HasPrefix(node.NodeID, "tool-") {
			continue
		}
		if strings.TrimSpace(node.BranchID) == "" {
			t.Fatalf("expected tool branch node to include branch_id: %#v", node)
		}
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
	assertRunTranscriptSession(t, service, run.Run.SessionIDOutput, []string{"await-session"}, []string{
		taskRunTranscriptEventMarker,
		"任务运行结束：awaiting_human",
		"节点 agent-node（agent）",
		"Need approval",
	})
	if !strings.Contains(run.Run.ResponsePreview, "Need approval") {
		t.Fatalf("unexpected awaiting preview: %#v", run.Run)
	}
}

func TestTaskWorkflowRunNowAllowsAllToolsWhenAllowlistCleared(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	tool := &workflowTestTool{name: "script_exec", output: `{"status":"ok"}`}
	registry := tools.NewRegistry()
	registry.Register(tool)
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

func TestTaskWorkflowRunNowKeepsDeprecatedTemplateTokensLiteral(t *testing.T) {
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
	if tool.calls[0]["command"] != "${inputs.command}" {
		t.Fatalf("unexpected first tool args: %#v", tool.calls[0])
	}
	if tool.calls[1]["command"] != "echo ${outputs.tool-read.cwd}" {
		t.Fatalf("unexpected second tool command: %#v", tool.calls[1])
	}
	if tool.calls[1]["ok"] != "${outputs.tool-read.ok}" {
		t.Fatalf("unexpected second tool ok arg: %#v", tool.calls[1])
	}
	if tool.calls[1]["text"] != "run ${inputs.command}" {
		t.Fatalf("unexpected second tool text arg: %#v", tool.calls[1])
	}
}

func TestTaskWorkflowRunNowResolvesFindIconVariableForDownstreamTool(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "screen_control,script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	screenTool := &workflowTestTool{
		name:   "screen_control",
		output: `{"matches":[{"center":{"x":321,"y":654}}],"display_id":7}`,
	}
	scriptTool := &workflowTestTool{name: "script_exec", output: `{"status":"ok"}`}
	registry := tools.NewRegistry()
	registry.Register(screenTool)
	registry.Register(scriptTool)
	service.runtimeFactory = proTestRuntimeFactory{
		deps: NewRuntimeDependencies(
			bridgeconfig.Config{Task: bridgeconfig.TaskConfig{WorkflowToolAllowlist: []string{"screen_control", "script_exec"}}},
			&workflowTestCompleter{},
			registry,
			"",
			nil,
		),
	}

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflowWithFindIconVariableConsumer("script_exec"),
		IntervalSeconds: 60,
	}, "trace-workflow-find-icon-variable-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create workflow task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-find-icon-variable-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if len(scriptTool.calls) != 1 {
		t.Fatalf("unexpected script_exec call count: %#v", scriptTool.calls)
	}
	if scriptTool.calls[0]["x"] != float64(321) || scriptTool.calls[0]["y"] != float64(654) {
		t.Fatalf("unexpected resolved coordinates: %#v", scriptTool.calls[0])
	}
	point, ok := scriptTool.calls[0]["point"].(map[string]any)
	if !ok || point["display_id"] != float64(7) {
		t.Fatalf("unexpected resolved point payload: %#v", scriptTool.calls[0]["point"])
	}
	if scriptTool.calls[0]["label"] != "prefix 321" {
		t.Fatalf("unexpected resolved label: %#v", scriptTool.calls[0]["label"])
	}
}

func TestTaskWorkflowRunNowFailsWhenFindIconVariableUsedBeforeDefinition(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	tool := &workflowTestTool{name: "script_exec", output: `{"status":"ok"}`}
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
		Workflow:        workflowWithUndefinedFindIconVariable("script_exec"),
		IntervalSeconds: 60,
	}, "trace-workflow-find-icon-missing-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create workflow task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-find-icon-missing-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusError {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if !strings.Contains(run.Run.Error, `workflow variable "find_icon" is not defined`) {
		t.Fatalf("unexpected run error: %#v", run.Run)
	}
}

func TestTaskWorkflowRunNowKeepsUndefinedTemplateLiteral(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	tool := &workflowTestTool{name: "script_exec", output: `{"status":"ok"}`}
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
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if len(tool.calls) != 1 || tool.calls[0]["command"] != "${outputs.missing.status}" {
		t.Fatalf("unexpected literal template argument: %#v", tool.calls)
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
	message          string
	sessionID        string
	runtimeOverrides *TaskRuntimeOverrides
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

func (r *workflowTestRunner) RunTurnWithOverrides(
	_ context.Context,
	message string,
	sessionID string,
	_ string,
	runtimeOverrides *TaskRuntimeOverrides,
) (string, string, error) {
	r.calls = append(r.calls, workflowRunnerCall{
		message:          message,
		sessionID:        sessionID,
		runtimeOverrides: cloneTaskRuntimeOverrides(runtimeOverrides),
	})
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

func assertWorkflowNodeResultsSequence(t *testing.T, nodeResults []RunNodeResult, expectedNodeIDs []string) {
	t.Helper()
	if len(nodeResults) != len(expectedNodeIDs) {
		t.Fatalf("unexpected node result count: got %d want %d", len(nodeResults), len(expectedNodeIDs))
	}
	for index, node := range nodeResults {
		if node.CompletedSeq != index+1 {
			t.Fatalf("unexpected completed_seq at index=%d: %#v", index, node)
		}
		if node.NodeID != expectedNodeIDs[index] {
			t.Fatalf("unexpected node order at index=%d: got %q want %q", index, node.NodeID, expectedNodeIDs[index])
		}
		if strings.TrimSpace(node.Status) == "" || node.StartedAt.IsZero() || node.FinishedAt.IsZero() {
			t.Fatalf("expected non-empty status/timestamps: %#v", node)
		}
	}
}

func assertWorkflowNodeResultsMonotonic(t *testing.T, nodeResults []RunNodeResult) {
	t.Helper()
	if len(nodeResults) == 0 {
		t.Fatal("expected workflow node results")
	}
	lastSeq := 0
	for index, node := range nodeResults {
		if node.CompletedSeq <= lastSeq {
			t.Fatalf("completed_seq must be strictly increasing at index=%d: %#v", index, node)
		}
		lastSeq = node.CompletedSeq
	}
}

func assertRunTranscriptSession(
	t *testing.T,
	service *bridgeService,
	sessionID string,
	disallowed []string,
	required []string,
) {
	t.Helper()
	if strings.TrimSpace(sessionID) == "" {
		t.Fatal("expected run transcript session id")
	}
	for _, blocked := range disallowed {
		if sessionID == blocked {
			t.Fatalf("expected display transcript session, got execution session %q", sessionID)
		}
	}
	if service == nil || service.sessionStore == nil {
		t.Fatal("session store is not configured")
	}
	loaded, err := service.sessionStore.Load(sessionID)
	if err != nil {
		t.Fatalf("load run transcript session %q: %v", sessionID, err)
	}
	transcript := sessionMessagesText(loaded.Messages)
	for _, expected := range required {
		if !strings.Contains(transcript, expected) {
			t.Fatalf("run transcript missing %q in:\n%s", expected, transcript)
		}
	}
}

func sessionMessagesText(messages []llm.Message) string {
	parts := make([]string, 0, len(messages)*2)
	for _, message := range messages {
		parts = append(parts, string(message.Role)+": "+message.Text)
		for _, call := range message.ToolCalls {
			parts = append(parts, "tool_call: "+call.Name+" "+string(call.Arguments))
		}
	}
	return strings.Join(parts, "\n")
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

func workflowWithFindIconVariableConsumer(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{
				ID:   "screen-node",
				Type: workflowNodeTypeTool,
				Tool: &WorkflowToolNode{
					ToolName: "screen_control",
					Arguments: map[string]any{
						"mode":   "atomic",
						"action": "find_icon",
						"params": map[string]any{"template_path": "/tmp/icon.png"},
					},
				},
			},
			{
				ID:   "tool-use",
				Type: workflowNodeTypeTool,
				Tool: &WorkflowToolNode{
					ToolName: toolName,
					Arguments: map[string]any{
						"x":     "${find_icon.x}",
						"y":     "${find_icon.y}",
						"point": "${find_icon}",
						"label": "prefix ${find_icon.x}",
					},
				},
			},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "screen-node"},
			{FromNodeID: "screen-node", ToNodeID: "tool-use"},
			{FromNodeID: "tool-use", ToNodeID: "end-node"},
		},
	}
}

func workflowWithUndefinedFindIconVariable(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{
				ID:   "tool-node",
				Type: workflowNodeTypeTool,
				Tool: &WorkflowToolNode{
					ToolName:  toolName,
					Arguments: map[string]any{"point": "${find_icon}"},
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
