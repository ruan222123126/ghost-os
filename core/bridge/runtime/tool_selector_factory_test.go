package runtime

import (
	"strings"
	"testing"

	"ghost-os/bridge/tools"
)

func TestBuildSystemPromptForCatalogIncludesProjectRootRegression(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(&catalogMockTool{name: "ask_human"})
	registry.Register(&catalogMockTool{name: "script_exec"})

	prompt, err := buildSystemPromptForCatalog(Config{
		MaxTurns:    3,
		ProjectRoot: "/tmp/ghost-os-project",
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
	registry := tools.NewRegistry()
	for _, name := range []string{"ask_human", "script_exec", "screen_action"} {
		registry.Register(&catalogMockTool{name: name})
	}

	prompt, err := buildSystemPromptForCatalog(Config{MaxTurns: 3}, tools.NewScopedCatalog(registry, []string{"ask_human", "script_exec"}))
	if err != nil {
		t.Fatalf("buildSystemPromptForCatalog returned error: %v", err)
	}
	for _, snippet := range []string{"`script_exec`", "`ask_human`"} {
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
	registry := tools.NewRegistry()
	for _, name := range []string{"ask_human", "feed_manage"} {
		registry.Register(&catalogMockTool{name: name})
	}

	withRSS, err := buildSystemPromptForCatalog(Config{MaxTurns: 3}, registry)
	if err != nil {
		t.Fatalf("buildSystemPromptForCatalog returned error: %v", err)
	}
	if !strings.Contains(withRSS, "RSS inbox polling and AI filtering") {
		t.Fatalf("expected RSS guidance in prompt, got %q", withRSS)
	}

	withoutRSS, err := buildSystemPromptForCatalog(
		Config{MaxTurns: 3},
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
