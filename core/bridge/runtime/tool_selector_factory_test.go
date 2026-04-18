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
	if !strings.Contains(prompt, "Project root: /tmp/ghost-os-project") {
		t.Fatalf("expected prompt to include project root, got %q", prompt)
	}
}

func TestBuildSystemPromptForCatalogUsesOnlyScopedToolGuidance(t *testing.T) {
	tempDir := setupRuntimeFactoryTestEnv(t)
	promptsDir := filepath.Join(tempDir, "prompts")
	registry := tools.NewRegistry()
	for _, name := range []string{"ask_human", "read_and_summarize", "script_exec", "screen_action"} {
		registry.Register(&catalogMockTool{name: name})
	}

	prompt, err := buildSystemPromptForCatalog(
		Config{MaxTurns: 3, PromptsDir: promptsDir},
		tools.NewScopedCatalog(registry, []string{"ask_human", "read_and_summarize", "script_exec"}),
	)
	if err != nil {
		t.Fatalf("buildSystemPromptForCatalog returned error: %v", err)
	}
	for _, snippet := range []string{"`read_and_summarize`", "`script_exec`", "`ask_human`"} {
		if !strings.Contains(prompt, snippet) {
			t.Fatalf("expected prompt to contain %q, got %q", snippet, prompt)
		}
	}
	for _, snippet := range []string{"`screen_action`", "Available tools:", "Tool list:"} {
		if strings.Contains(prompt, snippet) {
			t.Fatalf("expected prompt to exclude %q, got %q", snippet, prompt)
		}
	}
}

func TestBuildSystemPromptForCatalogInjectsRSSGuidanceOnlyWhenVisible(t *testing.T) {
	tempDir := setupRuntimeFactoryTestEnv(t)
	promptsDir := filepath.Join(tempDir, "prompts")
	registry := tools.NewRegistry()
	for _, name := range []string{"ask_human", "feed_manage"} {
		registry.Register(&catalogMockTool{name: name})
	}

	withRSS, err := buildSystemPromptForCatalog(Config{MaxTurns: 3, PromptsDir: promptsDir}, registry)
	if err != nil {
		t.Fatalf("buildSystemPromptForCatalog returned error: %v", err)
	}
	if !strings.Contains(withRSS, "RSS inbox polling and AI filtering") {
		t.Fatalf("expected RSS guidance in prompt, got %q", withRSS)
	}

	withoutRSS, err := buildSystemPromptForCatalog(
		Config{MaxTurns: 3, PromptsDir: promptsDir},
		tools.NewScopedCatalog(registry, []string{"ask_human"}),
	)
	if err != nil {
		t.Fatalf("buildSystemPromptForCatalog returned error: %v", err)
	}
	if strings.Contains(withoutRSS, "RSS inbox polling and AI filtering") {
		t.Fatalf("expected RSS guidance to stay hidden, got %q", withoutRSS)
	}
	if strings.Contains(withoutRSS, "END_SESSION") {
		t.Fatalf("expected default system prompt to exclude END_SESSION protocol, got %q", withoutRSS)
	}
}

func TestBuildSystemPromptForCatalogIncludesDynamicToolStateSection(t *testing.T) {
	tempDir := setupRuntimeFactoryTestEnv(t)
	promptsDir := filepath.Join(tempDir, "prompts")
	registry := tools.NewRegistry()
	registry.Register(&catalogMockTool{name: "ask_human"})

	prompt, err := buildSystemPromptForCatalog(Config{MaxTurns: 3, PromptsDir: promptsDir}, registry)
	if err != nil {
		t.Fatalf("buildSystemPromptForCatalog returned error: %v", err)
	}
	for _, snippet := range []string{"## Dynamic Tool State", "No dynamic tools loaded"} {
		if !strings.Contains(prompt, snippet) {
			t.Fatalf("expected prompt to contain %q, got %q", snippet, prompt)
		}
	}
}

func TestBuildSystemPromptForSessionIncludesImmediateAndActiveDynamicTools(t *testing.T) {
	tempDir := setupRuntimeFactoryTestEnv(t)
	promptsDir := filepath.Join(tempDir, "prompts")
	registry := tools.NewRegistry()
	for _, name := range []string{"ask_human", "web_search"} {
		registry.Register(&catalogMockTool{name: name})
	}
	catalog := tools.NewScopedCatalog(registry, []string{"ask_human", "web_search"})

	sess := session.NewSession("")
	sess.AdvanceToolTurn(3)
	sess.EnsureDynamicToolLoaded("web_search", "tfind")

	immediate, err := buildSystemPromptForSession(
		Config{MaxTurns: 3, ToolSearch: ToolSearchConfig{IdleTurns: 3}, PromptsDir: promptsDir},
		catalog,
		sess,
		3,
	)
	if err != nil {
		t.Fatalf("buildSystemPromptForSession returned error: %v", err)
	}
	if !strings.Contains(immediate, "`web_search` was loaded in this user turn and is available now.") {
		t.Fatalf("expected immediate dynamic tool state, got %q", immediate)
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
	if !strings.Contains(active, "`web_search` is active in this session; remaining_idle_turns=3.") {
		t.Fatalf("expected active dynamic tool state, got %q", active)
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
