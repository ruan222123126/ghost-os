package tools

import "testing"

func TestStaticVisibleToolNames_UsesOnlyAllowlistedResidentTools(t *testing.T) {
	visible := StaticVisibleToolNames([]string{
		"ask_human",
		"script_exec",
		"web_search",
	}, VisibilityOptions{Allowlist: []string{"script_exec"}})

	if !containsTool(visible, "script_exec") {
		t.Fatalf("expected allowlisted tool to remain visible, got %v", visible)
	}
	if containsTool(visible, "web_search") || containsTool(visible, "ask_human") {
		t.Fatalf("expected non-allowlisted tools to stay hidden, got %v", visible)
	}
}

func TestStaticVisibleToolNames_EmptyAllowlistHasNoResidents(t *testing.T) {
	visible := StaticVisibleToolNames([]string{
		"ask_human",
		"script_exec",
		"web_search",
	}, VisibilityOptions{})

	if len(visible) != 0 {
		t.Fatalf("expected no resident tools without allowlist, got %v", visible)
	}
}

func TestSelectorStaticToolNames_ReturnAllNonBlockedToolsOutsideStrictMode(t *testing.T) {
	visible := SelectorStaticToolNames([]string{
		"ask_human",
		"script_exec",
		"web_search",
	}, VisibilityOptions{
		Allowlist: []string{"ask_human"},
		Blocklist: []string{"web_search"},
	})

	if !containsTool(visible, "ask_human") || !containsTool(visible, "script_exec") {
		t.Fatalf("expected selector-visible tools to include non-blocked tools, got %v", visible)
	}
	if containsTool(visible, "web_search") {
		t.Fatalf("expected selector-visible tools to honor blocklist, got %v", visible)
	}
}

func TestSelectorStaticToolNames_StrictModeUsesAllowlist(t *testing.T) {
	visible := SelectorStaticToolNames([]string{
		"ask_human",
		"script_exec",
		"web_search",
	}, VisibilityOptions{
		AllowlistOnly: true,
		Allowlist:     []string{"ask_human"},
	})

	if !containsTool(visible, "ask_human") {
		t.Fatalf("expected allowlisted tool to remain visible, got %v", visible)
	}
	if containsTool(visible, "script_exec") || containsTool(visible, "web_search") {
		t.Fatalf("expected strict mode to hide non-allowlisted tools, got %v", visible)
	}
}

func TestSearchCandidateToolNames_IncludeHiddenStaticToolsWhenToolSearchEnabled(t *testing.T) {
	candidates := SearchCandidateToolNames([]string{
		"ask_human",
		"script_exec",
		"web_search",
		"sfind",
	}, nil, VisibilityOptions{
		ToolSearchEnabled: true,
	})

	if !containsTool(candidates, "script_exec") || !containsTool(candidates, "web_search") {
		t.Fatalf("unexpected sfind candidates: %v", candidates)
	}
}

func TestSearchCandidateToolNames_ExcludeResidentAndBlockedTools(t *testing.T) {
	candidates := SearchCandidateToolNames([]string{
		"ask_human",
		"script_exec",
		"web_search",
		"sfind",
	}, nil, VisibilityOptions{
		Allowlist: []string{"ask_human"},
		Blocklist: []string{"web_search"},
	})

	if containsTool(candidates, "ask_human") || containsTool(candidates, "web_search") {
		t.Fatalf("expected resident and blocked tools to stay out of search candidates, got %v", candidates)
	}
	if !containsTool(candidates, "script_exec") {
		t.Fatalf("expected hidden non-blocked tools to remain searchable, got %v", candidates)
	}
}

func containsTool(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
}
