package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ghost-os/bridge/session"
	"ghost-os/bridge/skills"
)

type toolSearchResponse struct {
	Action string `json:"action"`
	Kind   string `json:"kind"`
	Items  []struct {
		Kind               string `json:"kind"`
		Name               string `json:"name"`
		Summary            string `json:"summary"`
		Path               string `json:"path"`
		Status             string `json:"status"`
		AvailableNow       bool   `json:"available_now"`
		AvailableNextTurn  bool   `json:"available_next_turn"`
		RemainingIdleTurns int    `json:"remaining_idle_turns"`
		Dependencies       []struct {
			Kind   string `json:"kind"`
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"dependencies"`
	} `json:"items"`
	Errors []struct {
		Path  string `json:"path"`
		Error string `json:"error"`
	} `json:"errors"`
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
	if search.Kind != toolSearchKindTool {
		t.Fatalf("expected tool kind payload, got %+v", search)
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
	for _, name := range []string{"ask_human", "send_file", "screen_control", "web_search", ToolSearchToolName} {
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
		`{"action":"search","query":"desktop gui click and OCR tool; screen_control, visual automation, plus web_search internet retrieval"}`,
	)

	if !containsToolSearchItem(search.Items, "screen_control") {
		t.Fatalf("expected screen_control to match natural-language query, got %+v", search.Items)
	}
	if !containsToolSearchItem(search.Items, "web_search") {
		t.Fatalf("expected web_search to match natural-language query, got %+v", search.Items)
	}
}

func TestToolSearchTool_SkillKindSearchLoadListAndUnload(t *testing.T) {
	repoRoot := t.TempDir()
	writeSkillFile(
		t,
		filepath.Join(repoRoot, ".agents", "skills", "release"),
		"release_flow",
		"Release workflow",
		"Run release checklist before deploy.",
		"policy:\n  allow_implicit_invocation: true\ndependencies:\n  tools:\n    - web_search\n",
	)
	registry := NewRegistry()
	for _, name := range []string{"ask_human", "send_file", "script_exec", "web_search", ToolSearchToolName} {
		registry.Register(&mockTool{name: name})
	}
	tool := NewToolSearchTool(
		registry,
		VisibilityOptions{ToolSearchEnabled: true, Allowlist: []string{"send_file"}},
		3,
		ToolSearchOptions{
			ProjectRoot: repoRoot,
			SkillCatalog: skills.NewCatalogWithRoots([]string{
				filepath.Join(repoRoot, ".agents", "skills"),
			}),
		},
	)
	sess := session.NewSession("")
	sess.AdvanceToolTurn(3)
	ctx := WithSession(context.Background(), sess)

	search := decodeToolSearchResponse(t, tool, ctx, `{"action":"search","kind":"skill"}`)
	if search.Kind != toolSearchKindSkill || len(search.Items) != 1 {
		t.Fatalf("unexpected skill search result: %+v", search)
	}
	if search.Items[0].Kind != toolSearchKindSkill || search.Items[0].Name != "release_flow" {
		t.Fatalf("unexpected skill search item: %+v", search.Items[0])
	}

	load := decodeToolSearchResponse(t, tool, ctx, `{"action":"load","kind":"skill","tool_names":["release_flow"]}`)
	if len(load.Items) != 1 || load.Items[0].Status != "loaded" {
		t.Fatalf("unexpected skill load result: %+v", load.Items)
	}
	if len(load.Items[0].Dependencies) != 1 || load.Items[0].Dependencies[0].Name != "web_search" {
		t.Fatalf("expected dependency tool load result, got %+v", load.Items[0].Dependencies)
	}
	if got := sess.VisibleDynamicToolNames(3); len(got) != 1 || got[0] != "web_search" {
		t.Fatalf("expected dependency tool to be dynamically loaded, got %v", got)
	}

	list := decodeToolSearchResponse(t, tool, ctx, `{"action":"list","kind":"skill"}`)
	if len(list.Items) != 1 || list.Items[0].Status != "active" || list.Items[0].Name != "release_flow" {
		t.Fatalf("unexpected skill list result: %+v", list.Items)
	}

	unload := decodeToolSearchResponse(t, tool, ctx, `{"action":"unload","kind":"skill","tool_names":["release_flow"]}`)
	if len(unload.Items) != 1 || unload.Items[0].Status != "unloaded" {
		t.Fatalf("unexpected skill unload result: %+v", unload.Items)
	}
}

func TestToolSearchTool_SkillSearchIncludesDiscoveryErrors(t *testing.T) {
	repoRoot := t.TempDir()
	writeSkillFile(
		t,
		filepath.Join(repoRoot, ".agents", "skills", "broken"),
		"",
		"broken skill",
		"body",
		"",
	)
	registry := NewRegistry()
	for _, name := range []string{"ask_human", "send_file", ToolSearchToolName} {
		registry.Register(&mockTool{name: name})
	}
	tool := NewToolSearchTool(
		registry,
		VisibilityOptions{ToolSearchEnabled: true, Allowlist: []string{"send_file"}},
		3,
		ToolSearchOptions{
			ProjectRoot: repoRoot,
			SkillCatalog: skills.NewCatalogWithRoots([]string{
				filepath.Join(repoRoot, ".agents", "skills"),
			}),
		},
	)
	sess := session.NewSession("")
	sess.AdvanceToolTurn(1)
	ctx := WithSession(context.Background(), sess)

	search := decodeToolSearchResponse(t, tool, ctx, `{"action":"search","kind":"skill"}`)
	if len(search.Errors) == 0 {
		t.Fatalf("expected explicit discovery errors, got %+v", search)
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
	Kind               string `json:"kind"`
	Name               string `json:"name"`
	Summary            string `json:"summary"`
	Path               string `json:"path"`
	Status             string `json:"status"`
	AvailableNow       bool   `json:"available_now"`
	AvailableNextTurn  bool   `json:"available_next_turn"`
	RemainingIdleTurns int    `json:"remaining_idle_turns"`
	Dependencies       []struct {
		Kind   string `json:"kind"`
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"dependencies"`
}, want string) bool {
	for _, item := range items {
		if item.Name == want {
			return true
		}
	}
	return false
}

func writeSkillFile(
	t *testing.T,
	dir string,
	name string,
	description string,
	body string,
	openAIConfig string,
) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	content := "---\n"
	if name != "" {
		content += "name: " + name + "\n"
	}
	if description != "" {
		content += "description: " + description + "\n"
	}
	content += "---\n\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write skill file: %v", err)
	}
	if openAIConfig == "" {
		return
	}
	configPath := filepath.Join(dir, "agents", "openai.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir openai dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(openAIConfig), 0o644); err != nil {
		t.Fatalf("write openai config: %v", err)
	}
}
