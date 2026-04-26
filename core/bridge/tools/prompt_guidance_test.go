package tools

import (
	"strings"
	"testing"
)

func TestFormatPromptGuidanceForCatalog_UsesScopedToolHints(t *testing.T) {
	registry := NewRegistry()
	for _, name := range []string{"ask_human", "script_exec", "screen_action", ToolSearchToolName} {
		registry.Register(&mockTool{name: name})
	}

	scoped := NewScopedCatalog(registry, []string{"ask_human", "script_exec", ToolSearchToolName})
	guidance := FormatPromptGuidanceForCatalog(scoped)

	for _, snippet := range []string{
		"structured tool schema",
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

func TestFormatPromptGuidanceForCatalog_IncludesScriptExecUsageHintsWhenVisible(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&mockTool{name: "script_exec"})

	guidance := FormatPromptGuidanceForCatalog(registry)
	for _, snippet := range []string{
		"injected `tools` object",
		"Do not use `import tools` or `from tools...`",
		"prefer `tools.search_files` before shelling out with `tools.bash_exec`",
		"Do not call blocked builtins (`open`, `eval`, `exec`, `compile`, `input`)",
		"Print concise, structured output",
	} {
		if !strings.Contains(guidance, snippet) {
			t.Fatalf("expected script_exec guidance to contain %q, got %q", snippet, guidance)
		}
	}
}

func TestFormatPromptGuidanceForCatalog_NativeCatalogOmitsGraphQLSyntax(t *testing.T) {
	registry := NewRegistry()
	for _, name := range []string{
		AskHumanToolName,
		ToolSearchToolName,
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
		"screen_action",
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
		`<t:ID>{"action":"load","tool_names":["web_search"]}</t>`,
		"same user turn on the next completion",
		"`tfind(action: list)` only to inspect the current dynamic tool/skill load state",
		"Never repeat or fabricate `[TOOL_TAG_RESULT]`",
		"`screen_action.click_text`",
		"use plain Python plus the injected `tools` object",
		"Do not use `import tools` or `from tools...`",
		"prefer `tools.search_files` before shelling out with `tools.bash_exec`",
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

func TestFormatPromptGuidanceForCatalog_IncludesCodexActionHints(t *testing.T) {
	registry := NewRegistry()
	for _, name := range []string{"codex_cli"} {
		registry.Register(&mockTool{name: name})
	}

	guidance := FormatPromptGuidanceForCatalog(registry)
	for _, snippet := range []string{
		"`codex_cli` `op` must be one of: start, resume, fork, status (not `exec`).",
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
		"`screen_action.click_icon`",
		"direct click and skips template matching",
	} {
		if !strings.Contains(guidance, snippet) {
			t.Fatalf("expected screen_action guidance to contain %q, got %q", snippet, guidance)
		}
	}
}

func TestFormatPromptGuidanceForCatalog_IncludesScreenControlHints(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&mockTool{name: screenControlToolName})

	guidance := FormatPromptGuidanceForCatalog(registry)
	for _, snippet := range []string{
		"`screen_control` is the unified atomic screen entrypoint",
		"`screen_control` atomic action must be one of: screenshot, ocr_scan, click_text, find_icon, click_icon, mouse_position, text_input.",
		"`mode=\"atomic\"`",
		"`action=\"click_icon\"`",
		"`action=\"text_input\"`",
		"direct click and skips template matching",
	} {
		if !strings.Contains(guidance, snippet) {
			t.Fatalf("expected screen_control guidance to contain %q, got %q", snippet, guidance)
		}
	}
}
