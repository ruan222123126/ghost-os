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
		"`tfind(action=\"search\")`",
	} {
		if !strings.Contains(guidance, snippet) {
			t.Fatalf("expected guidance to contain %q, got %q", snippet, guidance)
		}
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
		"`tfind(action=\"load\")`",
		"becomes available next turn",
		"`tfind(action=\"unload\")`",
	} {
		if !strings.Contains(withToolSearch, snippet) {
			t.Fatalf("expected tool search guidance to contain %q, got %q", snippet, withToolSearch)
		}
	}

	withoutToolSearch := FormatPromptGuidanceForCatalog(NewScopedCatalog(registry, []string{"ask_human"}))
	if strings.Contains(withoutToolSearch, "`tfind(action=\"search\")`") {
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
	if !strings.Contains(guidance, "allows custom input") {
		t.Fatalf("expected ask_human guidance to require custom input option, got %q", guidance)
	}
}

func TestFormatPromptGuidanceForCatalog_IncludesGraphQLWorkflowWhenVisible(t *testing.T) {
	registry := NewRegistry()
	for _, name := range []string{"ask_human", "graphql_schema_lookup", "graphql_mutation"} {
		registry.Register(&mockTool{name: name})
	}

	guidance := FormatPromptGuidanceForCatalog(registry)
	for _, snippet := range []string{
		"`graphql_schema_lookup`",
		"`graphql_mutation(action=\"prepare\")`",
		"`intent_id`",
	} {
		if !strings.Contains(guidance, snippet) {
			t.Fatalf("expected guidance to contain %q, got %q", snippet, guidance)
		}
	}
}
