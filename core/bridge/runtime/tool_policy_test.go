package runtime

import (
	"path/filepath"
	"strings"
	"testing"

	"ghost-os/bridge/tools"
)

func TestToolSelectionPolicy_ResidentAndSelectorScopesSplitOutsideStrictMode(t *testing.T) {
	registry := tools.NewRegistry()
	for _, name := range []string{"ask_human", "script_exec", "web_search", "tfind"} {
		registry.Register(&catalogMockTool{name: name})
	}

	policy := newToolSelectionPolicy(Config{
		ToolSelector: ToolSelectorConfig{
			Allowlist: []string{"ask_human"},
			Blocklist: []string{"web_search"},
		},
	})

	resident := tools.CatalogToolNames(policy.residentCatalog(registry))
	if len(resident) != 1 || resident[0] != "ask_human" {
		t.Fatalf("expected resident catalog to contain only allowlisted tools, got %v", resident)
	}

	selector := tools.CatalogToolNames(policy.selectorCatalog(registry))
	if containsToolName(selector, "web_search") {
		t.Fatalf("expected selector catalog to honor blocklist, got %v", selector)
	}
	for _, name := range []string{"ask_human", "script_exec", "tfind"} {
		if !containsToolName(selector, name) {
			t.Fatalf("expected selector catalog to include %q outside strict mode, got %v", name, selector)
		}
	}
}

func TestToolSelectionPolicy_SelectorScopeHonorsAllowlistOnly(t *testing.T) {
	policy := newToolSelectionPolicy(Config{
		ToolSelector: ToolSelectorConfig{
			AllowlistOnly: true,
			Allowlist:     []string{"ask_human"},
		},
	})

	visible := policy.selectorScope([]string{"ask_human", "script_exec", "web_search"})
	if len(visible) != 1 || visible[0] != "ask_human" {
		t.Fatalf("expected strict mode to limit selector scope to allowlist, got %v", visible)
	}
}

func TestToolSelectionPolicy_ApplyKeepsResidentAndSelectedTools(t *testing.T) {
	policy := newToolSelectionPolicy(Config{
		ToolSelector: ToolSelectorConfig{
			Allowlist: []string{"ask_human"},
			Blocklist: []string{"web_search"},
		},
	})

	selected := policy.apply([]string{"ask_human", "script_exec", "web_search"}, []string{"script_exec", "web_search"})
	if len(selected) != 2 {
		t.Fatalf("expected resident and selected tool set, got %v", selected)
	}
	if !containsToolName(selected, "ask_human") || !containsToolName(selected, "script_exec") {
		t.Fatalf("expected resident and selected tools to remain, got %v", selected)
	}
	if containsToolName(selected, "web_search") {
		t.Fatalf("expected blocked tool to stay hidden, got %v", selected)
	}
}

func TestBuildRuntimeSystemPromptUsesResidentCatalog(t *testing.T) {
	tempDir := setupRuntimeFactoryTestEnv(t)
	registry := tools.NewRegistry()
	for _, name := range []string{"ask_human", "script_exec"} {
		registry.Register(&catalogMockTool{name: name})
	}

	prompt, err := buildRuntimeSystemPrompt(Config{
		MaxTurns: 3,
		PromptsDir: filepath.Join(tempDir, "prompts"),
		ToolSelector: ToolSelectorConfig{
			Allowlist: []string{"ask_human"},
		},
	}, registry)
	if err != nil {
		t.Fatalf("buildRuntimeSystemPrompt returned error: %v", err)
	}
	if !strings.Contains(prompt, "`ask_human`") {
		t.Fatalf("expected resident tool guidance in runtime prompt, got %q", prompt)
	}
	if strings.Contains(prompt, "`script_exec`") {
		t.Fatalf("expected non-resident tool guidance to stay hidden, got %q", prompt)
	}
}
