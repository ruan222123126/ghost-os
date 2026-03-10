package app

import (
	"context"
	"testing"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/tools"
)

func TestNormalizeConfiguredToolLists_RejectsOverlap(t *testing.T) {
	_, _, err := normalizeConfiguredToolLists([]string{"read_file"}, []string{"read_file"})
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

func TestNormalizeConfiguredToolLists_IgnoresAskHumanBlocklist(t *testing.T) {
	allowlist, blocklist, err := normalizeConfiguredToolLists(nil, []string{"ask_human", "bash_exec"})
	if err != nil {
		t.Fatalf("normalizeConfiguredToolLists: %v", err)
	}
	if len(allowlist) != 0 {
		t.Fatalf("unexpected allowlist: %v", allowlist)
	}
	if len(blocklist) != 1 || blocklist[0] != "bash_exec" {
		t.Fatalf("unexpected blocklist: %v", blocklist)
	}
}

func TestToolSelectionPolicy_ApplyAddsAllowlistAndHonorsBlocklist(t *testing.T) {
	policy := newToolSelectionPolicy(ToolSelectorConfig{
		Allowlist: []string{"search_files"},
		Blocklist: []string{"bash_exec"},
	})

	selected := policy.apply([]string{"ask_human", "bash_exec", "read_file", "search_files"}, []string{"bash_exec", "read_file"})
	expected := []string{"ask_human", "read_file", "search_files"}
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
	deps := newRunnerTestDeps(Config{ToolSelector: ToolSelectorConfig{Blocklist: []string{"bash_exec"}}, MaxTurns: 6})

	catalog, prompt := preparer.selectToolsForTurn(context.Background(), deps, "runner-policy-disabled", agent.NewHistory("system prompt"), "read config", false, "trace-policy-disabled", runnerSelectorEnv())
	if catalog.Get("bash_exec") != nil {
		t.Fatal("expected bash_exec to be removed by blocklist")
	}
	if catalog.Get("ask_human") == nil {
		t.Fatal("expected ask_human to remain available")
	}
	if prompt != "" {
		t.Fatalf("expected empty prompt override, got %q", prompt)
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_AppliesAllowlistToSubset(t *testing.T) {
	selector := &fakeSelectorEngine{result: ToolSelectorResult{Mode: "subset", Tools: []string{"read_file"}, Confidence: 0.9}}
	preparer := &sessionTurnPreparer{selectorFactory: func(Config, tools.ToolCatalog) selectorEngine { return selector }}
	deps := newRunnerTestDeps(Config{
		ToolSelector: ToolSelectorConfig{
			Enabled:   true,
			Mode:      "llm",
			Allowlist: []string{"search_files"},
		},
		MaxTurns: 6,
	})

	catalog, prompt := preparer.selectToolsForTurn(context.Background(), deps, "runner-policy-subset", agent.NewHistory("system prompt"), "read config", false, "trace-policy-subset", runnerSelectorEnv())
	if catalog.Get("read_file") == nil || catalog.Get("search_files") == nil {
		t.Fatalf("expected allowlist tool to be forced into subset")
	}
	if catalog.Get("bash_exec") != nil {
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
			Blocklist: []string{"bash_exec"},
		},
		MaxTurns: 6,
	})

	preparer.selectToolsForTurn(context.Background(), deps, "runner-policy-visible", agent.NewHistory("system prompt"), "read config", false, "trace-policy-visible", runnerSelectorEnv())
	for _, name := range available {
		if name == "bash_exec" {
			t.Fatalf("expected selector-visible catalog to exclude blocked tool, got %v", available)
		}
	}
}
