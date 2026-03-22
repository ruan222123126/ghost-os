package tools

import (
	"strings"
	"testing"
)

func TestFormatPromptGuidanceForCatalog_UsesScopedToolHints(t *testing.T) {
	registry := NewRegistry()
	for _, name := range []string{"ask_human", "read_and_summarize", "script_exec", "screen_action", ToolSearchToolName} {
		registry.Register(&mockTool{name: name})
	}

	scoped := NewScopedCatalog(registry, []string{"ask_human", "read_and_summarize", "script_exec", ToolSearchToolName})
	guidance := FormatPromptGuidanceForCatalog(scoped)

	for _, snippet := range []string{
		"structured tool schema",
		"`read_and_summarize`",
		"`script_exec`",
		"`ask_human`",
		"`action=search`",
	} {
		if !strings.Contains(guidance, snippet) {
			t.Fatalf("expected guidance to contain %q, got %q", snippet, guidance)
		}
	}
	if strings.Contains(guidance, "mutation {") {
		t.Fatalf("expected native guidance to avoid GraphQL examples, got %q", guidance)
	}
	if strings.Contains(guidance, "`screen_action`") {
		t.Fatalf("expected guidance to exclude hidden tools, got %q", guidance)
	}
}

func TestFormatPromptGuidanceForCatalog_IncludesToolSearchWorkflowOnlyWhenVisible(t *testing.T) {
	registry := NewRegistry()
	for _, name := range []string{"ask_human", ToolSearchToolName} {
		registry.Register(&mockTool{name: name})
	}

	withToolSearch := FormatPromptGuidanceForCatalog(registry)
	for _, snippet := range []string{
		"currently visible tools are insufficient",
		"`action=load`",
		"becomes available next turn",
		"`action=unload`",
	} {
		if !strings.Contains(withToolSearch, snippet) {
			t.Fatalf("expected tool search guidance to contain %q, got %q", snippet, withToolSearch)
		}
	}
	if strings.Contains(withToolSearch, "mutation {") {
		t.Fatalf("expected native tool search guidance to avoid GraphQL examples, got %q", withToolSearch)
	}

	withoutToolSearch := FormatPromptGuidanceForCatalog(NewScopedCatalog(registry, []string{"ask_human"}))
	if strings.Contains(withoutToolSearch, "`action=search`") {
		t.Fatalf("expected tool search guidance to stay hidden, got %q", withoutToolSearch)
	}
}

func TestFormatPromptGuidanceForCatalog_IncludesRSSPipelineOnlyWhenVisible(t *testing.T) {
	registry := NewRegistry()
	for _, name := range []string{"ask_human", "feed_manage"} {
		registry.Register(&mockTool{name: name})
	}

	withRSS := FormatPromptGuidanceForCatalog(registry)
	if !strings.Contains(withRSS, "RSS inbox polling and AI filtering") {
		t.Fatalf("expected RSS guidance when rss tools are visible, got %q", withRSS)
	}

	withoutRSS := FormatPromptGuidanceForCatalog(NewScopedCatalog(registry, []string{"ask_human"}))
	if strings.Contains(withoutRSS, "RSS inbox polling and AI filtering") {
		t.Fatalf("expected RSS guidance to stay hidden when rss tools are not visible, got %q", withoutRSS)
	}
}

func TestFormatPromptGuidanceForCatalog_AskHumanRequiresCustomOption(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&mockTool{name: "ask_human"})

	guidance := FormatPromptGuidanceForCatalog(registry)
	if !strings.Contains(guidance, "final option must allow custom input") {
		t.Fatalf("expected ask_human guidance to require a custom option, got %q", guidance)
	}
	if strings.Contains(guidance, "mutation { ask_human(") {
		t.Fatalf("expected native ask_human guidance to avoid GraphQL examples, got %q", guidance)
	}
}

func TestFormatPromptGuidanceForCatalog_IncludesMemoryWorkflowWhenVisible(t *testing.T) {
	registry := NewRegistry()
	for _, name := range []string{"ask_human", "memory_manage"} {
		registry.Register(&mockTool{name: name})
	}

	guidance := FormatPromptGuidanceForCatalog(registry)
	for _, snippet := range []string{
		"`memory_manage`",
		"stable URI",
		"`create` for the first write",
		"`system://index`",
		"do not guess URIs",
	} {
		if !strings.Contains(guidance, snippet) {
			t.Fatalf("expected memory guidance to contain %q, got %q", snippet, guidance)
		}
	}
	if strings.Contains(guidance, "mutation { memory_manage(") {
		t.Fatalf("expected native memory guidance to avoid GraphQL examples, got %q", guidance)
	}
}

func TestFormatPromptGuidanceForCatalog_WorkspaceGuidanceRequiresBothTools(t *testing.T) {
	registry := NewRegistry()
	for _, name := range []string{"ask_human", "read_and_summarize", "script_exec"} {
		registry.Register(&mockTool{name: name})
	}

	withBoth := FormatPromptGuidanceForCatalog(registry)
	if !strings.Contains(withBoth, "Use `read_and_summarize` for broad local triage, then use `script_exec`") {
		t.Fatalf("expected combined workspace guidance to mention both tools, got %q", withBoth)
	}
	if strings.Contains(withBoth, "mutation { script_exec(") {
		t.Fatalf("expected native workspace guidance to avoid GraphQL examples, got %q", withBoth)
	}

	onlyScriptExec := FormatPromptGuidanceForCatalog(NewScopedCatalog(registry, []string{"ask_human", "script_exec"}))
	if strings.Contains(onlyScriptExec, "`script_exec` for exact workspace reads") ||
		strings.Contains(onlyScriptExec, "`read_and_summarize` for broad local triage") {
		t.Fatalf("expected workspace guidance to stay hidden when read_and_summarize is not visible, got %q", onlyScriptExec)
	}

	onlyReadAndSummarize := FormatPromptGuidanceForCatalog(NewScopedCatalog(registry, []string{"ask_human", "read_and_summarize"}))
	if strings.Contains(onlyReadAndSummarize, "`read_and_summarize` for broad local triage") ||
		strings.Contains(onlyReadAndSummarize, "broad multi-file triage") {
		t.Fatalf("expected workspace guidance to stay hidden when script_exec is not visible, got %q", onlyReadAndSummarize)
	}
}

func TestFormatPromptGuidanceForCatalog_NativeCatalogOmitsGraphQLSyntax(t *testing.T) {
	registry := NewRegistry()
	for _, name := range []string{
		AskHumanToolName,
		ToolSearchToolName,
		"read_and_summarize",
		"script_exec",
	} {
		registry.Register(&mockTool{name: name})
	}

	guidance := FormatPromptGuidanceForCatalog(registry)
	for _, snippet := range []string{
		"mutation {",
		"`tfind(action: search)`",
		"ask_human(prompt:",
		"script_exec(script:",
	} {
		if strings.Contains(guidance, snippet) {
			t.Fatalf("expected native guidance to omit %q, got %q", snippet, guidance)
		}
	}
}

func TestFormatPromptGuidanceForCatalog_GraphQLHiddenCatalogKeepsUsageHints(t *testing.T) {
	registry := NewRegistry()
	for _, name := range []string{
		AskHumanToolName,
		ToolSearchToolName,
		"memory_manage",
		"screen_action",
		computerUseToolName,
		"browser_control",
		"script_exec",
	} {
		registry.Register(&mockTool{name: name})
	}

	guidance := FormatPromptGuidanceForCatalog(NewStructuredToolHiddenCatalog(registry))

	for _, snippet := range []string{
		"GraphQL tool schema",
		"`ask_human` only when blocked",
		"Minimal `ask_human` options example",
		"`tfind(action: search)`",
		`mutation { tfind(action: load, tool_names: ["browser_control"]) }`,
		"`memory_manage` only for explicit long-term notes",
		"Minimal `memory_manage` create example",
		`mutation { memory_manage(operation: create, uri: "user://preferences/editor", content: "Prefer vim keybindings") }`,
		"`screen_action.click_text`",
		"`computer_use` only for desktop visual tasks",
		"`browser_control` for browser tasks",
		"`script_exec` for scriptable local operations",
	} {
		if !strings.Contains(guidance, snippet) {
			t.Fatalf("expected graphql guidance to contain %q, got %q", snippet, guidance)
		}
	}
	if strings.Contains(guidance, "structured tool schema") {
		t.Fatalf("expected graphql hidden catalog to avoid structured schema wording, got %q", guidance)
	}
}

func TestFormatPromptGuidanceForCatalog_IncludesWebRooterHintWhenVisible(t *testing.T) {
	registry := NewRegistry()
	for _, name := range []string{AskHumanToolName, webRooterToolName, "web_search"} {
		registry.Register(&mockTool{name: name})
	}

	guidance := FormatPromptGuidanceForCatalog(registry)
	if !strings.Contains(guidance, "`web_rooter`") {
		t.Fatalf("expected web_rooter guidance, got %q", guidance)
	}
	if !strings.Contains(guidance, "provide every action parameter explicitly in `params`") {
		t.Fatalf("expected explicit params guidance, got %q", guidance)
	}
	if !strings.Contains(guidance, "Prefer `web_rooter` when the task needs citations") {
		t.Fatalf("expected citation-first web_rooter guidance, got %q", guidance)
	}
	if !strings.Contains(guidance, "Use `web_search` for lighter real-time web lookups") {
		t.Fatalf("expected quick lookup web_search guidance, got %q", guidance)
	}
}
