package tools

import (
	"context"
	"encoding/json"
	"testing"

	"ghost-os/bridge/session"
)

type toolSearchResponse struct {
	Action string `json:"action"`
	Items  []struct {
		Name               string `json:"name"`
		Status             string `json:"status"`
		AvailableNow       bool   `json:"available_now"`
		AvailableNextTurn  bool   `json:"available_next_turn"`
		RemainingIdleTurns int    `json:"remaining_idle_turns"`
	} `json:"items"`
}

func TestToolSearchTool_SearchLoadListAndUnload(t *testing.T) {
	registry := NewRegistry()
	for _, name := range []string{"ask_human", "send_file", "script_exec", "web_search", ToolSearchToolName} {
		registry.Register(&mockTool{name: name})
	}

	tool := NewToolSearchTool(registry, VisibilityOptions{
		ToolSearchEnabled: true,
		Allowlist:         []string{"send_file"},
	}, 3)

	sess := session.NewSession("")
	sess.AdvanceToolTurn(3)
	ctx := WithSession(context.Background(), sess)

	search := decodeToolSearchResponse(t, tool, ctx, `{"action":"search"}`)
	if len(search.Items) != 2 || search.Items[0].Name != "script_exec" || search.Items[1].Name != "web_search" {
		t.Fatalf("unexpected search items: %+v", search.Items)
	}

	load := decodeToolSearchResponse(t, tool, ctx, `{"action":"load","tool_names":["web_search"]}`)
	if len(load.Items) != 1 || load.Items[0].Status != "loaded" || !load.Items[0].AvailableNow || load.Items[0].AvailableNextTurn {
		t.Fatalf("unexpected load result: %+v", load.Items)
	}

	activeNow := decodeToolSearchResponse(t, tool, ctx, `{"action":"list"}`)
	if len(activeNow.Items) != 1 || activeNow.Items[0].Status != "active" || !activeNow.Items[0].AvailableNow {
		t.Fatalf("unexpected active list in current turn: %+v", activeNow.Items)
	}

	sess.AdvanceToolTurn(3)
	active := decodeToolSearchResponse(t, tool, ctx, `{"action":"list"}`)
	if len(active.Items) != 1 || active.Items[0].Status != "active" || !active.Items[0].AvailableNow {
		t.Fatalf("unexpected active list: %+v", active.Items)
	}

	afterLoadSearch := decodeToolSearchResponse(t, tool, ctx, `{"action":"search"}`)
	if len(afterLoadSearch.Items) != 1 || afterLoadSearch.Items[0].Name != "script_exec" {
		t.Fatalf("unexpected search items after load: %+v", afterLoadSearch.Items)
	}

	unload := decodeToolSearchResponse(t, tool, ctx, `{"action":"unload","tool_names":["web_search"]}`)
	if len(unload.Items) != 1 || unload.Items[0].Status != "unloaded" {
		t.Fatalf("unexpected unload result: %+v", unload.Items)
	}
}

func TestToolSearchTool_SearchMatchesNaturalLanguageQuery(t *testing.T) {
	registry := NewRegistry()
	for _, name := range []string{"ask_human", "send_file", "browser_control", "web_search", ToolSearchToolName} {
		registry.Register(&mockTool{name: name})
	}

	tool := NewToolSearchTool(registry, VisibilityOptions{
		ToolSearchEnabled: true,
		Allowlist:         []string{"send_file"},
	}, 3)

	sess := session.NewSession("")
	sess.AdvanceToolTurn(1)
	ctx := WithSession(context.Background(), sess)

	search := decodeToolSearchResponse(
		t,
		tool,
		ctx,
		`{"action":"search","query":"website search tool availability; web_search, browser_control, internet retrieval, web browser"}`,
	)

	if !containsToolSearchItem(search.Items, "browser_control") {
		t.Fatalf("expected browser_control to match natural-language query, got %+v", search.Items)
	}
	if !containsToolSearchItem(search.Items, "web_search") {
		t.Fatalf("expected web_search to match natural-language query, got %+v", search.Items)
	}
}

func decodeToolSearchResponse(t *testing.T, tool Tool, ctx context.Context, raw string) toolSearchResponse {
	t.Helper()

	output, err := tool.Execute(ctx, json.RawMessage(raw), "trace-test")
	if err != nil {
		t.Fatalf("tool search execute: %v", err)
	}
	var payload toolSearchResponse
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode tool search output: %v", err)
	}
	return payload
}

func containsToolSearchItem(items []struct {
	Name               string `json:"name"`
	Status             string `json:"status"`
	AvailableNow       bool   `json:"available_now"`
	AvailableNextTurn  bool   `json:"available_next_turn"`
	RemainingIdleTurns int    `json:"remaining_idle_turns"`
}, want string) bool {
	for _, item := range items {
		if item.Name == want {
			return true
		}
	}
	return false
}
