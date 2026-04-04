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
		"Do not use `tfind` for greetings",
		"`action=load`",
		"same user turn on the next completion",
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

func TestFormatPromptGuidanceForCatalog_IncludesScriptExecUsageHintsWhenVisible(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&mockTool{name: "script_exec"})

	guidance := FormatPromptGuidanceForCatalog(registry)
	for _, snippet := range []string{
		"injected `tools` object",
		"Do not use `import tools` or `from tools...`",
		"Do not call blocked builtins (`open`, `eval`, `exec`, `compile`, `input`)",
		"Print concise, structured output",
	} {
		if !strings.Contains(guidance, snippet) {
			t.Fatalf("expected script_exec guidance to contain %q, got %q", snippet, guidance)
		}
	}
	if strings.Contains(guidance, "`read_and_summarize` for broad local triage") {
		t.Fatalf("expected workspace-combo guidance to stay hidden when read_and_summarize is not visible, got %q", guidance)
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
		"current tool id list",
		"`ask_human` only when blocked",
		"Minimal `ask_human` tag example",
		`<t:ID>{"action":"search","query":"..."}</t>`,
		"Do not use `tfind` for greetings",
		"keep the assistant message focused on tool tags",
		`<t:ID>{"action":"load","tool_names":["browser_control"]}</t>`,
		"same user turn on the next completion",
		"`tfind(action: list)` only to inspect the current dynamic tool load state",
		"Never repeat or fabricate `[TOOL_TAG_RESULT]`",
		"`memory_manage` only for explicit long-term notes",
		"Minimal `memory_manage` create tag example",
		`<t:ID>{"operation":"create","uri":"user://preferences/editor","content":"Prefer vim keybindings"}</t>`,
		"`screen_action.click_text`",
		"`computer_use` only for desktop visual tasks",
		"`browser_control` for browser tasks",
		"`script_exec` for scriptable local operations",
		"Do not use `import tools` or `from tools...`",
		"Do not call blocked builtins (`open`, `eval`, `exec`, `compile`, `input`)",
		"Minimal `script_exec` tag example",
	} {
		if !strings.Contains(guidance, snippet) {
			t.Fatalf("expected graphql guidance to contain %q, got %q", snippet, guidance)
		}
	}
	if strings.Contains(guidance, "GraphQL tool schema") {
		t.Fatalf("expected hidden catalog guidance to avoid graphql schema wording, got %q", guidance)
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

func TestFormatPromptGuidanceForCatalog_IncludesBrowserControlActionHints(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&mockTool{name: "browser_control"})

	guidance := FormatPromptGuidanceForCatalog(registry)
	for _, snippet := range []string{
		"`browser_control` action must be one of",
		"`action=\"goto\"`",
		"do not use `navigate`",
		"no standalone browser `wait` action",
		"`params.wait` (`none|dom|load`)",
		"do not use `extract`",
	} {
		if !strings.Contains(guidance, snippet) {
			t.Fatalf("expected browser_control guidance to contain %q, got %q", snippet, guidance)
		}
	}
}

func TestFormatPromptGuidanceForCatalog_IncludesTaskFeedAndCodexActionHints(t *testing.T) {
	registry := NewRegistry()
	for _, name := range []string{"task_manage", "feed_manage", "codex_cli"} {
		registry.Register(&mockTool{name: name})
	}

	guidance := FormatPromptGuidanceForCatalog(registry)
	for _, snippet := range []string{
		"`task_manage` operation must be one of: create, update, delete, list, get.",
		"`feed_manage` operation must be one of: subscribe, list, update, unsubscribe.",
		"`codex_cli` `op` must be one of: start, resume, fork, status (not `exec`).",
		"`task_manage` requires `id` for update/delete/get",
		"`feed_manage` requires `url` for subscribe",
		"`codex_cli` requires `prompt` for start/resume/fork",
	} {
		if !strings.Contains(guidance, snippet) {
			t.Fatalf("expected action guidance to contain %q, got %q", snippet, guidance)
		}
	}
}

func TestFormatPromptGuidanceForCatalog_IncludesScreenActionList(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&mockTool{name: "screen_action"})

	guidance := FormatPromptGuidanceForCatalog(registry)
	for _, snippet := range []string{
		"`screen_action` action must be one of: screenshot, ocr_scan, click_text, find_icon, click_icon.",
		"`screen_action.click_text`",
	} {
		if !strings.Contains(guidance, snippet) {
			t.Fatalf("expected screen_action guidance to contain %q, got %q", snippet, guidance)
		}
	}
}
