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
		Model:         " gpt-5.4 ",
		ToolAllowlist: []string{"web_search", "script_exec", "web_search"},
	})
	if err != nil {
		t.Fatalf("normalize valid overrides: %v", err)
	}
	if normalized.Model != "gpt-5.4" {
		t.Fatalf("unexpected model: %#v", normalized)
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
	cfg := bridgeconfig.Config{
		MaxTurns: 4,
		ToolSelector: bridgeconfig.ToolSelectorConfig{
			Allowlist: []string{"script_exec", "web_search"},
		},
		ToolSearch: bridgeconfig.ToolSearchConfig{
			IdleTurns: 3,
		},
	}
	runtimeFactory := proTestRuntimeFactory{
		deps: NewRuntimeDependencies(cfg, completer, registry, "system prompt", nil),
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
