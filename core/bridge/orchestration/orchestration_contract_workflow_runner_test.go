package orchestration

import (
	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
	"net/http"
	"strings"
	"testing"
	"time"
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
	completer := &workflowTestCompleter{
		response: &llm.CompletionResponse{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "agent done"},
			FinishReason: llm.FinishStop,
		},
	}
	factory := &captureRuntimeOverrideFactory{
		completer: completer,
		registry:  tools.NewRegistry(),
	}
	service.runtimeFactory = factory
	service.agentRunner = NewSessionAgentRunner(
		factory,
		service.configStore,
		service.sessionStore,
		service.runRegistry,
	)

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
	if len(completer.requests) != 1 {
		t.Fatalf("expected one completion request, got %d", len(completer.requests))
	}
	if len(factory.configs) != 1 {
		t.Fatalf("expected one runtime config build, got %d", len(factory.configs))
	}
	cfg := factory.configs[0]
	if cfg.Provider.Type != llm.ProviderOpenAI || cfg.Provider.Model != "gpt-5.4" {
		t.Fatalf("unexpected provider/model overrides: %#v", cfg.Provider)
	}
	if cfg.MaxTurns != 2 {
		t.Fatalf("unexpected max_turns override: %d", cfg.MaxTurns)
	}
	if len(run.Run.RunCards) != 1 || run.Run.RunCards[0].SourceSessionID == "" {
		t.Fatalf("expected workflow agent run card with source session, got %#v", run.Run.RunCards)
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
