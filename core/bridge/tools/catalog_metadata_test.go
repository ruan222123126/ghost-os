package tools

import (
	"strings"
	"testing"
)

func TestGetToolMetadata_CoversExpectedTools(t *testing.T) {
	metadata := GetToolMetadata()
	expected := []string{
		"read_and_summarize",
		"send_file",
		"set_project_root",
		"script_exec",
		"codex_cli",
		"web_search",
		"graphql_query",
		"graphql_schema_lookup",
		"graphql_mutation",
		"feed_manage",
		"rss_fetch",
		"memory_manage",
		"memory_learned_list",
		"memory_recall_debug",
		"screen_action",
		"browser_control",
		"text_input",
		"task_manage",
		"tfind",
		"ask_human",
	}
	seen := make(map[string]ToolMetadata, len(metadata))
	alwaysOnCount := 0
	for _, item := range metadata {
		if item.Name == "" {
			t.Fatal("tool metadata name should not be empty")
		}
		if item.Domain == "" {
			t.Fatalf("tool metadata domain should not be empty: %+v", item)
		}
		if len(item.Tags) == 0 {
			t.Fatalf("tool metadata tags should not be empty: %+v", item)
		}
		if item.ShortDesc == "" {
			t.Fatalf("tool metadata short description should not be empty: %+v", item)
		}
		seen[item.Name] = item
		if item.AlwaysOn {
			alwaysOnCount++
		}
	}

	for _, name := range expected {
		if _, ok := seen[name]; !ok {
			t.Fatalf("expected metadata for tool %q", name)
		}
	}
	if alwaysOnCount != 1 {
		t.Fatalf("expected exactly one always-on tool, got %d", alwaysOnCount)
	}
	if !seen["ask_human"].AlwaysOn {
		t.Fatal("ask_human should be marked always-on")
	}
}

func TestFormatMetadataForSelector_HidesOnDemandTools(t *testing.T) {
	formatted := FormatMetadataForSelector()
	if strings.TrimSpace(formatted) == "" {
		t.Fatal("formatted metadata should not be empty")
	}

	for _, item := range GetToolMetadata() {
		if item.Name == ToolSearchToolName || item.OnDemand {
			continue
		}
		if !strings.Contains(formatted, item.Name) {
			t.Fatalf("formatted metadata should contain %q", item.Name)
		}
	}
	if strings.Contains(formatted, ToolSearchToolName) {
		t.Fatalf("formatted metadata should exclude %q: %q", ToolSearchToolName, formatted)
	}
	for _, name := range []string{"graphql_query", "graphql_schema_lookup", "graphql_mutation"} {
		if strings.Contains(formatted, name) {
			t.Fatalf("formatted metadata should exclude on-demand tool %q: %q", name, formatted)
		}
	}
	if !strings.Contains(formatted, "always_on=true") {
		t.Fatal("formatted metadata should include always_on marker")
	}
}

func TestFormatMetadataForCatalog_FiltersToVisibleTools(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&mockTool{name: "script_exec"})
	registry.Register(&mockTool{name: "ask_human"})

	formatted := FormatMetadataForCatalog(registry)
	if strings.Contains(formatted, "web_search") {
		t.Fatalf("formatted metadata should exclude hidden tools: %q", formatted)
	}
	for _, name := range []string{"script_exec", "ask_human"} {
		if !strings.Contains(formatted, name) {
			t.Fatalf("formatted metadata should contain %q", name)
		}
	}
}

func TestFormatPromptToolsForCatalog_UsesShortDescriptions(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&mockTool{name: "script_exec"})
	registry.Register(&mockTool{name: ToolSearchToolName})

	formatted := FormatPromptToolsForCatalog(registry)
	if !strings.Contains(formatted, "- script_exec: Run a Python script in sandbox.") {
		t.Fatalf("unexpected prompt tool list: %q", formatted)
	}
	if !strings.Contains(formatted, "- tfind: Find or load optional tools.") {
		t.Fatalf("unexpected prompt tool list: %q", formatted)
	}
}
