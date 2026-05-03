package tools

import (
	"context"
	"encoding/base64"
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

func TestToolSearchTool_HidesBlockedSkillAndFallsBackToUserDuplicate(t *testing.T) {
	root := t.TempDir()
	repoRoot := filepath.Join(root, "repo")
	homeRoot := filepath.Join(root, "home")
	repoSkills := filepath.Join(repoRoot, ".agents", "skills")
	userSkills := filepath.Join(homeRoot, ".ghost-os", "skills")
	writeSkillFile(t, filepath.Join(repoSkills, "release"), "release_flow", "repo release", "repo body", "")
	writeSkillFile(t, filepath.Join(userSkills, "release"), "release_flow", "user release", "user body", "")
	t.Setenv("HOME", homeRoot)

	registry := NewRegistry()
	registry.Register(&mockTool{name: ToolSearchToolName})
	tool := NewToolSearchTool(
		registry,
		VisibilityOptions{ToolSearchEnabled: true},
		3,
		ToolSearchOptions{
			SkillConfig: skills.Config{
				ProjectRoot:    repoRoot,
				SkillBlocklist: []string{encodeToolSearchSkillID("repo", "release")},
			},
		},
	)

	sess := session.NewSession("")
	sess.AdvanceToolTurn(1)
	ctx := WithSession(context.Background(), sess)
	search := decodeToolSearchResponse(t, tool, ctx, `{"action":"search","query":"release"}`)
	if len(search.Items) != 1 {
		t.Fatalf("expected one visible fallback skill, got %+v", search.Items)
	}
	if search.Items[0].Summary != "user release" {
		t.Fatalf("expected user skill to become visible, got %+v", search.Items[0])
	}
}

func TestToolSearchTool_SearchLoadListAndUnload(t *testing.T) {
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
	for _, name := range []string{"ask_human", "codex_cli", "script_exec", "web_search", ToolSearchToolName} {
		registry.Register(&mockTool{name: name})
	}

	tool := NewToolSearchTool(
		registry,
		VisibilityOptions{ToolSearchEnabled: true, Allowlist: []string{"codex_cli"}},
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

	search := decodeToolSearchResponse(t, tool, ctx, `{"action":"search","query":"release deploy"}`)
	if len(search.Items) != 1 || search.Items[0].Name != "release_flow" {
		t.Fatalf("unexpected search items: %+v", search.Items)
	}
	if search.Kind != toolSearchKindSkill {
		t.Fatalf("expected skill kind payload, got %+v", search)
	}

	load := decodeToolSearchResponse(t, tool, ctx, `{"action":"load","skill_names":["release_flow"]}`)
	if len(load.Items) != 1 || load.Items[0].Status != "loaded" || !load.Items[0].AvailableNow || load.Items[0].AvailableNextTurn {
		t.Fatalf("unexpected load result: %+v", load.Items)
	}
	if got := sess.VisibleDynamicToolNames(3); len(got) != 0 {
		t.Fatalf("expected sfind to avoid dynamic tool loads, got %v", got)
	}

	activeNow := decodeToolSearchResponse(t, tool, ctx, `{"action":"list"}`)
	if len(activeNow.Items) != 1 || activeNow.Items[0].Status != "active" || !activeNow.Items[0].AvailableNow || activeNow.Items[0].Name != "release_flow" {
		t.Fatalf("unexpected active list in current turn: %+v", activeNow.Items)
	}

	sess.AdvanceToolTurn(3)
	active := decodeToolSearchResponse(t, tool, ctx, `{"action":"list"}`)
	if len(active.Items) != 1 || active.Items[0].Status != "active" || !active.Items[0].AvailableNow || active.Items[0].Name != "release_flow" {
		t.Fatalf("unexpected active list: %+v", active.Items)
	}

	afterLoadSearch := decodeToolSearchResponse(t, tool, ctx, `{"action":"search","query":"release"}`)
	if len(afterLoadSearch.Items) != 0 {
		t.Fatalf("unexpected search items after load: %+v", afterLoadSearch.Items)
	}

	unload := decodeToolSearchResponse(t, tool, ctx, `{"action":"unload","skill_names":["release_flow"]}`)
	if len(unload.Items) != 1 || unload.Items[0].Status != "unloaded" {
		t.Fatalf("unexpected unload result: %+v", unload.Items)
	}
}

func TestToolSearchTool_RejectsNonSkillKind(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&mockTool{name: ToolSearchToolName})
	tool := NewToolSearchTool(registry, VisibilityOptions{ToolSearchEnabled: true}, 3)

	sess := session.NewSession("")
	sess.AdvanceToolTurn(1)
	ctx := WithSession(context.Background(), sess)

	_, err := tool.Execute(ctx, json.RawMessage(`{"action":"search","kind":"tool"}`), "trace-test")
	if err == nil || err.Error() != "sfind supports skills only" {
		t.Fatalf("expected non-skill kind error, got %v", err)
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
	for _, name := range []string{"ask_human", "codex_cli", ToolSearchToolName} {
		registry.Register(&mockTool{name: name})
	}
	tool := NewToolSearchTool(
		registry,
		VisibilityOptions{ToolSearchEnabled: true, Allowlist: []string{"codex_cli"}},
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

	search := decodeToolSearchResponse(t, tool, ctx, `{"action":"search"}`)
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

func encodeToolSearchSkillID(source string, relativePath string) string {
	raw := source + "|" + relativePath
	return "skill_" + base64.RawURLEncoding.EncodeToString([]byte(raw))
}
