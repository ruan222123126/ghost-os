package orchestration

import (
	"context"
	"testing"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/tools"
)

func TestNormalizeConfiguredToolLists_RejectsOverlap(t *testing.T) {
	_, _, err := normalizeConfiguredToolLists([]string{"script_exec"}, []string{"script_exec"})
	if err == nil {
		t.Fatal("expected overlap error")
	}
}

func TestNormalizeConfiguredToolLists_RejectsUnknownTool(t *testing.T) {
	_, _, err := normalizeConfiguredToolLists([]string{"ghost_tool"}, nil)
	if err == nil {
		t.Fatal("expected unknown tool error")
	}
}

func TestNormalizeConfiguredToolLists_AllowsCodexCLI(t *testing.T) {
	allowlist, blocklist, err := normalizeConfiguredToolLists([]string{"codex_cli"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(allowlist) != 1 || allowlist[0] != "codex_cli" {
		t.Fatalf("unexpected allowlist: %v", allowlist)
	}
	if len(blocklist) != 0 {
		t.Fatalf("unexpected blocklist: %v", blocklist)
	}
}

func TestNormalizeConfiguredToolLists_AllowsBlockingAskHumanAndToolSearch(t *testing.T) {
	allowlist, blocklist, err := normalizeConfiguredToolLists(nil, []string{"ask_human", "script_exec", "tfind"})
	if err != nil {
		t.Fatalf("normalizeConfiguredToolLists: %v", err)
	}
	if len(allowlist) != 0 {
		t.Fatalf("unexpected allowlist: %v", allowlist)
	}
	expected := []string{"ask_human", "script_exec", "tfind"}
	if len(blocklist) != len(expected) {
		t.Fatalf("unexpected blocklist: %v", blocklist)
	}
	for index, name := range expected {
		if blocklist[index] != name {
			t.Fatalf("unexpected blocklist at %d: got %v want %v", index, blocklist, expected)
		}
	}
}

func TestToolSelectionPolicy_ApplyAddsAllowlistAndHonorsBlocklist(t *testing.T) {
	policy := bridgeruntime.NewToolSelectionPolicy(bridgeconfig.Config{
		ToolSelector: bridgeconfig.ToolSelectorConfig{
			Allowlist: []string{"send_file"},
			Blocklist: []string{"script_exec"},
		},
	})

	selected := policy.Apply([]string{"ask_human", "script_exec", "send_file", "web_search"}, []string{"script_exec", "web_search"})
	expected := []string{"send_file", "web_search"}
	if len(selected) != len(expected) {
		t.Fatalf("unexpected tool count: got %v want %v", selected, expected)
	}
	for index, name := range expected {
		if selected[index] != name {
			t.Fatalf("unexpected selection at %d: got %v want %v", index, selected, expected)
		}
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_HasNoResidentToolsWithoutAllowlist(t *testing.T) {
	preparer := &sessionTurnPreparer{}
	deps := newRunnerTestDeps(bridgeconfig.Config{ToolSelector: bridgeconfig.ToolSelectorConfig{Blocklist: []string{"script_exec"}}, MaxTurns: 6})

	catalog, prompt, err := preparer.selectToolsForTurn(context.Background(), deps, nil, agent.NewHistory("system prompt"), "read config", false, "trace-policy-disabled")
	if err != nil {
		t.Fatalf("selectToolsForTurn returned error: %v", err)
	}
	for _, name := range []string{"script_exec", "ask_human", "send_file", "web_search"} {
		if catalog.Get(name) != nil {
			t.Fatalf("expected %q to stay hidden without allowlist, got visible catalog", name)
		}
	}
	if prompt != "" {
		t.Fatalf("expected empty prompt override, got %q", prompt)
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_AllowlistDefinesResidentToolsWithoutAllowlistOnly(t *testing.T) {
	preparer := &sessionTurnPreparer{}
	deps := newRunnerTestDeps(bridgeconfig.Config{
		ToolSelector: bridgeconfig.ToolSelectorConfig{
			Allowlist: []string{"script_exec"},
		},
		MaxTurns: 6,
	})

	catalog, _, err := preparer.selectToolsForTurn(context.Background(), deps, nil, agent.NewHistory("system prompt"), "read config", false, "trace-policy-allowlist")
	if err != nil {
		t.Fatalf("selectToolsForTurn returned error: %v", err)
	}
	if catalog.Get("script_exec") == nil {
		t.Fatal("expected allowlisted tool to remain available")
	}
	if catalog.Get("ask_human") != nil || catalog.Get("web_search") != nil || catalog.Get("send_file") != nil {
		t.Fatalf("expected non-allowlisted tools to stay hidden")
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_AppliesAllowlistToSubset(t *testing.T) {
	selector := &fakeSelectorEngine{result: bridgeruntime.ToolSelectorResult{Mode: "subset", Tools: []string{"send_file"}, Confidence: 0.9}}
	preparer := &sessionTurnPreparer{selectorFactory: func(bridgeconfig.Config, tools.ToolCatalog) bridgeruntime.SelectorEngine { return selector }}
	deps := newRunnerTestDeps(bridgeconfig.Config{
		ToolSelector: bridgeconfig.ToolSelectorConfig{
			Enabled:   true,
			Mode:      "llm",
			Allowlist: []string{"ask_human"},
		},
		MaxTurns: 6,
	})

	catalog, prompt, err := preparer.selectToolsForTurn(context.Background(), deps, nil, agent.NewHistory("system prompt"), "read config", false, "trace-policy-subset")
	if err != nil {
		t.Fatalf("selectToolsForTurn returned error: %v", err)
	}
	if catalog.Get("ask_human") == nil || catalog.Get("send_file") == nil {
		t.Fatal("expected resident and selector-selected tools to remain in scoped subset")
	}
	for _, name := range []string{"web_search", "script_exec", "tfind"} {
		if catalog.Get(name) != nil {
			t.Fatalf("expected %q to stay hidden outside scoped subset", name)
		}
	}
	if prompt == "" {
		t.Fatal("expected prompt override for scoped subset")
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_PassesSelectorVisibleCatalogOutsideStrictMode(t *testing.T) {
	var available []string
	preparer := &sessionTurnPreparer{
		selectorFactory: func(_ bridgeconfig.Config, catalog tools.ToolCatalog) bridgeruntime.SelectorEngine {
			available = toolCatalogNames(catalog)
			return &fakeSelectorEngine{result: bridgeruntime.ToolSelectorResult{Mode: "all"}}
		},
	}
	deps := newRunnerTestDeps(bridgeconfig.Config{
		ToolSelector: bridgeconfig.ToolSelectorConfig{
			Enabled:   true,
			Mode:      "llm",
			Allowlist: []string{"ask_human"},
			Blocklist: []string{"script_exec"},
		},
		MaxTurns: 6,
	})

	_, _, err := preparer.selectToolsForTurn(context.Background(), deps, nil, agent.NewHistory("system prompt"), "read config", false, "trace-policy-visible")
	if err != nil {
		t.Fatalf("selectToolsForTurn returned error: %v", err)
	}
	if containsToolName(available, "script_exec") {
		t.Fatalf("expected selector-visible catalog to exclude blocked tool, got %v", available)
	}
	for _, name := range []string{"ask_human", "send_file", "web_search"} {
		if !containsToolName(available, name) {
			t.Fatalf("expected selector-visible catalog to include %q outside strict mode, got %v", name, available)
		}
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_AllowlistOnlyScopesVisibleTools(t *testing.T) {
	var available []string
	preparer := &sessionTurnPreparer{
		selectorFactory: func(_ bridgeconfig.Config, catalog tools.ToolCatalog) bridgeruntime.SelectorEngine {
			available = toolCatalogNames(catalog)
			return &fakeSelectorEngine{result: bridgeruntime.ToolSelectorResult{Mode: "all"}}
		},
	}
	deps := newRunnerTestDeps(bridgeconfig.Config{
		ToolSelector: bridgeconfig.ToolSelectorConfig{
			Enabled:       true,
			Mode:          "llm",
			AllowlistOnly: true,
			Allowlist:     []string{"script_exec"},
		},
		MaxTurns: 6,
	})

	catalog, _, err := preparer.selectToolsForTurn(context.Background(), deps, nil, agent.NewHistory("system prompt"), "read config", false, "trace-policy-allowlist-only")
	if err != nil {
		t.Fatalf("selectToolsForTurn returned error: %v", err)
	}
	if catalog.Get("script_exec") == nil {
		t.Fatal("expected allowlisted tool to remain available")
	}
	if catalog.Get("web_search") != nil || catalog.Get("send_file") != nil {
		t.Fatal("expected non-allowlisted tools to be hidden")
	}
	if catalog.Get("ask_human") != nil {
		t.Fatal("expected ask_human to stay hidden when not allowlisted")
	}
	if len(available) != 1 || available[0] != "script_exec" {
		t.Fatalf("expected selector-visible catalog to stay strict, got %v", available)
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_ToolSearchScopesVisibleTools(t *testing.T) {
	preparer := &sessionTurnPreparer{}
	deps := newRunnerTestDeps(bridgeconfig.Config{
		ToolSelector: bridgeconfig.ToolSelectorConfig{
			Allowlist: []string{"send_file", "tfind"},
		},
		ToolSearch: bridgeconfig.ToolSearchConfig{
			Enabled:   true,
			IdleTurns: 3,
		},
		MaxTurns: 6,
	})

	catalog, _, err := preparer.selectToolsForTurn(context.Background(), deps, nil, agent.NewHistory("system prompt"), "find tools", false, "trace-policy-tool-search")
	if err != nil {
		t.Fatalf("selectToolsForTurn returned error: %v", err)
	}
	for _, name := range []string{"send_file", "tfind"} {
		if catalog.Get(name) == nil {
			t.Fatalf("expected %q to remain visible", name)
		}
	}
	for _, name := range []string{"ask_human", "script_exec", "web_search"} {
		if catalog.Get(name) != nil {
			t.Fatalf("expected %q to stay hidden until dynamically loaded", name)
		}
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_CanSelectNonResidentToolsWithoutAllowlist(t *testing.T) {
	selector := &fakeSelectorEngine{result: bridgeruntime.ToolSelectorResult{Mode: "subset", Tools: []string{"script_exec"}, Confidence: 0.9}}
	preparer := &sessionTurnPreparer{selectorFactory: func(bridgeconfig.Config, tools.ToolCatalog) bridgeruntime.SelectorEngine { return selector }}
	deps := newRunnerTestDeps(bridgeconfig.Config{
		ToolSelector: bridgeconfig.ToolSelectorConfig{
			Enabled: true,
			Mode:    "llm",
		},
		MaxTurns: 6,
	})

	catalog, prompt, err := preparer.selectToolsForTurn(context.Background(), deps, nil, agent.NewHistory("system prompt"), "read config", false, "trace-policy-empty-resident-subset")
	if err != nil {
		t.Fatalf("selectToolsForTurn returned error: %v", err)
	}
	if catalog.Get("script_exec") == nil {
		t.Fatal("expected selector to enable non-resident tool")
	}
	for _, name := range []string{"ask_human", "send_file", "web_search"} {
		if catalog.Get(name) != nil {
			t.Fatalf("expected %q to stay hidden outside scoped subset", name)
		}
	}
	if prompt == "" {
		t.Fatal("expected prompt override for scoped subset")
	}
}

func containsToolName(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
}
