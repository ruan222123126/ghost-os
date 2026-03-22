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

func TestSearchCandidateToolNames_IncludeHiddenStaticToolsWhenToolSearchEnabled(t *testing.T) {
	candidates := SearchCandidateToolNames([]string{
		"ask_human",
		"script_exec",
		"web_search",
		"tfind",
	}, nil, VisibilityOptions{
		ToolSearchEnabled: true,
	})

	if !containsTool(candidates, "script_exec") || !containsTool(candidates, "web_search") {
		t.Fatalf("unexpected tfind candidates: %v", candidates)
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
