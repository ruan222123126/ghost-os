package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
	"strings"
	"testing"
)

func TestOrchestrationOwnerPublicDispatchAppendsSharedTranscript(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	installOwnerDispatchRuntime(
		service,
		dispatchToolResponse("dispatch-1", `{"action":"public_once","participant_ids":["agent-2"],"instruction":"speak now"}`),
		dispatchToolResponse("dispatch-2", `{"action":"end_group"}`),
	)

	run := runOrchestrationTaskNow(t, service, buildOwnerDefinition("agent-1", 3))
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	sharedTranscript := outputSlice(t, groupOutput, "shared_transcript")
	if len(sharedTranscript) != 1 {
		t.Fatalf("expected one public transcript entry, got %#v", sharedTranscript)
	}
	entry := sharedTranscript[0].(map[string]any)
	if entry["agent_id"] != "agent-2" || entry["content"] != "ok" {
		t.Fatalf("expected public member result in shared transcript, got %#v", entry)
	}
}

func TestOrchestrationOwnerPrivateDispatchDoesNotAppendSharedTranscript(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	installOwnerDispatchRuntime(
		service,
		dispatchToolResponse("dispatch-1", `{"action":"private_once","participant_ids":["agent-1","agent-2"],"instruction":"private chat"}`),
		dispatchToolResponse("dispatch-2", `{"action":"end_group"}`),
	)

	run := runOrchestrationTaskNow(t, service, buildOwnerDefinition("agent-1", 3))
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	if sharedTranscript := outputSlice(t, groupOutput, "shared_transcript"); len(sharedTranscript) != 0 {
		t.Fatalf("expected private dispatch to leave shared transcript empty, got %#v", sharedTranscript)
	}
	dispatchResults := outputSlice(t, groupOutput, "dispatch_results")
	privateTranscript := dispatchResults[0].(map[string]any)["private_transcript"].([]any)
	if len(privateTranscript) == 0 {
		t.Fatalf("expected owner-visible private transcript in dispatch log, got %#v", dispatchResults[0])
	}
}

func TestOrchestrationOwnerEndGroupStopsBeforeMemberDispatch(t *testing.T) {
	agentCalls := 0
	executor := func(_ context.Context, _ string, _ string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
		agentCalls++
		return "unexpected member dispatch", "member-session", nil
	}
	_, service, _ := newTestHandlerWithService(t, executor, nil)
	installOwnerDispatchRuntime(service, dispatchToolResponse("dispatch-1", `{"action":"end_group"}`))

	run := runOrchestrationTaskNow(t, service, buildOwnerDefinition("agent-1", 3))
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	if agentCalls != 0 {
		t.Fatalf("expected end_group to skip member dispatch, got %d calls", agentCalls)
	}
	if completed := outputInt(t, groupOutput, "completed_rounds"); completed != 0 {
		t.Fatalf("expected completed_rounds=0 after immediate end_group, got %d", completed)
	}
	if memberResults := outputSlice(t, groupOutput, "member_results"); len(memberResults) != 0 {
		t.Fatalf("expected no member results after immediate end_group, got %#v", memberResults)
	}
	dispatchResults := outputSlice(t, groupOutput, "dispatch_results")
	if len(dispatchResults) != 1 || dispatchResults[0].(map[string]any)["action"] != "end_group" {
		t.Fatalf("expected single end_group dispatch result, got %#v", dispatchResults)
	}
}

func TestOrchestrationMemberRuntimeOverridesApplyToAgentRuntime(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	configureRuntimeOverrideProviders(t, service)
	preset := createRuntimeOverridePreset(t, service)
	completer := &workflowTestCompleter{
		response: &llm.CompletionResponse{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "member ok"},
			FinishReason: llm.FinishStop,
		},
	}
	registry := tools.NewRegistry()
	registry.Register(&workflowTestTool{name: "script_exec", output: `{"status":"ok"}`})
	registry.Register(&workflowTestTool{name: "web_search", output: `{"status":"ok"}`})
	factory := &captureRuntimeOverrideFactory{completer: completer, registry: registry}
	service.runtimeFactory = factory
	service.agentRunner = NewSessionAgentRunner(factory, service.configStore, service.sessionStore, service.runRegistry)

	run := runOrchestrationTaskNow(t, service, buildSingleMemberDefinitionWithOverrides(&TaskRuntimeOverrides{
		ProviderName:      "anthropic-main",
		Model:             "claude-3.7",
		PresetID:          preset.ID,
		ToolAllowlistOnly: boolPointer(true),
		ToolAllowlist:     []string{"script_exec"},
		MaxTurns:          intPointer(9),
	}))

	assertMemberRuntimeRun(t, run, factory, completer, preset.ID)
}

func assertMemberRuntimeRun(
	t *testing.T,
	run taskRunPayload,
	factory *captureRuntimeOverrideFactory,
	completer *workflowTestCompleter,
	presetID string,
) {
	t.Helper()
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	assertCapturedMemberRuntimeConfig(t, factory)
	assertMemberCompletionRequest(t, completer)
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	memberResult := outputSlice(t, groupOutput, "member_results")[0].(map[string]any)
	runtimeOverrides := memberResult["runtime_overrides"].(map[string]any)
	if runtimeOverrides["preset_id"] != presetID {
		t.Fatalf("expected member preset_id snapshot, got %#v", runtimeOverrides)
	}
	if _, exists := runtimeOverrides["max_turns"]; exists {
		t.Fatalf("expected orchestration member max_turns to be stripped, got %#v", runtimeOverrides)
	}
}

func assertCapturedMemberRuntimeConfig(t *testing.T, factory *captureRuntimeOverrideFactory) {
	t.Helper()
	if len(factory.configs) != 1 {
		t.Fatalf("expected one runtime config build, got %#v", factory.configs)
	}
	cfg := factory.configs[0]
	if cfg.Provider.Type != llm.ProviderAnthropic || cfg.Provider.Model != "claude-3.7" {
		t.Fatalf("unexpected provider/model override config: %+v", cfg.Provider)
	}
	if !cfg.ToolSelector.AllowlistOnly || strings.Join(cfg.ToolSelector.Allowlist, ",") != "script_exec" {
		t.Fatalf("unexpected member tool allowlist config: %+v", cfg.ToolSelector)
	}
}

func assertMemberCompletionRequest(t *testing.T, completer *workflowTestCompleter) {
	t.Helper()
	if len(completer.requests) != 1 {
		t.Fatalf("expected one completion request, got %d", len(completer.requests))
	}
	request := completer.requests[0]
	if strings.Join(completionToolNames(request.Tools), ",") != "script_exec" {
		t.Fatalf("unexpected completion tool scope: %#v", request.Tools)
	}
	systemPrompt := request.Messages[0].Text
	if !strings.Contains(systemPrompt, "preset rule") ||
		!strings.Contains(systemPrompt, "preset core") ||
		!strings.Contains(systemPrompt, "preset context") {
		t.Fatalf("expected preset prompt in member runtime request, got %q", systemPrompt)
	}
}

func installOwnerDispatchRuntime(service *bridgeService, responses ...*llm.CompletionResponse) *proTestCompleter {
	completer := &proTestCompleter{responses: responses}
	service.runtimeFactory = proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg:          bridgeconfig.Config{MaxTurns: 4, Provider: bridgeconfig.ProviderConfig{Model: "gpt-5.4"}},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "system prompt",
		},
	}
	return completer
}

func dispatchToolResponse(id string, args string) *llm.CompletionResponse {
	return &llm.CompletionResponse{
		Message: llm.Message{
			Role: llm.RoleAssistant,
			ToolCalls: []llm.ToolCall{{
				ID:        id,
				Name:      orchestrationDispatchToolName,
				Arguments: json.RawMessage(args),
			}},
		},
		FinishReason: llm.FinishToolCalls,
	}
}

func createRuntimeOverridePreset(t *testing.T, service *bridgeService) bridgeconfig.Preset {
	t.Helper()
	library := []bridgeconfig.SystemPromptLibraryItem{
		{ID: "rule-preset", Name: "Rule Preset", InsertPoint: bridgeconfig.SystemPromptInsertPointRule, Content: "preset rule", Active: false},
		{ID: "core-preset", Name: "Core Preset", InsertPoint: bridgeconfig.SystemPromptInsertPointCoreJob, Content: "preset core", Active: false},
		{ID: "context-preset", Name: "Context Preset", InsertPoint: bridgeconfig.SystemPromptInsertPointContext, Content: "preset context", Active: false},
	}
	if _, err := service.configStore.UpdateSystemPrompts(bridgeconfig.SystemPromptUpdateRequest{PromptLibrary: &library}); err != nil {
		t.Fatalf("seed prompt library: %v", err)
	}
	preset, err := service.configStore.CreatePreset(bridgeconfig.PresetCreateRequest{
		Name:          "Research",
		ToolAllowlist: []string{},
		PromptRefs: bridgeconfig.PresetPromptRefs{
			Rule:    "rule-preset",
			CoreJob: "core-preset",
			Context: []string{"context-preset"},
		},
	})
	if err != nil {
		t.Fatalf("create preset: %v", err)
	}
	return preset
}

func completionToolNames(defs []llm.ToolDef) []string {
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		names = append(names, def.Name)
	}
	return names
}

func outputSlice(t *testing.T, output map[string]any, key string) []any {
	t.Helper()
	if output[key] == nil {
		return nil
	}
	items, ok := output[key].([]any)
	if !ok {
		t.Fatalf("expected %s slice in output, got %#v", key, output[key])
	}
	return items
}

func outputInt(t *testing.T, output map[string]any, key string) int {
	t.Helper()
	switch value := output[key].(type) {
	case int:
		return value
	case float64:
		return int(value)
	default:
		t.Fatalf("expected %s int in output, got %#v", key, output[key])
		return 0
	}
}

func TestOrchestrationOwnerRepairsEmptyResponseIntoDispatch(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	completer := installOwnerDispatchRuntime(
		service,
		ownerAssistantResponse(""),
		dispatchToolResponse("dispatch-1", `{"action":"end_group"}`),
	)

	run := runOrchestrationTaskNow(t, service, buildOwnerDefinition("agent-1", 1))
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("expected repaired owner run to succeed, got %#v", run.Run)
	}
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	dispatchResults := groupOutput["dispatch_results"].([]any)
	if len(dispatchResults) != 1 || dispatchResults[0].(map[string]any)["action"] != "end_group" {
		t.Fatalf("expected repaired end_group dispatch, got %#v", dispatchResults)
	}
	if len(completer.requests) != 2 {
		t.Fatalf("expected owner repair round, got %#v", completer.requests)
	}
	repairRequest := completer.requests[1]
	if repairRequest.ToolChoice != "" {
		t.Fatalf("expected repair request to keep free tool choice, got %#v", repairRequest)
	}
	lastMessage := repairRequest.Messages[len(repairRequest.Messages)-1].Text
	if !strings.Contains(lastMessage, "(empty response)") || !strings.Contains(lastMessage, "did not advance the owner-led orchestration yet") {
		t.Fatalf("unexpected owner repair prompt: %q", lastMessage)
	}
}

func TestOrchestrationOwnerFailureRetainsOwnerSessionIDAfterRepairAttempt(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	completer := installOwnerDispatchRuntime(
		service,
		ownerAssistantResponse(""),
		ownerAssistantResponse("我还是不调用工具。"),
	)

	run := runOrchestrationTaskNow(t, service, buildOwnerDefinition("agent-1", 1))
	if run.Run.Status != taskRunStatusError {
		t.Fatalf("expected owner run to fail after invalid repair, got %#v", run.Run)
	}
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	if strings.TrimSpace(groupOutput["owner_session_id"].(string)) == "" {
		t.Fatalf("expected failed owner run to retain owner_session_id, got %#v", groupOutput)
	}
	if len(completer.requests) != 2 || completer.requests[1].ToolChoice != "" {
		t.Fatalf("expected repair request before failure, got %#v", completer.requests)
	}
}

func ownerAssistantResponse(text string) *llm.CompletionResponse {
	return &llm.CompletionResponse{
		Message: llm.Message{
			Role: llm.RoleAssistant,
			Text: text,
		},
		FinishReason: llm.FinishStop,
	}
}

func TestOrchestrationOwnerPrivateSendReachesOnlyRecipient(t *testing.T) {
	executor := privateSendRecipientExecutor()
	_, service, _ := newTestHandlerWithService(t, executor, nil)
	installPrivateSendOwnerRuntime(service)

	run := runOrchestrationTaskNow(t, service, buildOwnerDefinition("agent-1", 3))
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	dispatches := groupOutput["dispatch_results"].([]any)
	first := dispatches[0].(map[string]any)
	if first["action"] != "private_send" {
		t.Fatalf("expected private_send first, got %#v", first)
	}
	deliveries := first["private_deliveries"].([]any)
	if len(deliveries) != 1 || deliveries[0].(map[string]any)["content"] != "secret-role" {
		t.Fatalf("expected visible private delivery content, got %#v", first)
	}
	member := dispatches[1].(map[string]any)["member_results"].([]any)[0].(map[string]any)
	if member["content"] != "saw-secret" {
		t.Fatalf("expected recipient to see private content, got %#v", member)
	}
	assertRunTranscriptSession(t, service, run.Run.SessionIDOutput, nil, []string{"编排进入第 1 轮"})
	loaded, err := service.sessionStore.Load(run.Run.SessionIDOutput)
	if err != nil {
		t.Fatalf("load run transcript session %q: %v", run.Run.SessionIDOutput, err)
	}
	transcript := sessionMessagesText(loaded.Messages)
	required := []string{
		"tool_call: orchestration_dispatch {\"action\":\"private_send\",\"private_messages\":[{\"content\":\"secret-role\",\"participant_id\":\"agent-2\"}]}",
		"tool: {\"status\":\"success\",\"tool\":\"orchestration_dispatch\"",
		"secret-role",
	}
	for _, expected := range required {
		if !strings.Contains(transcript, expected) {
			t.Fatalf("run transcript missing %q in:\n%s", expected, transcript)
		}
	}
}

func privateSendRecipientExecutor() agentExecutorFunc {
	return func(_ context.Context, message string, _ string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
		if strings.Contains(message, "secret-role") {
			return "saw-secret", "member-session", nil
		}
		return "missing-secret", "member-session", nil
	}
}

func installPrivateSendOwnerRuntime(service *bridgeService) {
	completer := &proTestCompleter{responses: []*llm.CompletionResponse{
		dispatchToolResponse("dispatch-1", `{"action":"private_send","private_messages":[{"participant_id":"agent-2","content":"secret-role"}]}`),
		dispatchToolResponse("dispatch-2", `{"action":"public_once","participant_ids":["agent-2"],"instruction":"speak now"}`),
		dispatchToolResponse("dispatch-3", `{"action":"end_group"}`),
	}}
	service.runtimeFactory = proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg:          bridgeconfig.Config{MaxTurns: 4, Provider: bridgeconfig.ProviderConfig{Model: "gpt-5.4"}},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "system prompt",
		},
	}
}

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
	sessionInputs := make([]string, 0, 3)
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
	definition := buildSingleMemberDefinition(3)
	run := runOrchestrationTaskNow(t, service, definition)
	if len(sessionInputs) != 3 || sessionInputs[0] != "" || sessionInputs[1] == "" || sessionInputs[2] == "" {
		t.Fatalf("unexpected session reuse inputs: %#v", sessionInputs)
	}
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	sessions := groupOutput["member_session_ids"].(map[string]any)
	if sessions["agent-1"] != sessionInputs[2] {
		t.Fatalf("unexpected member sessions: %#v", sessions)
	}
	loaded, err := service.sessionStore.Load(run.Run.SessionIDOutput)
	if err != nil {
		t.Fatalf("load run transcript session %q: %v", run.Run.SessionIDOutput, err)
	}
	transcript := sessionMessagesText(loaded.Messages)
	required := []string{"编排进入第 1 轮", "编排进入第 2 轮", "编排进入第 3 轮"}
	for _, text := range required {
		if !strings.Contains(transcript, text) {
			t.Fatalf("run transcript missing %q in:\n%s", text, transcript)
		}
	}
	if strings.Count(transcript, "Looper（agent-1） · 第 1 轮\n\nloop") != 1 {
		t.Fatalf("expected round 1 sender exactly once, got transcript:\n%s", transcript)
	}
	if strings.Count(transcript, "Looper（agent-1） · 第 2 轮\n\nloop") != 1 {
		t.Fatalf("expected round 2 sender exactly once, got transcript:\n%s", transcript)
	}
	if strings.Count(transcript, "Looper（agent-1） · 第 3 轮\n\nloop") != 1 {
		t.Fatalf("expected round 3 sender exactly once, got transcript:\n%s", transcript)
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
