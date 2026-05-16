package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

func TestOrchestrationSequentialRoundSeesPreviousMemberOutput(t *testing.T) {
	executor := func(_ context.Context, message string, sessionID string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
		switch {
		case strings.Contains(message, "agent-a"):
			return "alpha", "session-a", nil
		case strings.Contains(message, "agent-b"):
			if strings.Contains(message, "A: alpha") {
				return "saw-alpha", "session-b", nil
			}
			return "missing-alpha", "session-b", nil
		default:
			return "unknown", "session-x", nil
		}
	}
	_, service, _ := newTestHandlerWithService(t, executor, nil)
	run := runOrchestrationTaskNow(t, service, buildTestOrchestrationDefinition(orchestrationModeSequential, 1))
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	memberResults := groupOutput["member_results"].([]any)
	lastResult := memberResults[len(memberResults)-1].(map[string]any)
	if lastResult["content"] != "saw-alpha" {
		t.Fatalf("expected sequential member to see previous transcript, got %#v", lastResult)
	}
	assertRunTranscriptSession(t, service, run.Run.SessionIDOutput, nil, []string{
		taskRunTranscriptEventMarker,
		"编排进入群组：Group 1（group-1）",
		"编排进入第 1 轮",
		"A（agent-1） · 第 1 轮",
		"B（agent-2） · 第 1 轮",
	})
}

func TestOrchestrationParallelRoundUsesSharedSnapshot(t *testing.T) {
	executor := func(_ context.Context, message string, sessionID string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
		switch {
		case strings.Contains(message, "agent-a"):
			return "alpha", "session-a", nil
		case strings.Contains(message, "agent-b"):
			if strings.Contains(message, "A: alpha") {
				return "snapshot-leaked", "session-b", nil
			}
			return "snapshot-clean", "session-b", nil
		default:
			return "unknown", "session-x", nil
		}
	}
	_, service, _ := newTestHandlerWithService(t, executor, nil)
	run := runOrchestrationTaskNow(t, service, buildTestOrchestrationDefinition(orchestrationModeParallel, 1))
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	memberResults := groupOutput["member_results"].([]any)
	lastResult := memberResults[len(memberResults)-1].(map[string]any)
	if lastResult["content"] != "snapshot-clean" {
		t.Fatalf("expected parallel member snapshot isolation, got %#v", lastResult)
	}
}

func TestOrchestrationReusesMemberSessionWithinRun(t *testing.T) {
	sessionInputs := make([]string, 0, 2)
	executor := func(_ context.Context, _ string, sessionID string, _ string, _ bridgeconfig.Store, sessionStore *session.Store) (string, string, error) {
		sessionInputs = append(sessionInputs, sessionID)
		if sessionID == "" {
			sess := session.NewSession("")
			if err := sessionStore.Save(sess); err != nil {
				return "", "", err
			}
			return "loop", sess.ID, nil
		}
		return "loop", sessionID, nil
	}
	_, service, _ := newTestHandlerWithService(t, executor, nil)
	definition := buildSingleMemberDefinition(2)
	run := runOrchestrationTaskNow(t, service, definition)
	if len(sessionInputs) != 2 || sessionInputs[0] != "" || sessionInputs[1] == "" {
		t.Fatalf("unexpected session reuse inputs: %#v", sessionInputs)
	}
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	sessions := groupOutput["member_session_ids"].(map[string]any)
	if sessions["agent-1"] != sessionInputs[1] {
		t.Fatalf("unexpected member sessions: %#v", sessions)
	}
}

func TestOrchestrationMemberFailureRunTranscriptReplaysPersistedToolMessages(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	seedFailedMemberToolSession(t, service.sessionStore, "failed-member-session")
	service.agentRunner = &workflowTestRunner{
		sessionID: "failed-member-session",
		err:       errors.New("trace_id=trace-member turn=0 complete_once: read response body: context deadline exceeded"),
	}

	run := runOrchestrationTaskNow(t, service, buildSingleMemberDefinition(1))
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	memberResults := groupOutput["member_results"].([]any)
	member := memberResults[0].(map[string]any)
	if member["session_id"] != "failed-member-session" {
		t.Fatalf("expected failed member session id to be preserved, got %#v", member)
	}

	assertRunTranscriptSession(t, service, run.Run.SessionIDOutput, nil, []string{
		"成员状态：Looper（agent-1） · 第 1 轮 -> error",
		"tool_call: read_file {\"path\":\"README.md\"}",
		"tool: {\"status\":\"success\",\"tool\":\"read_file\"",
	})
}

func TestOrchestrationLegacyBoundaryNodesStillRun(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	run := runOrchestrationTaskNow(t, service, buildLegacyBoundaryDefinition(orchestrationModeSequential, 1))
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected legacy run status: %#v", run.Run)
	}
	for _, result := range run.Run.NodeResults {
		if result.NodeType == orchestrationNodeTypeStart || result.NodeType == orchestrationNodeTypeEnd {
			t.Fatalf("legacy boundary nodes should not appear in run results: %#v", run.Run.NodeResults)
		}
	}
}

func TestOrchestrationMemberResultsIncludePresetIDAndOmitMaxTurns(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	preset, err := service.configStore.CreatePreset(bridgeconfig.PresetCreateRequest{
		Name:          "Research",
		ToolAllowlist: []string{},
	})
	if err != nil {
		t.Fatalf("create preset: %v", err)
	}

	run := runOrchestrationTaskNow(t, service, buildSingleMemberDefinitionWithOverrides(&TaskRuntimeOverrides{
		PresetID: preset.ID,
		MaxTurns: intPointer(3),
	}))
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	memberResults := groupOutput["member_results"].([]any)
	runtimeOverrides := memberResults[0].(map[string]any)["runtime_overrides"].(map[string]any)
	if runtimeOverrides["preset_id"] != preset.ID {
		t.Fatalf("expected preset_id in member runtime snapshot, got %#v", runtimeOverrides)
	}
	if _, exists := runtimeOverrides["max_turns"]; exists {
		t.Fatalf("expected max_turns to be omitted from orchestration member snapshot, got %#v", runtimeOverrides)
	}
}

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
		"编排第 1 轮调度：public_once",
		"参与者: agent-2",
		"指令: speak now",
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

func buildLegacyBoundaryDefinition(mode string, maxRounds int) *OrchestrationDefinition {
	return &OrchestrationDefinition{
		Nodes: []OrchestrationNode{
			{ID: "start-node", Type: orchestrationNodeTypeStart},
			{ID: "group-1", Type: orchestrationNodeTypeGroup, Group: &OrchestrationGroupNode{Title: "Group 1", SharedContext: "", SpeakingMode: mode, MaxRounds: maxRounds}},
			{ID: "agent-1", Type: orchestrationNodeTypeAgent, Agent: &OrchestrationAgentNode{Title: "Looper", Message: "loop-agent"}},
			{ID: "end-node", Type: orchestrationNodeTypeEnd},
		},
		Edges: []OrchestrationEdge{
			{FromNodeID: "start-node", ToNodeID: "group-1", Kind: orchestrationEdgeKindControl},
			{FromNodeID: "group-1", ToNodeID: "end-node", Kind: orchestrationEdgeKindControl},
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

func seedFailedMemberToolSession(t *testing.T, store *session.Store, sessionID string) {
	t.Helper()
	sess := session.NewSession("")
	sess.ID = sessionID
	sess.AddMessage(llm.Message{
		Role: llm.RoleAssistant,
		ToolCalls: []llm.ToolCall{{
			ID:        "call-1",
			Name:      "read_file",
			Arguments: json.RawMessage(`{"path":"README.md"}`),
		}},
	})
	sess.AddMessage(llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: "call-1",
		Text:       agent.FormatToolResult("read_file", "trace-member", "file body", nil),
	})
	if err := store.Save(sess); err != nil {
		t.Fatalf("save failed member tool session: %v", err)
	}
}
