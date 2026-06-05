package orchestration

import (
	"encoding/json"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
	"strings"
	"testing"
)

func TestOrchestrationOwnerPublicOnceReturnsToOwnerAndLogsDispatch(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{{
						ID:        "dispatch-1",
						Name:      orchestrationDispatchToolName,
						Arguments: json.RawMessage(`{"action":"public_once","participant_ids":["agent-2"],"instruction":"speak now"}`),
					}},
				},
				FinishReason: llm.FinishToolCalls,
			},
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{{
						ID:        "dispatch-2",
						Name:      orchestrationDispatchToolName,
						Arguments: json.RawMessage(`{"action":"end_group"}`),
					}},
				},
				FinishReason: llm.FinishToolCalls,
			},
		},
	}
	service.runtimeFactory = proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg:          bridgeconfig.Config{MaxTurns: 4, Provider: bridgeconfig.ProviderConfig{Model: "gpt-5.4"}},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "system prompt",
		},
	}
	run := runOrchestrationTaskNow(t, service, buildOwnerDefinition("agent-1", 3))
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	dispatchResults := groupOutput["dispatch_results"].([]any)
	if len(dispatchResults) != 2 {
		t.Fatalf("expected public dispatch and end_group to be logged, got %#v output=%#v", dispatchResults, groupOutput)
	}
	dispatch := dispatchResults[0].(map[string]any)
	if dispatch["action"] != "public_once" {
		t.Fatalf("unexpected dispatch action: %#v", dispatch)
	}
	memberResults := dispatch["member_results"].([]any)
	if len(memberResults) != 1 {
		t.Fatalf("unexpected member results: %#v", dispatch)
	}
	member := memberResults[0].(map[string]any)
	if member["agent_id"] != "agent-2" {
		t.Fatalf("unexpected public participant: %#v", member)
	}
	if groupOutput["owner_agent_id"] != "agent-1" {
		t.Fatalf("missing owner_agent_id in output: %#v", groupOutput)
	}
	if strings.TrimSpace(groupOutput["owner_session_id"].(string)) == "" {
		t.Fatalf("expected owner_session_id in output: %#v", groupOutput)
	}
	lastDispatch := dispatchResults[1].(map[string]any)
	if lastDispatch["action"] != "end_group" {
		t.Fatalf("expected end_group to be logged, got %#v", lastDispatch)
	}
	assertRunTranscriptSession(t, service, run.Run.SessionIDOutput, nil, []string{
		"编排进入第 1 轮",
		"tool_call: orchestration_dispatch {\"action\":\"public_once\",\"instruction\":\"speak now\",\"participant_ids\":[\"agent-2\"]}",
		"tool: {\"status\":\"success\",\"tool\":\"orchestration_dispatch\"",
		"Member（agent-2） · 第 1 轮",
	})
}

func TestOrchestrationOwnerPrivateDispatchStaysOutOfSharedTranscript(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{{
						ID:        "dispatch-1",
						Name:      orchestrationDispatchToolName,
						Arguments: json.RawMessage(`{"action":"private_once","participant_ids":["agent-1","agent-2"],"instruction":"private chat"}`),
					}},
				},
				FinishReason: llm.FinishToolCalls,
			},
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{{
						ID:        "dispatch-2",
						Name:      orchestrationDispatchToolName,
						Arguments: json.RawMessage(`{"action":"end_group"}`),
					}},
				},
				FinishReason: llm.FinishToolCalls,
			},
		},
	}
	service.runtimeFactory = proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg:          bridgeconfig.Config{MaxTurns: 4, Provider: bridgeconfig.ProviderConfig{Model: "gpt-5.4"}},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "system prompt",
		},
	}
	run := runOrchestrationTaskNow(t, service, buildOwnerDefinition("agent-1", 3))
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	dispatchResults := groupOutput["dispatch_results"].([]any)
	if len(dispatchResults) != 2 {
		t.Fatalf("expected private dispatch and end_group to be logged, got %#v output=%#v", dispatchResults, groupOutput)
	}
	dispatch := dispatchResults[0].(map[string]any)
	if dispatch["action"] != "private_once" {
		t.Fatalf("unexpected dispatch action: %#v", dispatch)
	}
	privateTranscript, ok := dispatch["private_transcript"].([]any)
	if !ok {
		t.Fatalf("expected private_transcript in dispatch log: %#v", dispatch)
	}
	if len(privateTranscript) == 0 {
		t.Fatalf("expected non-empty private_transcript in dispatch log: %#v", dispatch)
	}
	sharedTranscript, ok := groupOutput["shared_transcript"].([]any)
	if !ok {
		sharedTranscript = []any{}
	}
	for _, item := range sharedTranscript {
		entry := item.(map[string]any)
		if strings.Contains(entry["content"].(string), "private") {
			t.Fatalf("shared transcript leaked private content: %#v", sharedTranscript)
		}
	}
}

func TestOrchestrationOwnerPrivateDispatchRetainsTranscriptWhenOwnerNotIncluded(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{{
						ID:        "dispatch-1",
						Name:      orchestrationDispatchToolName,
						Arguments: json.RawMessage(`{"action":"private_once","participant_ids":["agent-2","agent-3"],"instruction":"private chat"}`),
					}},
				},
				FinishReason: llm.FinishToolCalls,
			},
		},
	}
	service.runtimeFactory = proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg:          bridgeconfig.Config{MaxTurns: 4, Provider: bridgeconfig.ProviderConfig{Model: "gpt-5.4"}},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "system prompt",
		},
	}
	run := runOrchestrationTaskNow(t, service, buildOwnerDefinitionWithThirdMember("agent-1", 1))
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	dispatchResults := groupOutput["dispatch_results"].([]any)
	dispatch := dispatchResults[0].(map[string]any)
	privateTranscript, ok := dispatch["private_transcript"].([]any)
	if !ok || len(privateTranscript) == 0 {
		t.Fatalf("expected private_transcript to stay available for user-visible logs: %#v", dispatch)
	}
	if dispatch["owner_visible"] != false {
		t.Fatalf("expected owner_visible=false when owner is absent: %#v", dispatch)
	}
}

func TestOrchestrationOwnerDispatchAppliesRuntimeOverrides(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	configureRuntimeOverrideProviders(t, service)
	completer := &workflowTestCompleter{
		response: &llm.CompletionResponse{
			Message: llm.Message{
				Role: llm.RoleAssistant,
				ToolCalls: []llm.ToolCall{{
					ID:        "dispatch-1",
					Name:      orchestrationDispatchToolName,
					Arguments: json.RawMessage(`{"action":"end_group"}`),
				}},
			},
			FinishReason: llm.FinishToolCalls,
		},
	}
	factory := &captureRuntimeOverrideFactory{
		baseConfig: bridgeconfig.Config{
			MaxTurns: 9,
			Provider: bridgeconfig.ProviderConfig{Model: "baseline-model"},
		},
		completer: completer,
		registry:  tools.NewRegistry(),
	}
	factory.registry.Register(&workflowTestTool{name: "script_exec", output: `{"status":"ok"}`})
	service.runtimeFactory = factory

	ownerDefinition := buildOwnerDefinition("agent-1", 1)
	ownerDefinition.Nodes[1].Agent.RuntimeOverrides = &TaskRuntimeOverrides{
		ProviderName:      "anthropic-main",
		Model:             "claude-3.7",
		SystemPrompt:      "judge override",
		ToolAllowlistOnly: boolPointer(true),
		ToolAllowlist:     []string{"script_exec"},
	}

	run := runOrchestrationTaskNow(t, service, ownerDefinition)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if len(factory.configs) != 1 {
		t.Fatalf("expected one runtime build, got %#v", factory.configs)
	}
	cfg := factory.configs[0]
	if cfg.Provider.Model != "claude-3.7" {
		t.Fatalf("expected owner override model, got %#v", cfg.Provider)
	}
	if cfg.Provider.Type != llm.ProviderAnthropic {
		t.Fatalf("expected owner override provider type, got %#v", cfg.Provider)
	}
	if !cfg.ToolSelector.AllowlistOnly {
		t.Fatalf("expected allowlist_only for owner dispatch runtime, got %#v", cfg.ToolSelector)
	}
	if len(cfg.ToolSelector.Allowlist) != 1 || cfg.ToolSelector.Allowlist[0] != "script_exec" {
		t.Fatalf("unexpected owner allowlist: %#v", cfg.ToolSelector)
	}
	if len(completer.requests) != 1 {
		t.Fatalf("expected one owner completion request, got %#v", completer.requests)
	}
	request := completer.requests[0]
	if strings.Join(completionToolNames(request.Tools), ",") != "orchestration_dispatch,script_exec" {
		t.Fatalf("expected owner to see dispatch plus allowlisted tools, got %#v", request.Tools)
	}
	if len(request.Messages) == 0 || request.Messages[0].Role != llm.RoleSystem {
		t.Fatalf("expected system prompt in owner request, got %#v", request.Messages)
	}
	systemPrompt := request.Messages[0].Text
	if !strings.Contains(systemPrompt, "judge override") {
		t.Fatalf("expected owner override prompt prefix, got %q", systemPrompt)
	}
	if !strings.Contains(systemPrompt, "你是当前群组的群主") ||
		!strings.Contains(systemPrompt, "你可以像普通 agent 一样自由分析") {
		t.Fatalf("expected owner control prompt, got %q", systemPrompt)
	}
}

func runOrchestrationTaskNow(t *testing.T, service *bridgeService, definition *OrchestrationDefinition) taskRunPayload {
	t.Helper()
	createdRaw, _, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindOrchestration,
		Name:            "test-orchestration",
		Orchestration:   definition,
		IntervalSeconds: 60,
		Scope:           taskListScopeOrchestration,
	}, "trace-orchestration-create")
	if err != nil {
		t.Fatalf("create orchestration task: %v", err)
	}
	created := createdRaw.(taskPayload)
	runRaw, _, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID, Scope: taskListScopeOrchestration}, "trace-orchestration-run")
	if err != nil {
		t.Fatalf("run orchestration task: %v", err)
	}
	return runRaw.(taskRunPayload)
}

func findNodeOutput(t *testing.T, results []RunNodeResult, nodeID string) map[string]any {
	t.Helper()
	for _, result := range results {
		if result.NodeID == nodeID {
			output, ok := result.Output.(map[string]any)
			if !ok {
				t.Fatalf("node %s output type: %#v", nodeID, result.Output)
			}
			return output
		}
	}
	t.Fatalf("node result %s not found", nodeID)
	return nil
}

func buildTestOrchestrationDefinition(mode string, maxRounds int) *OrchestrationDefinition {
	return &OrchestrationDefinition{
		Nodes: []OrchestrationNode{
			{ID: "group-1", Type: orchestrationNodeTypeGroup, Group: &OrchestrationGroupNode{Title: "Group 1", SharedContext: "shared", SpeakingMode: mode, MaxRounds: maxRounds}},
			{ID: "agent-1", Type: orchestrationNodeTypeAgent, Agent: &OrchestrationAgentNode{Title: "A", Message: "agent-a"}},
			{ID: "agent-2", Type: orchestrationNodeTypeAgent, Agent: &OrchestrationAgentNode{Title: "B", Message: "agent-b"}},
		},
		Edges: []OrchestrationEdge{
			{FromNodeID: "agent-1", ToNodeID: "group-1", Kind: orchestrationEdgeKindMember},
			{FromNodeID: "agent-2", ToNodeID: "group-1", Kind: orchestrationEdgeKindMember},
		},
	}
}

func buildSingleMemberDefinition(maxRounds int) *OrchestrationDefinition {
	return &OrchestrationDefinition{
		Nodes: []OrchestrationNode{
			{ID: "group-1", Type: orchestrationNodeTypeGroup, Group: &OrchestrationGroupNode{Title: "Group 1", SharedContext: "", SpeakingMode: orchestrationModeSequential, MaxRounds: maxRounds}},
			{ID: "agent-1", Type: orchestrationNodeTypeAgent, Agent: &OrchestrationAgentNode{Title: "Looper", Message: "loop-agent"}},
		},
		Edges: []OrchestrationEdge{
			{FromNodeID: "agent-1", ToNodeID: "group-1", Kind: orchestrationEdgeKindMember},
		},
	}
}

func buildOwnerDefinitionWithThirdMember(ownerAgentID string, maxRounds int) *OrchestrationDefinition {
	definition := buildOwnerDefinition(ownerAgentID, maxRounds)
	definition.Nodes = append(definition.Nodes,
		OrchestrationNode{ID: "agent-3", Type: orchestrationNodeTypeAgent, Agent: &OrchestrationAgentNode{Title: "Member 3", Message: "member-3"}},
	)
	definition.Edges = append(definition.Edges,
		OrchestrationEdge{FromNodeID: "agent-3", ToNodeID: "group-1", Kind: orchestrationEdgeKindMember},
	)
	return definition
}

func TestValidateTaskDefinitionOrchestrationAcceptsEmptyDraft(t *testing.T) {
	task := ScheduledTask{
		TaskKind:      taskKindOrchestration,
		Name:          "empty-orchestration",
		Orchestration: &OrchestrationDefinition{},
	}
	if err := validateTaskDefinition(&task); err != nil {
		t.Fatalf("validate empty orchestration: %v", err)
	}
}

func TestValidateTaskDefinitionOrchestrationRejectsAgentOnlyGraph(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindOrchestration,
		Name:     "broken-orchestration",
		Orchestration: &OrchestrationDefinition{
			Nodes: []OrchestrationNode{
				{ID: "agent-1", Type: orchestrationNodeTypeAgent, Agent: &OrchestrationAgentNode{Title: "Solo", Message: "hello"}},
			},
		},
	}
	err := validateTaskDefinition(&task)
	if err == nil || !strings.Contains(err.Error(), "requires at least 1 group node") {
		t.Fatalf("expected missing-group error, got %v", err)
	}
}

func TestValidateTaskDefinitionOrchestrationOwnerModeRequiresOwnerAgentID(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindOrchestration,
		Name:     "owner-missing",
		Orchestration: &OrchestrationDefinition{
			Nodes: []OrchestrationNode{
				{
					ID:   "group-1",
					Type: orchestrationNodeTypeGroup,
					Group: &OrchestrationGroupNode{
						Title:        "Group 1",
						SpeakingMode: orchestrationModeOwner,
						MaxRounds:    1,
					},
				},
				{ID: "agent-1", Type: orchestrationNodeTypeAgent, Agent: &OrchestrationAgentNode{Title: "Owner", Message: "owner"}},
			},
			Edges: []OrchestrationEdge{
				{FromNodeID: "agent-1", ToNodeID: "group-1", Kind: orchestrationEdgeKindMember},
			},
		},
	}
	err := validateTaskDefinition(&task)
	if err == nil || !strings.Contains(err.Error(), `requires owner_agent_id in owner mode`) {
		t.Fatalf("expected owner_agent_id validation error, got %v", err)
	}
}

func TestValidateTaskDefinitionOrchestrationOwnerModeRequiresOwnerToBeMember(t *testing.T) {
	task := ScheduledTask{
		TaskKind:      taskKindOrchestration,
		Name:          "owner-not-member",
		Orchestration: buildOwnerDefinition("ghost-owner", 1),
	}
	err := validateTaskDefinition(&task)
	if err == nil || !strings.Contains(err.Error(), `owner_agent_id "ghost-owner" must be an existing member`) {
		t.Fatalf("expected owner membership validation error, got %v", err)
	}
}

func TestOrchestrationTaskCreateStripsMemberMaxTurns(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindOrchestration,
		Name:            "strip-member-max-turns",
		Orchestration:   buildSingleMemberDefinitionWithOverrides(&TaskRuntimeOverrides{MaxTurns: intPointer(3)}),
		IntervalSeconds: 60,
		Scope:           taskListScopeOrchestration,
	}, "trace-orchestration-strip-max-turns")
	if err != nil || code != 201 {
		t.Fatalf("create orchestration task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	task, err := service.taskStore().LoadTask(created.ID)
	if err != nil {
		t.Fatalf("load saved orchestration task: %v", err)
	}
	agentNode := task.Orchestration.Nodes[1]
	if agentNode.Agent == nil {
		t.Fatalf("expected agent node payload, got %#v", task.Orchestration.Nodes)
	}
	if agentNode.Agent.RuntimeOverrides != nil && agentNode.Agent.RuntimeOverrides.MaxTurns != nil {
		t.Fatalf("expected orchestration member max_turns to be stripped, got %#v", agentNode.Agent.RuntimeOverrides)
	}
}

func buildSingleMemberDefinitionWithOverrides(overrides *TaskRuntimeOverrides) *OrchestrationDefinition {
	definition := buildSingleMemberDefinition(1)
	definition.Nodes[1].Agent.RuntimeOverrides = overrides
	return definition
}

func buildOwnerDefinition(ownerAgentID string, maxRounds int) *OrchestrationDefinition {
	return &OrchestrationDefinition{
		Nodes: []OrchestrationNode{
			{
				ID:   "group-1",
				Type: orchestrationNodeTypeGroup,
				Group: &OrchestrationGroupNode{
					Title:         "Group 1",
					SharedContext: "",
					SpeakingMode:  orchestrationModeOwner,
					OwnerAgentID:  ownerAgentID,
					MaxRounds:     maxRounds,
				},
			},
			{ID: "agent-1", Type: orchestrationNodeTypeAgent, Agent: &OrchestrationAgentNode{Title: "Owner", Message: "owner-agent"}},
			{ID: "agent-2", Type: orchestrationNodeTypeAgent, Agent: &OrchestrationAgentNode{Title: "Member", Message: "member-agent"}},
		},
		Edges: []OrchestrationEdge{
			{FromNodeID: "agent-1", ToNodeID: "group-1", Kind: orchestrationEdgeKindMember},
			{FromNodeID: "agent-2", ToNodeID: "group-1", Kind: orchestrationEdgeKindMember},
		},
	}
}
