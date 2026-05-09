package orchestration

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
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
