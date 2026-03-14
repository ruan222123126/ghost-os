package orchestration

import (
	"context"
	"testing"

	"ghost-os/bridge/agent"
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

func TestNormalizeConfiguredToolLists_IgnoresAskHumanBlocklist(t *testing.T) {
	allowlist, blocklist, err := normalizeConfiguredToolLists(nil, []string{"ask_human", "script_exec", "tfind"})
	if err != nil {
		t.Fatalf("normalizeConfiguredToolLists: %v", err)
	}
	if len(allowlist) != 0 {
		t.Fatalf("unexpected allowlist: %v", allowlist)
	}
	if len(blocklist) != 1 || blocklist[0] != "script_exec" {
		t.Fatalf("unexpected blocklist: %v", blocklist)
	}
}

func TestToolSelectionPolicy_ApplyAddsAllowlistAndHonorsBlocklist(t *testing.T) {
	policy := newToolSelectionPolicy(Config{
		ToolSelector: ToolSelectorConfig{
			Allowlist: []string{"send_file"},
			Blocklist: []string{"script_exec"},
		},
	})

	selected := policy.apply([]string{"ask_human", "script_exec", "send_file", "web_search"}, []string{"script_exec", "web_search"})
	expected := []string{"ask_human", "send_file", "web_search"}
	if len(selected) != len(expected) {
		t.Fatalf("unexpected tool count: got %v want %v", selected, expected)
	}
	for index, name := range expected {
		if selected[index] != name {
			t.Fatalf("unexpected selection at %d: got %v want %v", index, selected, expected)
		}
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_AppliesBlocklistWhenSelectorDisabled(t *testing.T) {
	preparer := &sessionTurnPreparer{}
	deps := newRunnerTestDeps(Config{ToolSelector: ToolSelectorConfig{Blocklist: []string{"script_exec"}}, MaxTurns: 6})

	catalog, prompt, err := preparer.selectToolsForTurn(context.Background(), deps, nil, agent.NewHistory("system prompt"), "read config", false, "trace-policy-disabled")
	if err != nil {
		t.Fatalf("selectToolsForTurn returned error: %v", err)
	}
	if catalog.Get("script_exec") != nil {
		t.Fatal("expected script_exec to be removed by blocklist")
	}
	if catalog.Get("ask_human") == nil {
		t.Fatal("expected ask_human to remain available")
	}
	if prompt != "" {
		t.Fatalf("expected empty prompt override, got %q", prompt)
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_AppliesAllowlistToSubset(t *testing.T) {
	selector := &fakeSelectorEngine{result: ToolSelectorResult{Mode: "subset", Tools: []string{"web_search"}, Confidence: 0.9}}
	preparer := &sessionTurnPreparer{selectorFactory: func(Config, tools.ToolCatalog) selectorEngine { return selector }}
	deps := newRunnerTestDeps(Config{
		ToolSelector: ToolSelectorConfig{
			Enabled:   true,
			Mode:      "llm",
			Allowlist: []string{"send_file"},
		},
		MaxTurns: 6,
	})

	catalog, prompt, err := preparer.selectToolsForTurn(context.Background(), deps, nil, agent.NewHistory("system prompt"), "read config", false, "trace-policy-subset")
	if err != nil {
		t.Fatalf("selectToolsForTurn returned error: %v", err)
	}
	if catalog.Get("web_search") == nil || catalog.Get("send_file") == nil {
		t.Fatalf("expected allowlist tool to be forced into subset")
	}
	if catalog.Get("script_exec") != nil {
		t.Fatal("expected unselected tool to stay hidden")
	}
	if prompt == "" {
		t.Fatal("expected prompt override for scoped subset")
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_PassesPolicyScopedCatalogToSelector(t *testing.T) {
	var available []string
	preparer := &sessionTurnPreparer{
		selectorFactory: func(_ Config, catalog tools.ToolCatalog) selectorEngine {
			available = toolCatalogNames(catalog)
			return &fakeSelectorEngine{result: ToolSelectorResult{Mode: "all"}}
		},
	}
	deps := newRunnerTestDeps(Config{
		ToolSelector: ToolSelectorConfig{
			Enabled:   true,
			Mode:      "llm",
			Blocklist: []string{"script_exec"},
		},
		MaxTurns: 6,
	})

	_, _, err := preparer.selectToolsForTurn(context.Background(), deps, nil, agent.NewHistory("system prompt"), "read config", false, "trace-policy-visible")
	if err != nil {
		t.Fatalf("selectToolsForTurn returned error: %v", err)
	}
	for _, name := range available {
		if name == "script_exec" {
			t.Fatalf("expected selector-visible catalog to exclude blocked tool, got %v", available)
		}
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_AllowlistOnlyScopesVisibleTools(t *testing.T) {
	preparer := &sessionTurnPreparer{}
	deps := newRunnerTestDeps(Config{
		ToolSelector: ToolSelectorConfig{
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
	if catalog.Get("ask_human") == nil {
		t.Fatal("expected ask_human to remain available")
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_ToolSearchScopesVisibleTools(t *testing.T) {
	preparer := &sessionTurnPreparer{}
	deps := newRunnerTestDeps(Config{
		ToolSelector: ToolSelectorConfig{
			Allowlist: []string{"send_file"},
		},
		ToolSearch: ToolSearchConfig{
			Enabled:   true,
			IdleTurns: 3,
		},
		MaxTurns: 6,
	})

	catalog, _, err := preparer.selectToolsForTurn(context.Background(), deps, nil, agent.NewHistory("system prompt"), "find tools", false, "trace-policy-tool-search")
	if err != nil {
		t.Fatalf("selectToolsForTurn returned error: %v", err)
	}
	for _, name := range []string{"ask_human", "send_file", "tfind"} {
		if catalog.Get(name) == nil {
			t.Fatalf("expected %q to remain visible", name)
		}
	}
	for _, name := range []string{"script_exec", "web_search"} {
		if catalog.Get(name) != nil {
			t.Fatalf("expected %q to stay hidden until dynamically loaded", name)
		}
	}
}
