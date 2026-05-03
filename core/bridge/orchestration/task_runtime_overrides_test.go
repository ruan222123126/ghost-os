package orchestration

import (
	"net/http"
	"strings"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

func TestNormalizeTaskRuntimeOverrides(t *testing.T) {
	normalized, err := normalizeTaskRuntimeOverrides(nil)
	if err != nil {
		t.Fatalf("normalize nil overrides: %v", err)
	}
	if normalized != nil {
		t.Fatalf("expected nil overrides, got %#v", normalized)
	}

	normalized, err = normalizeTaskRuntimeOverrides(&TaskRuntimeOverrides{
		Model:         " ",
		ToolAllowlist: []string{"  ", "\n"},
	})
	if err != nil {
		t.Fatalf("normalize empty overrides: %v", err)
	}
	if normalized != nil {
		t.Fatalf("expected empty overrides to collapse to nil, got %#v", normalized)
	}

	normalized, err = normalizeTaskRuntimeOverrides(&TaskRuntimeOverrides{
		ProviderName:      " openai-main ",
		Model:             " gpt-5.4 ",
		SystemPrompt:      " be concise ",
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
	if cfg.ToolSearch.IdleTurns == 0 && base.ToolSearch.IdleTurns > 0 {
		cfg.ToolSearch.IdleTurns = base.ToolSearch.IdleTurns
	}
	return cfg
}

func boolPointer(value bool) *bool {
	return &value
}

func intPointer(value int) *int {
	return &value
}

func stringPointer(value string) *string {
	return &value
}
