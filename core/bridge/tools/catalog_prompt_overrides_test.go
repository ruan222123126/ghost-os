package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestPromptOverrideCatalogAppliesDescriptionOverride(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&mockPromptOverrideTool{name: "script_exec", description: "run scripts"})

	catalog := NewPromptOverrideCatalog(registry, map[string]string{
		"script_exec": "custom prompt for script exec",
	})
	defs := catalog.ToolDefs()
	if len(defs) != 1 {
		t.Fatalf("unexpected tool defs count: %d", len(defs))
	}
	if defs[0].Description != "custom prompt for script exec" {
		t.Fatalf("unexpected overridden description: %q", defs[0].Description)
	}
	if catalog.Get("script_exec") == nil {
		t.Fatal("expected wrapped catalog to preserve Get behavior")
	}
}

func TestPromptOverrideCatalogIgnoresEmptyOverrides(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&mockPromptOverrideTool{name: "web_search", description: "search web"})

	catalog := NewPromptOverrideCatalog(registry, map[string]string{
		"web_search": "   ",
	})
	defs := catalog.ToolDefs()
	if len(defs) != 1 {
		t.Fatalf("unexpected tool defs count: %d", len(defs))
	}
	if defs[0].Description != "search web" {
		t.Fatalf("expected original description, got %q", defs[0].Description)
	}
}

type mockPromptOverrideTool struct {
	name        string
	description string
}

func (m *mockPromptOverrideTool) Name() string {
	return m.name
}

func (m *mockPromptOverrideTool) Description() string {
	return m.description
}

func (m *mockPromptOverrideTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}

func (m *mockPromptOverrideTool) Execute(_ context.Context, _ json.RawMessage, _ string) (string, error) {
	return "", nil
}
