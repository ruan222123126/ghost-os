package orchestration

import (
	"context"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	apptasks "ghost-os/bridge/orchestration/internal/app/tasks"
	"ghost-os/bridge/tools"
	"net/http"
	"strings"
	"testing"
)

func TestNormalizeTaskRuntimeOverrides(t *testing.T) {
	normalized, err := apptasks.NormalizeTaskRuntimeOverrides(nil)
	if err != nil {
		t.Fatalf("normalize nil overrides: %v", err)
	}
	if normalized != nil {
		t.Fatalf("expected nil overrides, got %#v", normalized)
	}

	normalized, err = apptasks.NormalizeTaskRuntimeOverrides(&TaskRuntimeOverrides{
		Model:         " ",
		ToolAllowlist: []string{"  ", "\n"},
	})
	if err != nil {
		t.Fatalf("normalize empty overrides: %v", err)
	}
	if normalized != nil {
		t.Fatalf("expected empty overrides to collapse to nil, got %#v", normalized)
	}

	normalized, err = apptasks.NormalizeTaskRuntimeOverrides(&TaskRuntimeOverrides{
		ProviderName:      " openai-main ",
		Model:             " gpt-5.4 ",
		SystemPrompt:      " be concise ",
		PresetID:          " preset-a ",
		ToolAllowlist:     []string{"web_search", "script_exec", "web_search"},
		ToolAllowlistOnly: boolPointer(true),
		MaxTurns:          intPointer(3),
	})
	if err != nil {
		t.Fatalf("normalize valid overrides: %v", err)
	}
	if normalized.ProviderName != "openai-main" {
		t.Fatalf("unexpected provider_name: %#v", normalized)
	}
	if normalized.Model != "gpt-5.4" {
		t.Fatalf("unexpected model: %#v", normalized)
	}
	if normalized.SystemPrompt != "be concise" {
		t.Fatalf("unexpected system_prompt: %#v", normalized)
	}
	if normalized.PresetID != "preset-a" {
		t.Fatalf("unexpected preset_id: %#v", normalized)
	}
	if normalized.ToolAllowlistOnly == nil || !*normalized.ToolAllowlistOnly {
		t.Fatalf("expected tool_allowlist_only=true, got %#v", normalized)
	}
	if normalized.MaxTurns == nil || *normalized.MaxTurns != 3 {
		t.Fatalf("unexpected max_turns: %#v", normalized)
	}
	if strings.Join(normalized.ToolAllowlist, ",") != "script_exec,web_search" {
		t.Fatalf("unexpected allowlist normalization: %#v", normalized.ToolAllowlist)
	}
}

func TestTaskCreateRejectsUnknownRuntimeOverrideTool(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	_, code, err := service.executeTaskCreateAction(taskCreateParams{
		Message:          "run with bad override",
		TaskKind:         taskKindAgentMessage,
		RuntimeOverrides: &TaskRuntimeOverrides{ToolAllowlist: []string{"ghost_tool"}},
		IntervalSeconds:  60,
	}, "trace-task-runtime-override-invalid-tool")
	if err == nil {
		t.Fatal("expected runtime override tool validation to fail")
	}
	if code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d want %d", code, http.StatusBadRequest)
	}
	if !strings.Contains(err.Error(), "unknown tool in tool_allowlist: ghost_tool") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskRunNowAppliesRuntimeOverrideToolAllowlist(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	if err := service.configStore.UpdateTool(bridgeconfig.ToolUpdateRequest{
		Name:    "script_exec",
		Enabled: boolPointer(false),
	}); err != nil {
		t.Fatalf("disable script_exec: %v", err)
	}

	completer := &workflowTestCompleter{
		response: &llm.CompletionResponse{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "ok"},
			FinishReason: llm.FinishStop,
		},
	}
	registry := tools.NewRegistry()
	registry.Register(&workflowTestTool{name: "script_exec", output: `{"status":"ok"}`})
	registry.Register(&workflowTestTool{name: "web_search", output: `{"status":"ok"}`})
	runtimeFactory := &captureRuntimeOverrideFactory{
		baseConfig: bridgeconfig.Config{
			MaxTurns: 4,
			ToolSelector: bridgeconfig.ToolSelectorConfig{
				Allowlist: []string{"script_exec", "web_search"},
			},
			ToolSearch: bridgeconfig.ToolSearchConfig{
				IdleTurns: 3,
			},
		},
		completer: completer,
		registry:  registry,
	}
	service.runtimeFactory = runtimeFactory
	service.agentRunner = NewSessionAgentRunner(
		runtimeFactory,
		service.configStore,
		service.sessionStore,
		service.runRegistry,
	)

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		Message:  "use narrowed tool set",
		TaskKind: taskKindAgentMessage,
		RuntimeOverrides: &TaskRuntimeOverrides{
			ToolAllowlist: []string{"script_exec"},
		},
		IntervalSeconds: 60,
	}, "trace-task-runtime-override-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-task-runtime-override-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if len(run.Run.NodeResults) != 1 {
		t.Fatalf("expected one agent_message node result, got %#v", run.Run.NodeResults)
	}
	node := run.Run.NodeResults[0]
	if node.NodeID != taskKindAgentMessage || node.NodeType != taskKindAgentMessage || node.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected agent node identity: %#v", node)
	}
	input, ok := node.Input.(map[string]any)
	message, hasMessage := input["message"].(string)
	if !ok || !hasMessage || strings.TrimSpace(message) != "use narrowed tool set" {
		t.Fatalf("unexpected agent node input: %#v", node.Input)
	}
	output, ok := node.Output.(map[string]any)
	if !ok {
		t.Fatalf("unexpected agent node output: %#v", node.Output)
	}
	if _, exists := output["response_preview"]; !exists {
		t.Fatalf("expected response_preview in node output: %#v", node.Output)
	}
	if len(completer.requests) != 1 {
		t.Fatalf("expected one completion request, got %d", len(completer.requests))
	}
	names := make([]string, 0, len(completer.requests[0].Tools))
	for _, def := range completer.requests[0].Tools {
		names = append(names, def.Name)
	}
	if strings.Join(names, ",") != "script_exec" {
		t.Fatalf("unexpected tool scope: %v", names)
	}
}

func TestTaskRunNowAppliesRuntimeOverrideProviderPromptAndMaxTurns(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	configureRuntimeOverrideProviders(t, service)

	completer := &workflowTestCompleter{
		response: &llm.CompletionResponse{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "ok"},
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
		Message:  "run with full overrides",
		TaskKind: taskKindAgentMessage,
		RuntimeOverrides: &TaskRuntimeOverrides{
			ProviderName:      "anthropic-main",
			Model:             "claude-3.7",
			SystemPrompt:      "override prompt only",
			ToolAllowlistOnly: boolPointer(true),
			ToolAllowlist:     []string{},
			MaxTurns:          intPointer(1),
		},
		IntervalSeconds: 60,
	}, "trace-task-runtime-override-full-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-task-runtime-override-full-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if len(factory.configs) != 1 {
		t.Fatalf("expected one runtime config build, got %d", len(factory.configs))
	}
	cfg := factory.configs[0]
	if cfg.Provider.Type != llm.ProviderAnthropic || cfg.Provider.Model != "claude-3.7" {
		t.Fatalf("unexpected provider override config: %+v", cfg.Provider)
	}
	if cfg.MaxTurns != 1 {
		t.Fatalf("unexpected max_turns override: %d", cfg.MaxTurns)
	}
	if !cfg.ToolSelector.AllowlistOnly || len(cfg.ToolSelector.Allowlist) != 0 {
		t.Fatalf("unexpected tool selector override: %+v", cfg.ToolSelector)
	}
	if len(completer.requests) != 1 {
		t.Fatalf("expected one completion request, got %d", len(completer.requests))
	}
	if len(completer.requests[0].Messages) == 0 || completer.requests[0].Messages[0].Text != "override prompt only" {
		t.Fatalf("unexpected system prompt override request: %#v", completer.requests[0].Messages)
	}
	if len(completer.requests[0].Tools) != 0 {
		t.Fatalf("expected explicit no-tools request, got %#v", completer.requests[0].Tools)
	}
}

func TestTaskCreateRejectsUnknownRuntimeOverrideProviderModelAndMaxTurns(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	configureRuntimeOverrideProviders(t, service)

	tests := []struct {
		name      string
		overrides *TaskRuntimeOverrides
		want      string
	}{
		{
			name:      "unknown provider",
			overrides: &TaskRuntimeOverrides{ProviderName: "ghost", Model: "gpt-5.4"},
			want:      `provider_name "ghost" is not configured`,
		},
		{
			name:      "model outside provider",
			overrides: &TaskRuntimeOverrides{Model: "claude-3.7"},
			want:      `model "claude-3.7" is not configured for provider "openai-main"`,
		},
		{
			name:      "invalid max turns",
			overrides: &TaskRuntimeOverrides{MaxTurns: intPointer(0)},
			want:      "max_turns must be > 0",
		},
		{
			name:      "missing preset",
			overrides: &TaskRuntimeOverrides{PresetID: "ghost-preset"},
			want:      `preset_id "ghost-preset" is not configured`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			_, code, err := service.executeTaskCreateAction(taskCreateParams{
				Message:          "bad override",
				TaskKind:         taskKindAgentMessage,
				RuntimeOverrides: test.overrides,
				IntervalSeconds:  60,
			}, "trace-task-runtime-override-invalid")
			if err == nil {
				t.Fatal("expected runtime override validation to fail")
			}
			if code != http.StatusBadRequest {
				t.Fatalf("unexpected status code: got %d want %d", code, http.StatusBadRequest)
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestTaskRunNowBuildsSystemPromptFromPresetWithoutMutatingStoredPromptFiles(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	configureRuntimeOverrideProviders(t, service)
	library := []bridgeconfig.SystemPromptLibraryItem{
		{ID: "rule-global", Name: "Rule Global", InsertPoint: bridgeconfig.SystemPromptInsertPointRule, Content: "global rule", Active: true},
		{ID: "rule-preset", Name: "Rule Preset", InsertPoint: bridgeconfig.SystemPromptInsertPointRule, Content: "preset rule", Active: false},
		{ID: "core-global", Name: "Core Global", InsertPoint: bridgeconfig.SystemPromptInsertPointCoreJob, Content: "global core", Active: true},
		{ID: "core-preset", Name: "Core Preset", InsertPoint: bridgeconfig.SystemPromptInsertPointCoreJob, Content: "preset core", Active: false},
		{ID: "context-global", Name: "Context Global", InsertPoint: bridgeconfig.SystemPromptInsertPointContext, Content: "global context", Active: true},
		{ID: "context-preset", Name: "Context Preset", InsertPoint: bridgeconfig.SystemPromptInsertPointContext, Content: "preset context", Active: false},
	}
	if _, err := service.configStore.UpdateSystemPrompts(bridgeconfig.SystemPromptUpdateRequest{
		PromptLibrary: &library,
	}); err != nil {
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

	completer := &workflowTestCompleter{
		response: &llm.CompletionResponse{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "ok"},
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
		Message:  "run with preset",
		TaskKind: taskKindAgentMessage,
		RuntimeOverrides: &TaskRuntimeOverrides{
			PresetID:          preset.ID,
			ToolAllowlistOnly: boolPointer(true),
			ToolAllowlist:     []string{},
		},
		IntervalSeconds: 60,
	}, "trace-task-runtime-override-preset-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-task-runtime-override-preset-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if len(completer.requests) != 1 {
		t.Fatalf("expected one completion request, got %d", len(completer.requests))
	}
	systemPrompt := completer.requests[0].Messages[0].Text
	if !strings.Contains(systemPrompt, "preset rule") ||
		!strings.Contains(systemPrompt, "preset core") ||
		!strings.Contains(systemPrompt, "preset context") {
		t.Fatalf("unexpected preset system prompt: %q", systemPrompt)
	}
	if strings.Contains(systemPrompt, "global rule") || strings.Contains(systemPrompt, "global context") {
		t.Fatalf("expected preset runtime prompt to replace active prompt refs, got %q", systemPrompt)
	}

	files, err := service.configStore.SystemPrompts()
	if err != nil {
		t.Fatalf("reload prompt library: %v", err)
	}
	if activePromptID(files.PromptLibrary, bridgeconfig.SystemPromptInsertPointRule) != "rule-global" {
		t.Fatalf("expected stored rule prompt to remain global, got %+v", files.PromptLibrary)
	}
	if activePromptID(files.PromptLibrary, bridgeconfig.SystemPromptInsertPointCoreJob) != "core-global" {
		t.Fatalf("expected stored core prompt to remain global, got %+v", files.PromptLibrary)
	}
	if activePromptIDs(files.PromptLibrary, bridgeconfig.SystemPromptInsertPointContext)[0] != "context-global" {
		t.Fatalf("expected stored context prompt to remain global, got %+v", files.PromptLibrary)
	}
}

func TestExecuteAgentActionWithRuntimeOverridesRejectsMissingPresetAtRunTime(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	factory := &captureRuntimeOverrideFactory{
		completer: &workflowTestCompleter{
			response: &llm.CompletionResponse{
				Message:      llm.Message{Role: llm.RoleAssistant, Text: "ok"},
				FinishReason: llm.FinishStop,
			},
		},
		registry: tools.NewRegistry(),
	}
	service.runtimeFactory = factory
	service.agentRunner = NewSessionAgentRunner(
		factory,
		service.configStore,
		service.sessionStore,
		service.runRegistry,
	)

	_, err := service.executeAgentActionWithRuntimeOverrides(
		context.Background(),
		agentParams{Message: "run with missing preset"},
		&TaskRuntimeOverrides{PresetID: "ghost-preset"},
		"trace-missing-preset-run",
	)
	if err == nil || !strings.Contains(err.Error(), `preset_id "ghost-preset" is not configured`) {
		t.Fatalf("expected missing preset runtime error, got %v", err)
	}
}

func configureRuntimeOverrideProviders(t *testing.T, service *bridgeService) {
	t.Helper()
	if err := service.configStore.AddProvider(bridgeconfig.ProviderRecord{
		Name:    "openai-main",
		Type:    llm.ProviderOpenAI,
		BaseURL: "https://api.openai.com/v1",
		APIKey:  stringPointer("openai-key"),
		Models:  []string{"gpt-5.4"},
	}); err != nil {
		t.Fatalf("add openai provider: %v", err)
	}
	if err := service.configStore.AddProvider(bridgeconfig.ProviderRecord{
		Name:    "anthropic-main",
		Type:    llm.ProviderAnthropic,
		BaseURL: "https://api.anthropic.com",
		APIKey:  stringPointer("anthropic-key"),
		Models:  []string{"claude-3.7"},
	}); err != nil {
		t.Fatalf("add anthropic provider: %v", err)
	}
	if err := service.configStore.SetActiveProvider("openai-main"); err != nil {
		t.Fatalf("set active provider: %v", err)
	}
}

type captureRuntimeOverrideFactory struct {
	baseConfig bridgeconfig.Config
	configs    []bridgeconfig.Config
	completer  *workflowTestCompleter
	registry   *tools.Registry
	buildError error
}

func (f *captureRuntimeOverrideFactory) Build(store bridgeconfig.Store) (agentRuntimeDependencies, error) {
	if f.buildError != nil {
		return agentRuntimeDependencies{}, f.buildError
	}
	cfg, err := store.Config()
	if err != nil {
		return agentRuntimeDependencies{}, err
	}
	cfg = applyRuntimeOverrideBaseConfig(cfg, f.baseConfig)
	f.configs = append(f.configs, cfg)
	return NewRuntimeDependencies(cfg, f.completer, f.registry, "global prompt", nil), nil
}

func applyRuntimeOverrideBaseConfig(
	cfg bridgeconfig.Config,
	base bridgeconfig.Config,
) bridgeconfig.Config {
	if cfg.MaxTurns == 0 && base.MaxTurns > 0 {
		cfg.MaxTurns = base.MaxTurns
	}
	if len(cfg.ToolSelector.Allowlist) == 0 && len(base.ToolSelector.Allowlist) > 0 {
		cfg.ToolSelector.Allowlist = append([]string(nil), base.ToolSelector.Allowlist...)
	}
	if !cfg.ToolSelector.AllowlistOnly && base.ToolSelector.AllowlistOnly {
		cfg.ToolSelector.AllowlistOnly = true
	}
	if len(cfg.ToolSelector.Blocklist) == 0 && len(base.ToolSelector.Blocklist) > 0 {
		cfg.ToolSelector.Blocklist = append([]string(nil), base.ToolSelector.Blocklist...)
	}
	if cfg.ToolSearch.IdleTurns == 0 && base.ToolSearch.IdleTurns > 0 {
		cfg.ToolSearch.IdleTurns = base.ToolSearch.IdleTurns
	}
	return cfg
}

func activePromptID(
	library []bridgeconfig.SystemPromptLibraryItem,
	insertPoint bridgeconfig.SystemPromptInsertPoint,
) string {
	for _, item := range library {
		if item.InsertPoint == insertPoint && item.Active {
			return item.ID
		}
	}
	return ""
}

func activePromptIDs(
	library []bridgeconfig.SystemPromptLibraryItem,
	insertPoint bridgeconfig.SystemPromptInsertPoint,
) []string {
	ids := make([]string, 0, len(library))
	for _, item := range library {
		if item.InsertPoint == insertPoint && item.Active {
			ids = append(ids, item.ID)
		}
	}
	return ids
}
