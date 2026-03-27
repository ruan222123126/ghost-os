package runtime

import (
	"context"
	"encoding/json"
	"testing"

	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

type catalogMockTool struct {
	name string
}

func (m *catalogMockTool) Name() string { return m.name }

func (m *catalogMockTool) Description() string { return "mock tool" }

func (m *catalogMockTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }

func (m *catalogMockTool) Execute(context.Context, json.RawMessage, string) (string, error) {
	return "", nil
}

func TestSessionTurnCatalog_ExposesLoadedToolsImmediately(t *testing.T) {
	registry := tools.NewRegistry()
	for _, name := range []string{"ask_human", "send_file", "web_search", "tfind"} {
		registry.Register(&catalogMockTool{name: name})
	}

	sess := session.NewSession("")
	sess.AdvanceToolTurn(3)
	sess.EnsureDynamicToolLoaded("web_search", "tfind")

	catalog := newSessionTurnCatalog(registry, []string{"ask_human", "send_file", "tfind"}, sess, 3, false)
	if catalog.Get("web_search") == nil {
		t.Fatal("expected loaded tool to become available immediately")
	}
}

func TestSessionTurnCatalog_HidesToolSearchFromSelector(t *testing.T) {
	registry := tools.NewRegistry()
	for _, name := range []string{"ask_human", "send_file", "web_search", "tfind"} {
		registry.Register(&catalogMockTool{name: name})
	}

	sess := session.NewSession("")
	sess.AdvanceToolTurn(3)
	sess.EnsureDynamicToolLoaded("web_search", "tfind")
	sess.AdvanceToolTurn(3)

	catalog := newSessionTurnCatalog(registry, []string{"ask_human", "send_file", "tfind"}, sess, 3, true)
	if catalog.Get("tfind") != nil {
		t.Fatal("expected selector catalog to exclude tfind")
	}
	if catalog.Get("web_search") == nil {
		t.Fatal("expected selector catalog to keep visible loaded tools")
	}
}
