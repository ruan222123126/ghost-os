package tools

import (
	"strings"
	"testing"
)

func TestGetToolMetadata_CoversExpectedTools(t *testing.T) {
	metadata := GetToolMetadata()
		expected := []string{
			"list_files",
			"read_file",
			"read_and_summarize",
			"search_files",
			"apply_diff",
			"bash_exec",
			"script_exec",
			"codex_cli",
			"web_search",
			"feed_subscribe",
			"feed_list",
			"feed_update",
			"feed_unsubscribe",
			"rss_fetch",
			"memory_manage",
			"screen_action",
			"browser_control",
			"text_input",
			"task_manage",
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

func TestFormatMetadataForSelector_ListsAllTools(t *testing.T) {
	formatted := FormatMetadataForSelector()
	if strings.TrimSpace(formatted) == "" {
		t.Fatal("formatted metadata should not be empty")
	}

	for _, item := range GetToolMetadata() {
		if !strings.Contains(formatted, item.Name) {
			t.Fatalf("formatted metadata should contain %q", item.Name)
		}
	}
	if !strings.Contains(formatted, "always_on=true") {
		t.Fatal("formatted metadata should include always_on marker")
	}
}

func TestFormatMetadataForCatalog_FiltersToVisibleTools(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&mockTool{name: "read_file"})
	registry.Register(&mockTool{name: "ask_human"})

	formatted := FormatMetadataForCatalog(registry)
	if strings.Contains(formatted, "bash_exec") {
		t.Fatalf("formatted metadata should exclude hidden tools: %q", formatted)
	}
	for _, name := range []string{"read_file", "ask_human"} {
		if !strings.Contains(formatted, name) {
			t.Fatalf("formatted metadata should contain %q", name)
		}
	}
}
