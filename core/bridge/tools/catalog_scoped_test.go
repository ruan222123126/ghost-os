package tools

import (
	"context"
	"encoding/json"
	"testing"
)

type mockTool struct {
	name string
}

func (m *mockTool) Name() string { return m.name }

func (m *mockTool) Description() string { return "mock tool" }

func (m *mockTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }

func (m *mockTool) Execute(context.Context, json.RawMessage, string) (string, error) { return "", nil }

func TestScopedCatalog_OnlyAllowsListedTools(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&mockTool{name: "tool_a"})
	registry.Register(&mockTool{name: "tool_b"})
	registry.Register(&mockTool{name: "tool_c"})

	scoped := NewScopedCatalog(registry, []string{"tool_a", "tool_c"})
	if scoped.Get("tool_a") == nil {
		t.Fatal("should allow tool_a")
	}
	if scoped.Get("tool_b") != nil {
		t.Fatal("should block tool_b")
	}
	if scoped.Get("tool_c") == nil {
		t.Fatal("should allow tool_c")
	}

	defs := scoped.ToolDefs()
	if len(defs) != 2 {
		t.Fatalf("expected 2 tool defs, got %d", len(defs))
	}
	if defs[0].Name != "tool_a" || defs[1].Name != "tool_c" {
		t.Fatalf("unexpected tool defs order: %+v", defs)
	}
}
