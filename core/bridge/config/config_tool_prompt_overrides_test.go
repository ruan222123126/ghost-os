package config

import (
	"strings"
	"testing"
)

func TestNormalizeToolPromptOverridesRejectsUnknownTool(t *testing.T) {
	_, err := normalizeToolPromptOverrides(map[string]string{
		"unknown_tool": "do something",
	})
	if err == nil || !strings.Contains(err.Error(), "unknown tool in tool_prompt_overrides") {
		t.Fatalf("expected unknown tool error, got %v", err)
	}
}

func TestNormalizeToolPromptOverridesTrimsAndDropsEmptyValues(t *testing.T) {
	got, err := normalizeToolPromptOverrides(map[string]string{
		" script_exec ": "  use this tool first  ",
		"web_search":    "   ",
	})
	if err != nil {
		t.Fatalf("normalizeToolPromptOverrides: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("unexpected size: got %d want %d", len(got), 1)
	}
	if got["script_exec"] != "use this tool first" {
		t.Fatalf("unexpected normalized value: got %q", got["script_exec"])
	}
}
