package runtime

import (
	"path/filepath"
	"strings"
	"testing"

	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

func TestBuildSystemPromptForCatalogIncludesProjectRootRegression(t *testing.T) {
	tempDir := setupRuntimeFactoryTestEnv(t)
	promptsDir := filepath.Join(tempDir, "prompts")
	registry := tools.NewRegistry()
	registry.Register(&catalogMockTool{name: "ask_human"})
	registry.Register(&catalogMockTool{name: "script_exec"})

	prompt, err := buildSystemPromptForCatalog(Config{
		MaxTurns:    3,
		ProjectRoot: "/tmp/ghost-os-project",
		PromptsDir:  promptsDir,
	}, registry)
	if err != nil {
		t.Fatalf("buildSystemPromptForCatalog returned error: %v", err)
	}
	if strings.Contains(prompt, "{{project_root}}") {
		t.Fatalf("expected project_root placeholder to be resolved, got %q", prompt)
	}
	if !strings.Contains(prompt, "Context:") {
		t.Fatalf("expected prompt to include context section, got %q", prompt)
	}
	if !strings.Contains(prompt, "Context:\nOS:") {
		t.Fatalf("expected prompt to render context block, got %q", prompt)
	}
	if !strings.Contains(prompt, "Root: /tmp/ghost-os-project") {
		t.Fatalf("expected prompt to include project root, got %q", prompt)
	}
}

func TestBuildSystemPromptForCatalogOmitsToolGuidanceSection(t *testing.T) {
	tempDir := setupRuntimeFactoryTestEnv(t)
	promptsDir := filepath.Join(tempDir, "prompts")
	registry := tools.NewRegistry()
	for _, name := range []string{"ask_human", "script_exec", "screen_action"} {
		registry.Register(&catalogMockTool{name: name})
	}

	prompt, err := buildSystemPromptForCatalog(
		Config{MaxTurns: 3, PromptsDir: promptsDir},
		tools.NewScopedCatalog(registry, []string{"ask_human", "script_exec"}),
	)
	if err != nil {
		t.Fatalf("buildSystemPromptForCatalog returned error: %v", err)
	}
	for _, snippet := range []string{"## Tool Guidance", "`script_exec`", "`ask_human`", "`screen_action`", "Available tools:", "Tool list:"} {
		if strings.Contains(prompt, snippet) {
			t.Fatalf("expected prompt to exclude %q, got %q", snippet, prompt)
		}
	}
}

func TestBuildSystemPromptForCatalogUsesFlatTemplateStructure(t *testing.T) {
	tempDir := setupRuntimeFactoryTestEnv(t)
	promptsDir := filepath.Join(tempDir, "prompts")
	registry := tools.NewRegistry()
	registry.Register(&catalogMockTool{name: "ask_human"})

	prompt, err := buildSystemPromptForCatalog(Config{MaxTurns: 3, PromptsDir: promptsDir}, registry)
	if err != nil {
		t.Fatalf("buildSystemPromptForCatalog returned error: %v", err)
	}
	for _, snippet := range []string{"Role:", "Job:", "Context:"} {
		if !strings.Contains(prompt, snippet) {
			t.Fatalf("expected prompt to contain %q, got %q", snippet, prompt)
		}
	}
	for _, snippet := range []string{"Skills:", "Skill Context:"} {
		if !strings.Contains(prompt, snippet) {
			t.Fatalf("expected prompt to include %q, got %q", snippet, prompt)
		}
	}
	for _, snippet := range []string{"## Dynamic Tool State", "## Dynamic Skill Context"} {
		if strings.Contains(prompt, snippet) {
			t.Fatalf("expected prompt to exclude %q, got %q", snippet, prompt)
		}
	}
}

func TestBuildSystemPromptForSessionOmitsDynamicToolStateFromFlatTemplate(t *testing.T) {
	tempDir := setupRuntimeFactoryTestEnv(t)
	promptsDir := filepath.Join(tempDir, "prompts")
	registry := tools.NewRegistry()
	for _, name := range []string{"ask_human", "web_search"} {
		registry.Register(&catalogMockTool{name: name})
	}
	catalog := tools.NewScopedCatalog(registry, []string{"ask_human", "web_search"})

	sess := session.NewSession("")
	sess.AdvanceToolTurn(3)
	sess.EnsureDynamicToolLoaded("web_search", "sfind")

	immediate, err := buildSystemPromptForSession(
		Config{MaxTurns: 3, ToolSearch: ToolSearchConfig{IdleTurns: 3}, PromptsDir: promptsDir},
		catalog,
		sess,
		3,
	)
	if err != nil {
		t.Fatalf("buildSystemPromptForSession returned error: %v", err)
	}
	for _, snippet := range []string{"Role:", "Job:", "Context:"} {
		if !strings.Contains(immediate, snippet) {
			t.Fatalf("expected immediate prompt to contain %q, got %q", snippet, immediate)
		}
	}
	if strings.Contains(immediate, "`web_search` was loaded in this user turn and is available now.") {
		t.Fatalf("expected immediate prompt to omit dynamic tool state, got %q", immediate)
	}

	sess.AdvanceToolTurn(3)
	active, err := buildSystemPromptForSession(
		Config{MaxTurns: 3, ToolSearch: ToolSearchConfig{IdleTurns: 3}, PromptsDir: promptsDir},
		catalog,
		sess,
		3,
	)
	if err != nil {
		t.Fatalf("buildSystemPromptForSession returned error: %v", err)
	}
	if strings.Contains(active, "`web_search` is active in this session; remaining_idle_turns=3.") {
		t.Fatalf("expected active prompt to omit dynamic tool state, got %q", active)
	}
}

func TestBuildSystemPromptForCatalogFailsWhenPromptConfigMissing(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(&catalogMockTool{name: "ask_human"})

	_, err := buildSystemPromptForCatalog(Config{
		MaxTurns:    3,
		PromptsPath: filepath.Join(t.TempDir(), "missing-prompts.yaml"),
	}, registry)
	if err == nil {
		t.Fatal("expected prompt load error, got nil")
	}
	if !strings.Contains(err.Error(), "load prompt manager") {
		t.Fatalf("unexpected error: %v", err)
	}
}
